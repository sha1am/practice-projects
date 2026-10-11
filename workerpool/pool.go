package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// Config controls the worker pool and per-connection limits.
// Zero values are replaced by safe defaults in NewServer.
type Config struct {
	Workers          int           // goroutines handling requests (default: max(32, 4*NumCPU))
	QueueSize        int           // accepted-but-unserved connections buffered (default: 4*Workers)
	ReadTimeout      time.Duration // budget for reading the ENTIRE request, head + body (default 10s)
	WriteTimeout     time.Duration // budget for writing the response (default 10s)
	HandlerTimeout   time.Duration // deadline put on the request context (default 15s)
	MaxHeaderBytes   int64         // request line + headers limit (default 64 KiB)
	MaxBodyBytes     int64         // request body limit (default 4 MiB)
	MaxResponseBytes int64         // buffered response limit (default 8 MiB)
}

func (c *Config) applyDefaults() {
	if c.Workers <= 0 {
		c.Workers = max(32, 4*runtime.NumCPU())
	}
	if c.QueueSize <= 0 {
		c.QueueSize = 4 * c.Workers
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 10 * time.Second
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 10 * time.Second
	}
	if c.HandlerTimeout <= 0 {
		c.HandlerTimeout = 15 * time.Second
	}
	if c.MaxHeaderBytes <= 0 {
		c.MaxHeaderBytes = 64 << 10
	}
	if c.MaxBodyBytes <= 0 {
		c.MaxBodyBytes = 4 << 20
	}
	if c.MaxResponseBytes <= 0 {
		c.MaxResponseBytes = 8 << 20
	}
}

// Stats is a point-in-time snapshot of server counters.
type Stats struct {
	Accepted uint64 // connections handed to the pool
	Rejected uint64 // connections turned away with 503 because the queue was full
	Served   uint64 // responses successfully written
	Panics   uint64 // recovered panics
}

type counters struct {
	accepted, rejected, served, panics atomic.Uint64
}

// Server is an HTTP/1.1 server in which one accept loop (the "main goroutine")
// hands connections over a bounded channel (the "work queue") to a fixed pool
// of worker goroutines. Each connection carries exactly one request
// (Connection: close), so a worker is never pinned by an idle keep-alive client.
type Server struct {
	cfg     Config
	handler http.Handler
	log     *slog.Logger

	queue chan net.Conn

	baseCtx    context.Context // parent of every request context; cancelled on forced shutdown
	cancelBase context.CancelFunc

	quit     chan struct{} // closed when Shutdown begins
	quitOnce sync.Once
	done     chan struct{} // closed once Serve has fully drained

	mu     sync.Mutex
	ln     net.Listener
	active map[net.Conn]struct{} // queued or in-flight connections

	stats counters
}

// NewServer builds a Server. handler must be safe for concurrent use.
func NewServer(cfg Config, handler http.Handler, log *slog.Logger) *Server {
	cfg.applyDefaults()
	if log == nil {
		log = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		cfg:        cfg,
		handler:    handler,
		log:        log,
		queue:      make(chan net.Conn, cfg.QueueSize),
		baseCtx:    ctx,
		cancelBase: cancel,
		quit:       make(chan struct{}),
		done:       make(chan struct{}),
		active:     make(map[net.Conn]struct{}),
	}
}

// Stats returns a snapshot of the server counters.
func (s *Server) Stats() Stats {
	return Stats{
		Accepted: s.stats.accepted.Load(),
		Rejected: s.stats.rejected.Load(),
		Served:   s.stats.served.Load(),
		Panics:   s.stats.panics.Load(),
	}
}

// Serve starts the worker pool and runs the accept loop on ln. It blocks until
// the server has shut down and every queued connection has been drained, then
// returns http.ErrServerClosed. Any other error is a fatal listener failure.
// Serve may be called once.
func (s *Server) Serve(ln net.Listener) error {
	s.mu.Lock()
	select {
	case <-s.quit:
		s.mu.Unlock()
		_ = ln.Close()
		return http.ErrServerClosed
	default:
	}
	if s.ln != nil {
		s.mu.Unlock()
		return errors.New("workerpool: Serve called twice")
	}
	s.ln = ln
	s.mu.Unlock()

	var workers sync.WaitGroup
	for i := 0; i < s.cfg.Workers; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for conn := range s.queue { // exits once queue is closed AND drained
				s.serveConn(conn)
			}
		}()
	}

	err := s.acceptLoop(ln)

	close(s.queue) // let workers finish what is already queued, then exit
	workers.Wait()
	close(s.done)
	return err
}

func (s *Server) acceptLoop(ln net.Listener) error {
	var backoff time.Duration
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return http.ErrServerClosed
			default:
			}
			if errors.Is(err, net.ErrClosed) {
				return err
			}
			// Likely transient (EMFILE, ECONNABORTED, ...): back off and retry.
			if backoff == 0 {
				backoff = 5 * time.Millisecond
			} else {
				backoff = min(backoff*2, time.Second)
			}
			s.log.Error("accept failed; retrying", "err", err, "backoff", backoff)
			select {
			case <-time.After(backoff):
				continue
			case <-s.quit:
				return http.ErrServerClosed
			}
		}
		backoff = 0

		s.track(conn)
		select {
		case s.queue <- conn: // step 2: connection sent over the channel
			s.stats.accepted.Add(1)
		default: // queue full: shed load immediately instead of piling up
			s.stats.rejected.Add(1)
			s.reject(conn)
		}
	}
}

// Shutdown stops accepting new connections, lets queued and in-flight requests
// finish, and returns nil once everything has drained. If ctx expires first,
// it cancels request contexts, force-closes all connections, and returns
// ctx.Err(). Handlers that ignore their context may still be running then.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.quitOnce.Do(func() { close(s.quit) })
	ln := s.ln
	s.mu.Unlock()

	if ln == nil {
		return nil // Serve never started (and will refuse to when it does)
	}
	_ = ln.Close()

	select {
	case <-s.done:
		s.cancelBase()
		return nil
	case <-ctx.Done():
		s.cancelBase()
		s.closeAll()
		return ctx.Err()
	}
}

// ---- connection handling ----------------------------------------------------

func (s *Server) track(c net.Conn) {
	s.mu.Lock()
	s.active[c] = struct{}{}
	s.mu.Unlock()
}

func (s *Server) finish(c net.Conn) {
	_ = c.Close()
	s.mu.Lock()
	delete(s.active, c)
	s.mu.Unlock()
}

func (s *Server) closeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.active {
		_ = c.Close()
	}
}

// reject sheds load without occupying a worker. The response is tiny, so the
// short deadline only matters for a pathological client and keeps the accept
// loop from stalling.
func (s *Server) reject(c net.Conn) {
	_ = c.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
	_, _ = io.WriteString(c, "HTTP/1.1 503 Service Unavailable\r\n"+
		"Retry-After: 1\r\nConnection: close\r\nContent-Length: 0\r\n\r\n")
	s.finish(c)
}

// serveConn runs on a worker goroutine: steps 3-5 of the diagram.
func (s *Server) serveConn(conn net.Conn) {
	defer s.finish(conn)
	defer func() { // safety net for anything outside the handler call
		if r := recover(); r != nil {
			s.stats.panics.Add(1)
			s.log.Error("panic serving connection", "panic", r, "stack", string(debug.Stack()))
		}
	}()

	// One deadline bounds the whole request read (slow-loris protection).
	_ = conn.SetReadDeadline(time.Now().Add(s.cfg.ReadTimeout))

	// Cap header size by limiting raw bytes read until the head is parsed.
	lr := &io.LimitedReader{R: conn, N: s.cfg.MaxHeaderBytes}
	req, err := http.ReadRequest(bufio.NewReader(lr))
	if err != nil {
		switch {
		case lr.N <= 0:
			s.writeError(conn, http.StatusRequestHeaderFieldsTooLarge)
		case errors.Is(err, io.EOF), errors.Is(err, net.ErrClosed):
			// client left, or forced shutdown: nothing to say
		case isTimeout(err):
			s.writeError(conn, http.StatusRequestTimeout)
		default:
			s.writeError(conn, http.StatusBadRequest)
		}
		return
	}
	lr.N = math.MaxInt64 // head is parsed; the body is bounded by MaxBytesReader below
	req.RemoteAddr = conn.RemoteAddr().String()

	ctx, cancel := context.WithTimeout(s.baseCtx, s.cfg.HandlerTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	rw := newBufferedResponse(s.cfg.MaxResponseBytes)
	req.Body = http.MaxBytesReader(rw, req.Body, s.cfg.MaxBodyBytes)

	if !s.callHandler(rw, req) || rw.overflowed {
		rw = newErrorResponse(http.StatusInternalServerError)
	}

	_ = conn.SetWriteDeadline(time.Now().Add(s.cfg.WriteTimeout))
	if err := rw.writeTo(conn, req); err != nil {
		s.log.Debug("write response failed", "err", err, "remote", req.RemoteAddr)
		return
	}
	s.stats.served.Add(1)

	// Drain a bounded amount of unread body so closing the socket doesn't RST
	// the client before it has read our response.
	_, _ = io.CopyN(io.Discard, req.Body, 256<<10)
}

// callHandler invokes the user handler and converts a panic into ok=false.
func (s *Server) callHandler(w http.ResponseWriter, r *http.Request) (ok bool) {
	defer func() {
		if rec := recover(); rec != nil {
			s.stats.panics.Add(1)
			s.log.Error("handler panic",
				"method", r.Method, "path", r.URL.Path, "panic", rec, "stack", string(debug.Stack()))
			ok = false
		}
	}()
	s.handler.ServeHTTP(w, r)
	return true
}

func (s *Server) writeError(c net.Conn, code int) {
	_ = c.SetWriteDeadline(time.Now().Add(s.cfg.WriteTimeout))
	_ = newErrorResponse(code).writeTo(c, nil)
}

func isTimeout(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// ---- minimal buffered http.ResponseWriter -----------------------------------

var errResponseTooLarge = errors.New("workerpool: response exceeds MaxResponseBytes")

// bufferedResponse collects a handler's output so the status line and
// Content-Length can be written correctly in one shot. Trade-off: no streaming.
type bufferedResponse struct {
	header     http.Header
	status     int
	wroteHdr   bool
	body       bytes.Buffer
	max        int64
	overflowed bool
}

func newBufferedResponse(max int64) *bufferedResponse {
	return &bufferedResponse{header: make(http.Header), max: max}
}

func newErrorResponse(code int) *bufferedResponse {
	b := newBufferedResponse(1 << 10)
	b.header.Set("Content-Type", "text/plain; charset=utf-8")
	b.WriteHeader(code)
	_, _ = b.Write([]byte(http.StatusText(code) + "\n"))
	return b
}

func (b *bufferedResponse) Header() http.Header { return b.header }

func (b *bufferedResponse) WriteHeader(code int) {
	if b.wroteHdr {
		return
	}
	b.status, b.wroteHdr = code, true
}

func (b *bufferedResponse) Write(p []byte) (int, error) {
	b.WriteHeader(http.StatusOK)
	if int64(b.body.Len())+int64(len(p)) > b.max {
		b.overflowed = true
		return 0, errResponseTooLarge
	}
	return b.body.Write(p)
}

func (b *bufferedResponse) writeTo(w io.Writer, req *http.Request) error {
	b.WriteHeader(http.StatusOK)
	h := b.header.Clone()
	if h.Get("Date") == "" {
		h.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	}
	if h.Get("Content-Type") == "" && b.body.Len() > 0 {
		h.Set("Content-Type", http.DetectContentType(b.body.Bytes()))
	}
	resp := &http.Response{
		StatusCode:    b.status,
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        h,
		Body:          io.NopCloser(bytes.NewReader(b.body.Bytes())),
		ContentLength: int64(b.body.Len()),
		Close:         true, // emits "Connection: close"
		Request:       req,  // lets Response.Write omit the body for HEAD
	}
	return resp.Write(w)
}

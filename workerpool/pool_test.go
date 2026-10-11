package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func startServer(t *testing.T, cfg Config, h http.Handler) (*Server, string, <-chan error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := NewServer(cfg, h, slog.New(slog.NewTextHandler(io.Discard, nil)))
	errCh := make(chan error, 1)
	go func() { errCh <- s.Serve(ln) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})
	return s, ln.Addr().String(), errCh
}

func rawGet(t *testing.T, addr string) (net.Conn, *bufio.Reader) {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	fmt.Fprint(c, "GET / HTTP/1.1\r\nHost: test\r\n\r\n")
	return c, bufio.NewReader(c)
}

func readStatus(t *testing.T, br *bufio.Reader) int {
	t.Helper()
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

func TestServesRequests(t *testing.T) {
	_, addr, _ := startServer(t, Config{Workers: 2}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hi %s", r.URL.Query().Get("n"))
	}))
	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}, Timeout: 5 * time.Second}
	resp, err := client.Get("http://" + addr + "/?n=bob")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(body) != "hi bob" {
		t.Fatalf("got %d %q", resp.StatusCode, body)
	}
}

func TestRejectsWhenSaturated(t *testing.T) {
	started := make(chan struct{}, 4)
	release := make(chan struct{})
	s, addr, _ := startServer(t, Config{Workers: 1, QueueSize: 1}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-release
	}))

	_, br1 := rawGet(t, addr) // occupies the only worker
	<-started
	_, br2 := rawGet(t, addr) // sits in the queue
	// Wait until the accept loop has queued conn #2 so conn #3 sees a full queue.
	deadline := time.Now().Add(2 * time.Second)
	for s.Stats().Accepted < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	_, br3 := rawGet(t, addr) // queue full -> shed

	if got := readStatus(t, br3); got != http.StatusServiceUnavailable {
		t.Fatalf("third connection: got %d, want 503", got)
	}
	close(release)
	if got := readStatus(t, br1); got != 200 {
		t.Fatalf("first: got %d", got)
	}
	if got := readStatus(t, br2); got != 200 {
		t.Fatalf("second: got %d", got)
	}
	if r := s.Stats().Rejected; r != 1 {
		t.Fatalf("rejected = %d, want 1", r)
	}
}

func TestGracefulShutdownDrains(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	s, addr, serveErr := startServer(t, Config{Workers: 1}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-release
		fmt.Fprint(w, "finished")
	}))

	_, br := rawGet(t, addr)
	<-started

	shutdownErr := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownErr <- s.Shutdown(ctx)
	}()

	select {
	case err := <-shutdownErr:
		t.Fatalf("Shutdown returned before in-flight request finished: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	if got := readStatus(t, br); got != 200 {
		t.Fatalf("in-flight request got %d", got)
	}
	if err := <-shutdownErr; err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Serve returned %v, want ErrServerClosed", err)
	}
	if _, err := net.DialTimeout("tcp", addr, 200*time.Millisecond); err == nil {
		t.Fatal("listener still accepting after shutdown")
	}
}

func TestShutdownForcesAfterTimeout(t *testing.T) {
	started := make(chan struct{}, 1)
	s, addr, _ := startServer(t, Config{Workers: 1}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-r.Context().Done() // cooperative handler: unblocks when the pool cancels contexts
	}))
	rawGet(t, addr)
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := s.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown = %v, want DeadlineExceeded", err)
	}
}

func TestPanicBecomes500AndWorkerSurvives(t *testing.T) {
	s, addr, _ := startServer(t, Config{Workers: 1}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/boom" {
			panic("boom")
		}
		fmt.Fprint(w, "ok")
	}))

	_, br := rawPath(t, addr, "/boom")
	if got := readStatus(t, br); got != 500 {
		t.Fatalf("panic request: got %d, want 500", got)
	}
	_, br = rawPath(t, addr, "/fine")
	if got := readStatus(t, br); got != 200 {
		t.Fatalf("follow-up: got %d, want 200 (worker should survive)", got)
	}
	if p := s.Stats().Panics; p != 1 {
		t.Fatalf("panics = %d, want 1", p)
	}
}

func rawPath(t *testing.T, addr, path string) (net.Conn, *bufio.Reader) {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	fmt.Fprintf(c, "GET %s HTTP/1.1\r\nHost: test\r\n\r\n", path)
	return c, bufio.NewReader(c)
}

func TestLimits(t *testing.T) {
	_, addr, _ := startServer(t, Config{Workers: 2, MaxHeaderBytes: 1024, MaxBodyBytes: 16}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			http.Error(w, "too big", http.StatusRequestEntityTooLarge)
			return
		}
		fmt.Fprint(w, "ok")
	}))

	t.Run("header too large", func(t *testing.T) {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		_ = c.SetDeadline(time.Now().Add(5 * time.Second))
		fmt.Fprintf(c, "GET / HTTP/1.1\r\nHost: t\r\nX-Big: %s\r\n\r\n", strings.Repeat("a", 4096))
		if got := readStatus(t, bufio.NewReader(c)); got != http.StatusRequestHeaderFieldsTooLarge {
			t.Fatalf("got %d, want 431", got)
		}
	})

	t.Run("body too large", func(t *testing.T) {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		_ = c.SetDeadline(time.Now().Add(5 * time.Second))
		body := strings.Repeat("x", 100)
		fmt.Fprintf(c, "POST / HTTP/1.1\r\nHost: t\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
		if got := readStatus(t, bufio.NewReader(c)); got != http.StatusRequestEntityTooLarge {
			t.Fatalf("got %d, want 413", got)
		}
	})

	t.Run("garbage request", func(t *testing.T) {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		_ = c.SetDeadline(time.Now().Add(5 * time.Second))
		fmt.Fprint(c, "NOT HTTP\r\n\r\n")
		if got := readStatus(t, bufio.NewReader(c)); got != http.StatusBadRequest {
			t.Fatalf("got %d, want 400", got)
		}
	})
}

func TestSlowClientTimesOut(t *testing.T) {
	_, addr, _ := startServer(t, Config{Workers: 1, ReadTimeout: 150 * time.Millisecond}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	fmt.Fprint(c, "GET / HTTP/1.1\r\nHost: t\r\n") // never finishes the head
	if got := readStatus(t, bufio.NewReader(c)); got != http.StatusRequestTimeout {
		t.Fatalf("got %d, want 408", got)
	}
}

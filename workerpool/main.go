package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		addr            = flag.String("addr", ":8080", "listen address")
		shutdownTimeout = flag.Duration("shutdown-timeout", 20*time.Second, "max time to drain on SIGINT/SIGTERM")
		cfg             Config
	)
	flag.IntVar(&cfg.Workers, "workers", 0, "worker goroutines (0 = auto)")
	flag.IntVar(&cfg.QueueSize, "queue", 0, "pending-connection queue size (0 = 4x workers)")
	flag.DurationVar(&cfg.ReadTimeout, "read-timeout", 0, "whole-request read timeout (0 = 10s)")
	flag.DurationVar(&cfg.WriteTimeout, "write-timeout", 0, "response write timeout (0 = 10s)")
	flag.DurationVar(&cfg.HandlerTimeout, "handler-timeout", 0, "request context deadline (0 = 15s)")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello, %s\n", r.URL.Query().Get("name"))
	})
	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
			fmt.Fprintln(w, "done")
		case <-r.Context().Done(): // honour timeouts and shutdown
			http.Error(w, "cancelled", http.StatusServiceUnavailable)
		}
	})

	srv := NewServer(cfg, mux, log)

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}
	log.Info("listening", "addr", ln.Addr().String(), "workers", srv.cfg.Workers, "queue", srv.cfg.QueueSize)

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr: // listener died on its own
		return err
	case <-sigCtx.Done():
	}
	stop() // a second signal now kills the process the default way

	log.Info("shutting down", "timeout", shutdownTimeout.String())
	ctx, cancel := context.WithTimeout(context.Background(), *shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced shutdown: %w", err)
	}
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("stopped cleanly", "stats", fmt.Sprintf("%+v", srv.Stats()))
	return nil
}

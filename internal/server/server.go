package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/tsisar/extended-log-go/log"
)

const shutdownTimeout = 10 * time.Second

type Server struct {
	http *http.Server
}

func New(addr string, handler http.Handler) *Server {
	return &Server{
		http: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Run starts the HTTP server and blocks until ctx is cancelled.
// On cancellation it performs a graceful shutdown.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		log.Infof("starting HTTP server: addr=%s", s.http.Addr)
		if err := s.http.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting down HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return s.http.Shutdown(shutdownCtx)
	}
}

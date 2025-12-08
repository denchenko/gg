package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/denchenko/gg/internal/core/app"
)

const (
	readTimeout  = 10 * time.Second
	writeTimeout = 10 * time.Second
	idleTimeout  = 120 * time.Second
)

// Server represents an HTTP server.
type Server struct {
	server *http.Server
	app    *app.App
}

// NewServer creates a new HTTP server.
func NewServer(addr string, appInstance *app.App) *Server {
	mux := http.NewServeMux()

	s := &Server{
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
		app: appInstance,
	}

	mux.HandleFunc("/gitlab/hook", s.handleGitLabWebhook)

	return s
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	log.Printf("Starting server on %s", s.server.Addr)

	if err := s.server.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	return nil
}

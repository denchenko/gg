package http

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/denchenko/gg/internal/core/app"
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
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		app: appInstance,
	}

	mux.HandleFunc("/gitlab/hook", s.handleGitLabWebhook)

	return s
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	log.Printf("Starting server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

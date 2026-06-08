package api

import (
	"context"
	"net/http"
)

// APIServer manages HTTP routes and coordinates tasks for the web client interface.
type APIServer struct {
	addr   string
	server *http.Server
}

// NewAPIServer creates a new API server on the specified address.
func NewAPIServer(addr string, dbPath string) *APIServer {
	return &APIServer{
		addr: addr,
	}
}

// RegisterRoutes registers endpoints for dashboard metrics, query execution, files, and config.
func (s *APIServer) RegisterRoutes() {
	// TODO: Handle endpoints: /api/stats, /api/query, /api/config, /api/run-pipeline, etc.
	// Also serve embedded frontend files (index.html, styles, app.js, docs.html)
}

// Start starts the HTTP server asynchronously.
func (s *APIServer) Start() error {
	// TODO: ListenAndServe
	return nil
}

// Stop gracefully shuts down the server.
func (s *APIServer) Stop(ctx context.Context) error {
	// TODO: Shutdown server
	return nil
}

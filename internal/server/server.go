package server

import (
	_ "embed"
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aakash1408/shortener/internal/config"
	"github.com/aakash1408/shortener/internal/store"
)

//go:embed static/index.html
var indexHTML []byte

type server struct {
	httpServer *http.Server
	store      store.Store
	cfg        config.Config
	logger     *slog.Logger
}

func New(cfg config.Config, st store.Store, logger *slog.Logger) *server {
	s := &server{
		store:  st,
		cfg:    cfg,
		logger: logger,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	return s
}

func (s *server) registerRoutes(mux *http.ServeMux) {
	// homepage
	mux.HandleFunc("GET /", s.handleIndex)

	// public — no auth needed
	mux.HandleFunc("POST /api/register", s.handleRegister)
	mux.HandleFunc("POST /api/login", s.handleLogin)
	mux.HandleFunc("GET /{shortCode}", s.handleRedirect)

	// protected — auth required
	mux.Handle("POST /api/urls", s.authMiddleware(http.HandlerFunc(s.handleCreateURL)))
	mux.Handle("GET /api/urls", s.authMiddleware(http.HandlerFunc(s.handleListURLs)))
	mux.Handle("DELETE /api/urls/{shortCode}", s.authMiddleware(http.HandlerFunc(s.handleDeleteURL)))
	mux.Handle("PATCH /api/urls/{shortCode}", s.authMiddleware(http.HandlerFunc(s.handleUpdateURL)))
	mux.Handle("GET /api/urls/{shortCode}/clicks", s.authMiddleware(http.HandlerFunc(s.handleGetClicks)))

}

func (s *server) Start() error {
	s.logger.Info("server starting", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func (s *server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

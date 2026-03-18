package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/driif/go-vibe-starter/internal/server/config"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	server *http.Server
	Router chi.Router
	Config config.App
}

func NewWithConfig(config config.App) *Server {
	return &Server{
		Router: chi.NewRouter(),
		Config: config,
	}
}

func (s *Server) Initialize() error {
	if s.server != nil {
		return nil
	}

	addr := s.Config.Service.Port
	if addr == "" {
		addr = ":9880"
	}
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	s.server = &http.Server{
		Addr:              addr,
		Handler:           s.Router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return nil
}

func (s *Server) Start() error {
	if err := s.Initialize(); err != nil {
		return err
	}
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

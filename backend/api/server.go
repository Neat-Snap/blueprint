package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Neat-Snap/blueprint-backend/config"
)

type Server struct {
	cfg    config.Config
	log    *slog.Logger
	serv   *http.Server
	router http.Handler
}

func NewServer(cfg config.Config, log *slog.Logger, router http.Handler) *Server {
	s := &Server{
		cfg:    cfg,
		log:    log,
		router: router,
	}
	s.serv = &http.Server{
		Addr:         cfg.Addr,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.ReadTimeoutS) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeoutS) * time.Second,
		IdleTimeout:  time.Duration(cfg.IdleTimeoutS) * time.Second,
	}
	return s
}

func (s *Server) Start() error {
	s.log.Info("starting http server", "addr", s.cfg.Addr, "env", s.cfg.Env)
	err := s.serv.ListenAndServe()
	return err
}

func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("stopping server")
	err := s.serv.Shutdown(ctx)
	return err
}

func (s *Server) String() string {
	return fmt.Sprintf("Server(%s)", s.cfg.Addr)
}

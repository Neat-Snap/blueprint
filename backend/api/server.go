package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/logger"
)

type Server struct {
	cfg    config.Config
	log    *logger.MultiLogger
	serv   *http.Server
	router http.Handler
}

func NewServer(cfg config.Config, log *logger.MultiLogger, router http.Handler) *Server {
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
	addr := s.cfg.Addr
	if strings.HasPrefix(addr, "http") {
		if u, err := url.Parse(addr); err == nil && u.Host != "" {
			addr = u.Host
		}
	}
	s.serv.Addr = addr
	s.log.Info("starting http server", "addr", addr, "env", s.cfg.Env)
	return s.serv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("stopping server")
	err := s.serv.Shutdown(ctx)
	return err
}

func (s *Server) String() string {
	return fmt.Sprintf("Server(%s)", s.cfg.Addr)
}

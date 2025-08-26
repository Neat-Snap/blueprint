package api

import (
	"github.com/Neat-Snap/blueprint-backend/api/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type RouterConfig struct {
	Env string
}

func NewRouter(c RouterConfig) chi.Router {
	r := chi.NewRouter()
	if c.Env != "prod" {
		r.Use(middleware.Logger)
	}
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	r.Get("/health", handlers.HealthHandler)
	return r
}

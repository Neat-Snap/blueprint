package api

import (
	"github.com/Neat-Snap/blueprint-backend/api/handlers"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"
)

type RouterConfig struct {
	Env    string
	DB     *gorm.DB
	Logger logger.MultiLogger
}

func NewRouter(c RouterConfig) chi.Router {
	r := chi.NewRouter()
	if c.Env != "prod" {
		r.Use(middleware.Logger)
	}
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// setting up health route (with struct and complexity to showcase the approach)
	api := handlers.NewTestHealthAPI(c.DB, c.Logger)
	r.Get("/health", api.HealthHandler)

	return r
}

package api

import (
	"time"

	"github.com/Neat-Snap/blueprint-backend/api/handlers"
	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/Neat-Snap/blueprint-backend/utils/email"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"gorm.io/gorm"
)

type RouterConfig struct {
	Env         string
	DB          *gorm.DB
	Logger      logger.MultiLogger
	Connection  *db.Connection
	EmailClient *email.EmailClient
	RedisSecret string
}

func NewRouter(c RouterConfig) chi.Router {
	r := chi.NewRouter()
	if c.Env != "prod" {
		r.Use(middleware.Logger)
	}
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(httprate.Limit(
		10,
		5*time.Second,
		httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint),
	))

	// setting up health route (with struct and complexity to showcase the approach)
	api := handlers.NewTestHealthAPI(c.DB, c.Logger)
	r.Get("/health", api.HealthHandler)

	authAPI := handlers.NewAuthAPI(c.DB, c.Logger, c.Connection, c.EmailClient, c.RedisSecret)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authAPI.RegisterEndpoint)
		r.Post("/confirm-email", authAPI.ConfirmEmailEndpoint)
	})

	return r
}

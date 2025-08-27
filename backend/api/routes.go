package api

import (
	"net/http"
	"time"

	"github.com/Neat-Snap/blueprint-backend/api/handlers"
	"github.com/Neat-Snap/blueprint-backend/config"
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
	Config      config.Config
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

	authAPI := handlers.NewAuthAPI(c.DB, c.Logger, c.Connection, c.EmailClient, c.RedisSecret, c.Env, c.Config.SESSION_SECRET, c.Config)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authAPI.RegisterEndpoint)
		r.Post("/confirm-email", authAPI.ConfirmEmailEndpoint)
		r.Post("/login", authAPI.LoginEndpoint)
		r.Get("/{provider}", authAPI.ProviderBeginAuthEndpoint)
		r.Get("/{provider}/callback", authAPI.ProviderCallbackEndpoint)
		// temp
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<a href="/auth/google">Sign in with Google</a>`))
		})
	})

	return r
}

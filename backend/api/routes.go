package api

import (
	"net/http"
	"time"

	"github.com/Neat-Snap/blueprint-backend/api/handlers"
	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	mw "github.com/Neat-Snap/blueprint-backend/middleware"
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

	r.Use(mw.CORS(c.Config.APP_URL))

	if c.Env != "prod" {
		r.Use(middleware.Logger)
	}

	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(httprate.Limit(
		20,
		5*time.Second,
		httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint),
	))

	authAPI, err := handlers.NewAuthAPI(c.Logger, c.Connection, c.Config)
	if err != nil {
		c.Logger.Fatal("failed to initialise auth api", "error", err)
	}
	r.Use(mw.AuthMiddlewareBuilder(c.Logger, c.Connection, authAPI.WorkOSService(), c.Config.Env == "prod", mw.DefaultSkipper))

	api := handlers.NewTestHealthAPI(c.DB, c.Logger)
	r.Get("/health", api.HealthHandler)

	feedbackAPI := handlers.NewFeedbackAPI(c.Logger, c.Connection, c.EmailClient, c.Config)
	r.With(mw.Confirmation(c.Config, c.EmailClient.R)).Post("/feedback", feedbackAPI.SubmitEndpoint)
	r.Route("/auth", func(r chi.Router) {
		r.Method(http.MethodGet, "/signup", http.HandlerFunc(authAPI.SignupEndpoint))
		r.Method(http.MethodPost, "/signup", http.HandlerFunc(authAPI.SignupEndpoint))
		r.Method(http.MethodGet, "/login", http.HandlerFunc(authAPI.LoginEndpoint))
		r.Method(http.MethodPost, "/login", http.HandlerFunc(authAPI.LoginEndpoint))
		r.Get("/callback", authAPI.CallbackEndpoint)
		r.With(mw.Confirmation(c.Config, c.EmailClient.R)).Get("/me", authAPI.MeEndpoint)
		r.Method(http.MethodGet, "/logout", http.HandlerFunc(authAPI.LogoutEndpoint))
		r.Method(http.MethodPost, "/logout", http.HandlerFunc(authAPI.LogoutEndpoint))
		r.Post("/resend-email", authAPI.ResendEmailEndpoint)
		r.Post("/password/reset", authAPI.ResetPasswordEndpoint)
		r.Post("/password/confirm", authAPI.ResetPasswordConfirmEndpoint)
	})

	dashboardAPI := handlers.NewDashboardAPI(c.Logger, c.Connection)
	r.Route("/dashboard", func(r chi.Router) {
		r.Use(mw.Confirmation(c.Config, c.EmailClient.R))
		r.Get("/overview", dashboardAPI.OverViewEndpoint)
	})

	teamsAPI := handlers.NewTeamsAPI(c.Logger, c.Connection)
	r.Route("/teams", func(r chi.Router) {
		r.Use(mw.Confirmation(c.Config, c.EmailClient.R))
		r.Get("/", teamsAPI.GetTeamsEndpoint)
		r.Post("/", teamsAPI.CreateTeamEndpoint)
		r.Get("/{id}", teamsAPI.GetTeamEndpoint)
		r.Get("/{id}/overview", teamsAPI.GetTeamOverviewEndpoint)
		r.Get("/{id}/invitations", teamsAPI.ListInvitationsEndpoint)
		r.Delete("/{id}/invitations/{inv_id}", teamsAPI.RevokeInvitationEndpoint)
		r.Post("/invitations/check", teamsAPI.CheckInvitationStatusEndpoint)
		r.Patch("/{id}", teamsAPI.UpdateTeamNameEndpoint)
		r.Delete("/{id}", teamsAPI.DeleteTeamEndpoint)
		r.Post("/{id}/members", teamsAPI.AddMemberEndpoint)
		r.Patch("/{id}/members/{user_id}/role", teamsAPI.UpdateMemberRoleEndpoint)
		r.Delete("/{id}/members/{user_id}", teamsAPI.RemoveMemberEndpoint)
		r.Post("/{id}/invitations", teamsAPI.CreateInvitationEndpoint)
		r.Post("/invitations/accept", teamsAPI.AcceptInvitationEndpoint)
	})

	usersAPI := handlers.NewUsersAPI(c.Logger, c.Connection, c.EmailClient, c.RedisSecret, c.Config)
	r.Route("/account", func(r chi.Router) {
		r.Use(mw.Confirmation(c.Config, c.EmailClient.R))
		r.Patch("/me", authAPI.MeEndpoint)
		r.Patch("/profile", usersAPI.UpdateProfileEndpoint)
		r.Patch("/email/change", usersAPI.ChangeEmailEndpoint)
		r.Patch("/email/confirm", usersAPI.ConfirmEmailEndpoint)
		r.Patch("/password/change", usersAPI.ChangePasswordEndpoint)

		r.Get("/preferences", usersAPI.GetPreferencesEndpoint)
		r.Post("/preferences/theme", usersAPI.UpdateUserThemeEndpoint)
		r.Post("/preferences/language", usersAPI.UpdateUserLanguage)
	})

	notificationsAPI := handlers.NewNotificationsAPI(c.Logger, c.Connection)
	r.Route("/notifications", func(r chi.Router) {
		r.Use(mw.Confirmation(c.Config, c.EmailClient.R))
		r.Get("/", notificationsAPI.ListEndpoint)
		r.Patch("/{id}/read", notificationsAPI.MarkReadEndpoint)
	})

	return r
}

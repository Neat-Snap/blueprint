package api

import (
	"time"

	"github.com/Neat-Snap/blueprint-backend/api/handlers"
	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	mw "github.com/Neat-Snap/blueprint-backend/middleware"
	"github.com/Neat-Snap/blueprint-backend/utils/email"
	"github.com/Neat-Snap/blueprint-backend/workosclient"
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
	Config      config.Config
	WorkOS      *workosclient.Client
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

	r.Use(mw.AuthMiddlewareBuilder(c.WorkOS, c.Logger, c.Connection, mw.DefaultSkipper))

	api := handlers.NewTestHealthAPI(c.DB, c.Logger)
	r.Get("/health", api.HealthHandler)

	feedbackAPI := handlers.NewFeedbackAPI(c.Logger, c.Connection, c.EmailClient, c.Config)
	r.With(mw.Confirmation(c.Config)).Post("/feedback", feedbackAPI.SubmitEndpoint)

	authAPI := handlers.NewAuthAPI(c.Logger, c.Connection, c.WorkOS, c.Config)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", authAPI.RegisterEndpoint)
		r.Post("/login", authAPI.LoginEndpoint)
		r.Post("/refresh", authAPI.RefreshEndpoint)
		r.Post("/logout", authAPI.LogoutEndpoint)
		r.Post("/password/reset", authAPI.ResetPasswordEndpoint)
		r.Post("/password/confirm", authAPI.ResetPasswordConfirmEndpoint)
		r.Post("/verify/resend", authAPI.ResendVerificationEndpoint)

		r.With(mw.Confirmation(c.Config)).Get("/me", authAPI.MeEndpoint)
		r.With(mw.Confirmation(c.Config)).Post("/verify/send", authAPI.SendVerificationEndpoint)
		r.With(mw.Confirmation(c.Config)).Post("/verify/confirm", authAPI.ConfirmEmailEndpoint)
	})

	dashboardAPI := handlers.NewDashboardAPI(c.Logger, c.Connection)
	r.Route("/dashboard", func(r chi.Router) {
		r.Use(mw.Confirmation(c.Config))
		r.Get("/overview", dashboardAPI.OverViewEndpoint)
	})

	teamsAPI := handlers.NewTeamsAPI(c.Logger, c.Connection)
	r.Route("/teams", func(r chi.Router) {
		r.Use(mw.Confirmation(c.Config))
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

	usersAPI := handlers.NewUsersAPI(c.Logger, c.Connection, c.Config, c.WorkOS)
	r.Route("/account", func(r chi.Router) {
		r.Use(mw.Confirmation(c.Config))
		r.Patch("/me", authAPI.MeEndpoint)
		r.Patch("/profile", usersAPI.UpdateProfileEndpoint)
		r.Patch("/email/change", usersAPI.ChangeEmailEndpoint)
		r.Patch("/password/change", usersAPI.ChangePasswordEndpoint)

		r.Get("/preferences", usersAPI.GetPreferencesEndpoint)
		r.Post("/preferences/theme", usersAPI.UpdateUserThemeEndpoint)
		r.Post("/preferences/language", usersAPI.UpdateUserLanguage)
	})

	notificationsAPI := handlers.NewNotificationsAPI(c.Logger, c.Connection)
	r.Route("/notifications", func(r chi.Router) {
		r.Use(mw.Confirmation(c.Config))
		r.Get("/", notificationsAPI.ListEndpoint)
		r.Patch("/{id}/read", notificationsAPI.MarkReadEndpoint)
	})

	return r
}

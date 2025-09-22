package handlers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/Neat-Snap/blueprint-backend/services"
	"github.com/Neat-Snap/blueprint-backend/utils"
	"github.com/workos/workos-go/v4/pkg/usermanagement"
	"gorm.io/gorm"
)

type VerifyUserEmailRequest struct {
	ConfirmationID string `json:"confirmation_id"`
	Code           string `json:"code"`
}

type AuthAPI struct {
	logger           logger.MultiLogger
	Connection       *db.Connection
	Config           config.Config
	workos           *services.WorkOSAuthService
	secureCookies    bool
	frontendReadyURL string
}

func NewAuthAPI(logger logger.MultiLogger, connection *db.Connection, cfg config.Config) (*AuthAPI, error) {
	workosService, err := services.NewWorkOSAuthService(cfg, logger)
	if err != nil {
		return nil, err
	}
	readyURL := strings.TrimSuffix(cfg.APP_URL, "/") + "/auth/ready"
	return &AuthAPI{
		logger:           logger,
		Connection:       connection,
		Config:           cfg,
		workos:           workosService,
		secureCookies:    cfg.Env == "prod",
		frontendReadyURL: readyURL,
	}, nil
}

func (a *AuthAPI) WorkOSService() *services.WorkOSAuthService {
	return a.workos
}

func (a *AuthAPI) SignupEndpoint(w http.ResponseWriter, r *http.Request) {
	a.redirectToWorkOS(w, r, usermanagement.SignUp)
}

func (a *AuthAPI) LoginEndpoint(w http.ResponseWriter, r *http.Request) {
	a.redirectToWorkOS(w, r, usermanagement.SignIn)
}

func (a *AuthAPI) CallbackEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()
	code := strings.TrimSpace(query.Get("code"))
	state := strings.TrimSpace(query.Get("state"))
	if code == "" || state == "" {
		utils.WriteError(w, a.logger, errors.New("missing code or state"), "Missing authorization parameters", http.StatusBadRequest)
		return
	}

	stateCookie, err := r.Cookie(services.StateCookieName)
	if err != nil || stateCookie == nil || stateCookie.Value == "" || stateCookie.Value != state {
		utils.WriteError(w, a.logger, errors.New("invalid state"), "Invalid state parameter", http.StatusUnauthorized)
		return
	}
	a.clearStateCookie(w)

	authResp, err := a.workos.AuthenticateWithCode(ctx, code, clientIP(r), r.UserAgent())
	if err != nil {
		a.handleWorkOSError(w, err, "Authentication failed")
		return
	}

	user, err := a.upsertWorkOSUser(ctx, authResp)
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to persist user", http.StatusInternalServerError)
		return
	}

	claims, err := a.workos.ParseAccessToken(authResp.AccessToken)
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to parse access token", http.StatusInternalServerError)
		return
	}
	a.setAuthCookies(w, authResp.AccessToken, claims)

	if err := a.ensureDefaultTeam(ctx, user); err != nil {
		a.logger.Warn("failed to ensure default team after login", "error", err)
	}

	http.Redirect(w, r, a.frontendReadyURL, http.StatusFound)
}

func (a *AuthAPI) MeEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tokenCookie, err := r.Cookie(services.AccessTokenCookieName)
	if err != nil || tokenCookie == nil || tokenCookie.Value == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	claims, err := a.workos.ParseAccessToken(tokenCookie.Value)
	if err != nil {
		a.clearAuthCookies(w)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	identity, err := a.Connection.Auth.FindAuthIdentity(ctx, services.WorkOSProvider, claims.Subject)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			a.clearAuthCookies(w)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		utils.WriteError(w, a.logger, err, "Failed to load auth identity", http.StatusInternalServerError)
		return
	}

	refresh := ""
	if identity.RefreshToken != nil {
		refresh = strings.TrimSpace(*identity.RefreshToken)
	}
	if refresh == "" {
		a.clearAuthCookies(w)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	refreshed, err := a.workos.AuthenticateWithRefreshToken(ctx, refresh, clientIP(r), r.UserAgent())
	if err != nil {
		var workosErr *services.WorkOSError
		if errors.As(err, &workosErr) && workosErr.Status == http.StatusUnauthorized {
			a.clearAuthCookies(w)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		a.handleWorkOSError(w, err, "Failed to refresh session")
		return
	}

	newClaims, err := a.workos.ParseAccessToken(refreshed.AccessToken)
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to parse refreshed access token", http.StatusInternalServerError)
		return
	}

	if err := a.Connection.Auth.UpdateIdentityTokens(ctx, identity.ID, refreshed.AccessToken, refreshed.RefreshToken); err != nil {
		utils.WriteError(w, a.logger, err, "Failed to persist refreshed tokens", http.StatusInternalServerError)
		return
	}

	a.setAuthCookies(w, refreshed.AccessToken, newClaims)

	user, err := a.Connection.Auth.FindUserByAuthIdentity(ctx, identity)
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to load user", http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"id":    user.ID,
		"email": "",
		"name":  "",
	}
	if user.Email != nil {
		resp["email"] = *user.Email
	}
	if user.Name != nil {
		resp["name"] = *user.Name
	}

	utils.WriteSuccess(w, a.logger, resp, http.StatusOK)
}

func (a *AuthAPI) LogoutEndpoint(w http.ResponseWriter, r *http.Request) {
	var sessionID string
	if c, err := r.Cookie(services.SessionIDCookieName); err == nil && c != nil {
		sessionID = strings.TrimSpace(c.Value)
	}
	if sessionID == "" {
		if c, err := r.Cookie(services.AccessTokenCookieName); err == nil && c != nil {
			if claims, err := a.workos.ParseAccessToken(c.Value); err == nil {
				sessionID = claims.SessionID
			}
		}
	}

	if sessionID != "" {
		if err := a.workos.RevokeSession(r.Context(), sessionID); err != nil {
			var workosErr *services.WorkOSError
			if !(errors.As(err, &workosErr) && workosErr.Status == http.StatusUnauthorized) {
				a.logger.Warn("failed to revoke workos session", "error", err)
			}
		}
	}

	a.clearAuthCookies(w)
	utils.WriteSuccess(w, a.logger, utils.DefaultResponse{Message: "Logged out", Success: true}, http.StatusOK)
}

func (a *AuthAPI) ResendEmailEndpoint(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email string `json:"email"`
	}
	if err := utils.ReadJSON(r.Body, w, a.logger, &payload); err != nil {
		return
	}
	email, err := utils.ValidateEmail(payload.Email)
	if err != nil {
		utils.WriteError(w, a.logger, err, "Invalid email", http.StatusBadRequest)
		return
	}

	lookup, err := a.workos.ListUsersByEmail(r.Context(), email)
	if err != nil {
		a.handleWorkOSError(w, err, "Failed to locate user")
		return
	}
	if len(lookup.Data) == 0 {
		utils.WriteError(w, a.logger, errors.New("user not found"), "User not found", http.StatusNotFound)
		return
	}

	if err := a.workos.SendVerificationEmail(r.Context(), lookup.Data[0].ID); err != nil {
		a.handleWorkOSError(w, err, "Failed to send verification email")
		return
	}

	utils.WriteSuccess(w, a.logger, utils.DefaultResponse{Message: "Verification email sent", Success: true}, http.StatusOK)
}

func (a *AuthAPI) ResetPasswordEndpoint(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email string `json:"email"`
	}
	if err := utils.ReadJSON(r.Body, w, a.logger, &payload); err != nil {
		return
	}
	email, err := utils.ValidateEmail(payload.Email)
	if err != nil {
		utils.WriteError(w, a.logger, err, "Invalid email", http.StatusBadRequest)
		return
	}

	if _, err := a.workos.CreatePasswordReset(r.Context(), email); err != nil {
		a.handleWorkOSError(w, err, "Failed to request password reset")
		return
	}

	utils.WriteSuccess(w, a.logger, utils.DefaultResponse{Message: "Password reset email sent", Success: true}, http.StatusOK)
}

func (a *AuthAPI) ResetPasswordConfirmEndpoint(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := utils.ReadJSON(r.Body, w, a.logger, &payload); err != nil {
		return
	}
	policy := utils.PolicyFromConfig(a.Config)
	if err := utils.ValidatePassword(payload.Password, policy); err != nil {
		utils.WriteError(w, a.logger, err, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := a.workos.ResetPassword(r.Context(), payload.Token, payload.Password); err != nil {
		a.handleWorkOSError(w, err, "Failed to reset password")
		return
	}

	utils.WriteSuccess(w, a.logger, utils.DefaultResponse{Message: "Password updated", Success: true}, http.StatusOK)
}

func (a *AuthAPI) redirectToWorkOS(w http.ResponseWriter, r *http.Request, hint usermanagement.ScreenHint) {
	state, err := a.workos.GenerateState()
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to generate state", http.StatusInternalServerError)
		return
	}
	a.setStateCookie(w, state)

	authURL, err := a.workos.AuthorizationURL(state, hint)
	if err != nil {
		a.handleWorkOSError(w, err, "Failed to create authorization URL")
		return
	}
	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

func (a *AuthAPI) setStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     services.StateCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300,
	})
}

func (a *AuthAPI) clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     services.StateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   a.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *AuthAPI) setAuthCookies(w http.ResponseWriter, token string, claims services.AccessTokenClaims) {
	maxAge := int(time.Until(claims.ExpiresAt).Seconds())
	if maxAge <= 0 {
		maxAge = 3600
	}
	accessCookie := &http.Cookie{
		Name:     services.AccessTokenCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secureCookies,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	}
	if !claims.ExpiresAt.IsZero() {
		accessCookie.Expires = claims.ExpiresAt
	}
	http.SetCookie(w, accessCookie)

	if claims.SessionID != "" {
		sessionCookie := &http.Cookie{
			Name:     services.SessionIDCookieName,
			Value:    claims.SessionID,
			Path:     "/",
			HttpOnly: true,
			Secure:   a.secureCookies,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   maxAge,
		}
		if !claims.ExpiresAt.IsZero() {
			sessionCookie.Expires = claims.ExpiresAt
		}
		http.SetCookie(w, sessionCookie)
	}
}

func (a *AuthAPI) clearAuthCookies(w http.ResponseWriter) {
	expired := time.Unix(0, 0)
	http.SetCookie(w, &http.Cookie{
		Name:     services.AccessTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  expired,
		HttpOnly: true,
		Secure:   a.secureCookies,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     services.SessionIDCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  expired,
		HttpOnly: true,
		Secure:   a.secureCookies,
		SameSite: http.SameSiteStrictMode,
	})
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}

func (a *AuthAPI) upsertWorkOSUser(ctx context.Context, authResp usermanagement.AuthenticateResponse) (*db.User, error) {
	email := utils.NormalizeEmail(authResp.User.Email)
	fullName := strings.TrimSpace(strings.TrimSpace(authResp.User.FirstName + " " + authResp.User.LastName))
	avatar := strings.TrimSpace(authResp.User.ProfilePictureURL)
	now := time.Now()
	var out *db.User

	err := a.Connection.WithTx(ctx, func(tx *db.Connection) error {
		user, err := tx.Auth.FindAuthIdentity(ctx, services.WorkOSProvider, authResp.User.ID)
		if err == nil {
			out = user.User
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if out == nil && email != "" {
			if existing, err := tx.Users.ByEmail(ctx, email); err == nil {
				out = existing
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		created := false
		if out == nil {
			out = &db.User{}
			created = true
		}

		if email != "" {
			if out.Email == nil || *out.Email != email {
				emailCopy := email
				out.Email = &emailCopy
			}
		}
		if fullName != "" {
			if out.Name == nil || *out.Name != fullName {
				nameCopy := fullName
				out.Name = &nameCopy
			}
		}
		if avatar != "" {
			if out.AvatarURL == nil || *out.AvatarURL != avatar {
				avatarCopy := avatar
				out.AvatarURL = &avatarCopy
			}
		}
		if authResp.User.EmailVerified {
			if out.EmailVerifiedAt == nil {
				verified := now
				out.EmailVerifiedAt = &verified
			}
		}

		if created {
			if err := tx.Users.Create(ctx, out); err != nil {
				if utils.IsUniqueViolation(err, "uniq_users_email") && email != "" {
					existing, err2 := tx.Users.ByEmail(ctx, email)
					if err2 != nil {
						return err2
					}
					out = existing
					created = false
				} else {
					return err
				}
			} else {
				if err := tx.Preferences.Create(ctx, out.ID); err != nil {
					return err
				}
			}
		} else {
			if err := tx.Users.Update(ctx, out); err != nil {
				return err
			}
		}

		providerEmail := email
		accessToken := authResp.AccessToken
		refreshToken := authResp.RefreshToken
		if err := tx.Auth.LinkIdentity(ctx, out.ID, services.WorkOSProvider, authResp.User.ID, &providerEmail, &accessToken, &refreshToken); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (a *AuthAPI) ensureDefaultTeam(ctx context.Context, user *db.User) error {
	if user == nil {
		return nil
	}
	return a.Connection.WithTx(ctx, func(tx *db.Connection) error {
		existing, err := tx.Teams.ListForUser(ctx, user.ID)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			return nil
		}
		name := "My team"
		if user.Name != nil {
			trimmed := strings.TrimSpace(*user.Name)
			if trimmed != "" {
				name = fmt.Sprintf("%s's team", trimmed)
			}
		}
		team := &db.Team{
			Name:    name,
			OwnerID: user.ID,
			Owner:   *user,
			Users:   []db.User{*user},
		}
		if err := tx.Teams.Create(ctx, team); err != nil {
			return err
		}
		if err := tx.Teams.AddMember(ctx, team.ID, user.ID, "owner"); err != nil {
			a.logger.Warn("failed to add owner to default team", "error", err)
		}
		return nil
	})
}

func (a *AuthAPI) handleWorkOSError(w http.ResponseWriter, err error, fallback string) {
	var workosErr *services.WorkOSError
	if errors.As(err, &workosErr) {
		status := workosErr.Status
		message := fallback
		if strings.TrimSpace(workosErr.Message) != "" {
			message = workosErr.Message
		}
		utils.WriteError(w, a.logger, err, message, status)
		return
	}
	utils.WriteError(w, a.logger, err, fallback, http.StatusBadGateway)
}

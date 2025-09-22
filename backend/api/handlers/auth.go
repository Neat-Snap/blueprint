package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/Neat-Snap/blueprint-backend/utils"
	"github.com/Neat-Snap/blueprint-backend/utils/email"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AuthAPI struct {
	DB            *gorm.DB
	logger        logger.MultiLogger
	Connection    *db.Connection
	EmailClient   *email.EmailClient
	RedisSecret   string
	CookieStore   *sessions.CookieStore
	Environment   string
	SessionSecret string
	Config        config.Config
}

type VerifyUserEmailRequest struct {
	ConfirmationID string `json:"confirmation_id"`
	Code           string `json:"code"`
}

type SignUpWithConfirmationIDResponse struct {
	ConfirmationID string `json:"confirmation_id"`
	Success        bool   `json:"success"`
	Message        string `json:"message"`
}

// -----------------------------------

// GET /auth/me
func (a *AuthAPI) MeEndpoint(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	if err != nil || c == nil || c.Value == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	email, err := utils.DecodeJWT([]byte(a.Config.JWT_SECRET), c.Value, a.Config.JWT_ISSUER, a.Config.JWT_AUDIENCE)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	u, err := a.Connection.Users.ByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	resp := map[string]any{
		"id":    u.ID,
		"email": u.Email,
		"name":  u.Name,
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func returnCookieToken(origin string, w http.ResponseWriter, token string, sessionID *string, cfg config.Config) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	secure := cfg.Env == "prod"
	sameSite := http.SameSiteStrictMode

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   3600 * 24 * 21,
	})

	if sessionID != nil && *sessionID != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    *sessionID,
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			SameSite: sameSite,
			MaxAge:   3600 * 24 * 21,
		})
	}
}

func returnDefaultPositiveResponse(w http.ResponseWriter, log logger.MultiLogger) {
	var r utils.DefaultResponse
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(r)
	if err != nil {
		utils.WriteError(w, log, err, "output error occured", http.StatusInternalServerError)
	}

}

// func tokenResponse(w http.ResponseWriter, token string, logger logger.MultiLogger) {
// 	w.WriteHeader(http.StatusOK)
// 	if err := json.NewEncoder(w).Encode(TokenResponse{Token: token, Success: true}); err != nil {
// 		logger.Warn("failed to encode response", "error", err)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		return
// 	}
// }

func NewAuthAPI(db *gorm.DB, logger logger.MultiLogger, connection *db.Connection, emailClient *email.EmailClient, redisSecret string, environment string, sessionSecret string, config config.Config) *AuthAPI {
	logger.Info("app url from config is", "app_url", config.BACKEND_PUBLIC_URL)
	cookieStore := sessions.NewCookieStore([]byte(sessionSecret))
	cookieStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 8,
		HttpOnly: true,
		Secure:   environment == "prod",
		SameSite: http.SameSiteLaxMode,
	}
	gothic.Store = cookieStore

	goth.UseProviders(
		google.New(
			config.GOOGLE_CLIENT_ID,
			config.GOOGLE_CLIENT_SECRET,
			fmt.Sprintf("%s/auth/google/callback", config.BACKEND_PUBLIC_URL),
			"openid", "email", "profile",
		),
		github.New(
			config.GITHUB_CLIENT_ID,
			config.GITHUB_CLIENT_SECRET,
			fmt.Sprintf("%s/auth/github/callback", config.BACKEND_PUBLIC_URL),
			"read:user",
		),
	)

	return &AuthAPI{DB: db, logger: logger, Connection: connection, EmailClient: emailClient, RedisSecret: redisSecret, CookieStore: cookieStore, Environment: environment, SessionSecret: sessionSecret, Config: config}
}

// POST /auth/register
func (a *AuthAPI) RegisterEndpoint(w http.ResponseWriter, r *http.Request) {
	utils.WriteError(w, a.logger, errors.New("email/password sign-up disabled"), "Email/password sign-up is no longer supported", http.StatusGone)
}

// POST /auth/confirm-email
func (a *AuthAPI) ConfirmEmailEndpoint(w http.ResponseWriter, r *http.Request) {
	var reqData VerifyUserEmailRequest
	if err := utils.ReadJSON(r.Body, w, a.logger, &reqData); err != nil {
		return
	}

	verifiedEmail, err := a.EmailClient.R.Verify(r.Context(), []byte(a.RedisSecret), email.VerifyPurpose, reqData.ConfirmationID, reqData.Code)
	if err != nil {
		switch {
		case errors.Is(err, email.ErrNotFound), errors.Is(err, email.ErrExpired):
			utils.WriteError(w, a.logger, err, "Invalid or expired code", http.StatusBadRequest)
		case errors.Is(err, email.ErrConsumed):
			utils.WriteError(w, a.logger, err, "Code already used", http.StatusBadRequest)
		case errors.Is(err, email.ErrMismatch):
			utils.WriteError(w, a.logger, err, "Invalid code", http.StatusBadRequest)
		case errors.Is(err, email.ErrTooMany):
			utils.WriteError(w, a.logger, err, "Too many attempts, try again later", http.StatusTooManyRequests)
		default:
			utils.WriteError(w, a.logger, err, "Failed to verify email", http.StatusInternalServerError)
		}
		return
	}

	err = a.Connection.WithTx(r.Context(), func(tx *db.Connection) error {
		u, err := tx.Users.ByEmail(r.Context(), verifiedEmail)
		if err != nil {
			return err
		}
		u.EmailVerified = true
		if err := tx.Users.Update(r.Context(), u); err != nil {
			return err
		}

		existing, err := tx.Teams.ListForUser(r.Context(), u.ID)
		if err != nil {
			return err
		}
		if len(existing) == 0 {
			name := "My team"
			if u.Name != nil && *u.Name != "" {
				name = *u.Name + "'s team"
			}
			ws := &db.Team{
				Name:    name,
				OwnerID: u.ID,
				Owner:   *u,
				Users:   []db.User{*u},
			}
			if err := tx.Teams.Create(r.Context(), ws); err != nil {
				return err
			}
			_ = tx.Teams.AddMember(r.Context(), ws.ID, u.ID, "owner")
		}
		return nil
	})
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to verify email", http.StatusInternalServerError)
		return
	}

	token, err := utils.GenerateJWT([]byte(a.Config.JWT_SECRET), verifiedEmail, a.Config.JWT_ISSUER, a.Config.JWT_AUDIENCE)
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to generate JWT", http.StatusInternalServerError)
		return
	}

	returnCookieToken(a.Config.APP_URL, w, token, nil, a.Config)

	returnDefaultPositiveResponse(w, a.logger)
}

// POST /auth/login
func (a *AuthAPI) LoginEndpoint(w http.ResponseWriter, r *http.Request) {
	utils.WriteError(w, a.logger, errors.New("email/password login disabled"), "Email/password login is no longer supported", http.StatusGone)
}

// GET /auth/{provider}
func (a *AuthAPI) ProviderBeginAuthEndpoint(w http.ResponseWriter, r *http.Request) {
	// if gothUser, err := gothic.CompleteUserAuth(res, req); err == nil {
	// 	t, _ := template.New("foo").Parse(userTemplate)
	// 	t.Execute(res, gothUser)
	// } else {
	// 	gothic.BeginAuthHandler(res, req)
	// }
	gothic.BeginAuthHandler(w, r)
}

// POST /auth/{provider}/callback
func (a *AuthAPI) ProviderCallbackEndpoint(w http.ResponseWriter, r *http.Request) {
	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		a.logger.Error("failed to complete auth", "error", err)
		utils.WriteError(w, a.logger, err, "Authentication failed", http.StatusUnauthorized)
		return
	}

	provider := strings.ToLower(strings.TrimSpace(gothUser.Provider))
	workosUserID := strings.TrimSpace(gothUser.UserID)
	if provider == "" || workosUserID == "" {
		http.Error(w, "invalid provider response", http.StatusBadRequest)
		return
	}

	email := utils.NormalizeEmail(gothUser.Email)
	name := utils.PickNonEmpty(gothUser.Name, gothUser.NickName)
	avatar := strings.TrimSpace(gothUser.AvatarURL)

	var metadata datatypes.JSONMap
	if len(gothUser.RawData) > 0 {
		metadata = datatypes.JSONMap(gothUser.RawData)
	}

	a.logger.Debug("got user from provider", "provider", provider, "userID", workosUserID, "email", email)

	var signedInUser *db.User
	err = a.Connection.WithTx(r.Context(), func(tx *db.Connection) error {
		var existing *db.User
		if workosUserID != "" {
			u, err := tx.Users.ByWorkOSUserID(r.Context(), workosUserID)
			if err == nil {
				existing = u
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		if existing == nil && email != "" {
			if u, err := tx.Users.ByEmail(r.Context(), email); err == nil {
				existing = u
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		updateUser := func(u *db.User) error {
			if email != "" {
				emailCopy := email
				u.Email = &emailCopy
			}
			if name != "" {
				nameCopy := name
				u.Name = &nameCopy
			}
			if avatar != "" {
				avatarCopy := avatar
				u.AvatarURL = &avatarCopy
			}
			if workosUserID != "" {
				workosCopy := workosUserID
				u.WorkOSUserID = &workosCopy
			}
			u.EmailVerified = true
			if metadata != nil {
				u.ProfileMetadata = metadata
			}
			if err := tx.Users.Update(r.Context(), u); err != nil {
				return err
			}
			signedInUser = u
			return nil
		}

		if existing != nil {
			return updateUser(existing)
		}

		newUser := &db.User{
			EmailVerified:   true,
			ProfileMetadata: metadata,
		}
		if email != "" {
			emailCopy := email
			newUser.Email = &emailCopy
		}
		if name != "" {
			nameCopy := name
			newUser.Name = &nameCopy
		}
		if avatar != "" {
			avatarCopy := avatar
			newUser.AvatarURL = &avatarCopy
		}
		if workosUserID != "" {
			workosCopy := workosUserID
			newUser.WorkOSUserID = &workosCopy
		}

		if err := tx.Users.Create(r.Context(), newUser); err != nil {
			switch {
			case utils.IsUniqueViolation(err, "uniq_users_email") && email != "":
				existingByEmail, err2 := tx.Users.ByEmail(r.Context(), email)
				if err2 != nil {
					return err2
				}
				return updateUser(existingByEmail)
			case utils.IsUniqueViolation(err, "uniq_users_workos_id"):
				existingByWorkOS, err2 := tx.Users.ByWorkOSUserID(r.Context(), workosUserID)
				if err2 != nil {
					return err2
				}
				return updateUser(existingByWorkOS)
			default:
				return err
			}
		}

		if err := tx.Preferences.Create(r.Context(), newUser.ID); err != nil {
			return err
		}

		signedInUser = newUser
		return nil
	})
	if err != nil {
		a.logger.Error("failed to complete provider auth", "error", err)
		http.Error(w, "auth failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	if terr := a.Connection.WithTx(r.Context(), func(tx *db.Connection) error {
		existing, err := tx.Teams.ListForUser(r.Context(), signedInUser.ID)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			return nil
		}
		teamName := "My team"
		if signedInUser.Name != nil && *signedInUser.Name != "" {
			teamName = *signedInUser.Name + "'s team"
		}
		ws := &db.Team{
			Name:    teamName,
			OwnerID: signedInUser.ID,
			Owner:   *signedInUser,
			Users:   []db.User{*signedInUser},
		}
		if err := tx.Teams.Create(r.Context(), ws); err != nil {
			return err
		}
		_ = tx.Teams.AddMember(r.Context(), ws.ID, signedInUser.ID, "owner")
		return nil
	}); terr != nil {
		a.logger.Warn("failed to ensure default team on oauth sign-in", "error", terr)
	}

	if signedInUser.Email == nil || *signedInUser.Email == "" {
		utils.WriteError(w, a.logger, errors.New("email missing"), "Email not available from identity provider", http.StatusUnauthorized)
		return
	}

	var sessionID *string
	if token := strings.TrimSpace(gothUser.RefreshToken); token != "" {
		hashed, hashErr := utils.HashSecret(token, utils.DefaultArgon)
		if hashErr != nil {
			a.logger.Warn("failed to hash refresh token", "error", hashErr)
		} else {
			expiresAt := gothUser.ExpiresAt
			if expiresAt.IsZero() {
				expiresAt = time.Now().Add(24 * time.Hour)
			}
			sid := uuid.NewString()
			session := &db.UserSession{
				UserID:           signedInUser.ID,
				SessionID:        sid,
				RefreshTokenHash: hashed,
				ExpiresAt:        expiresAt,
				LastUsedAt:       time.Now(),
			}
			if err := a.Connection.Auth.CreateSession(r.Context(), session); err != nil {
				a.logger.Warn("failed to persist user session", "error", err)
			} else {
				sessionID = &sid
			}
		}
	}

	token, err := utils.GenerateJWT([]byte(a.Config.JWT_SECRET), *signedInUser.Email, a.Config.JWT_ISSUER, a.Config.JWT_AUDIENCE)
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to generate JWT", http.StatusInternalServerError)
		return
	}

	returnCookieToken(a.Config.APP_URL, w, token, sessionID, a.Config)

	http.Redirect(w, r, a.Config.APP_URL+"/auth/ready", http.StatusFound)
}

// GET /auth/logout
func (a *AuthAPI) LogoutEndpoint(w http.ResponseWriter, r *http.Request) {
	if sessionCookie, err := r.Cookie("session_id"); err == nil && sessionCookie.Value != "" {
		if delErr := a.Connection.Auth.DeleteSessionByID(r.Context(), sessionCookie.Value); delErr != nil {
			a.logger.Warn("failed to revoke session", "error", delErr)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	w.WriteHeader(http.StatusOK)
	var response utils.DefaultResponse
	response.Success = true
	response.Message = "Logged out successfully"
	if err := json.NewEncoder(w).Encode(response); err != nil {
		a.logger.Warn("failed to encode response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// POST /auth/resend-email
func (a *AuthAPI) ResendEmailEndpoint(w http.ResponseWriter, r *http.Request) {
	var requestStruct struct {
		Email string `json:"email"`
	}

	err := utils.ReadJSON(r.Body, w, a.logger, &requestStruct)
	if err != nil {
		utils.WriteError(w, a.logger, err, "internal server error", http.StatusInternalServerError)
		return
	}

	mail := requestStruct.Email
	if _, err := utils.ValidateEmail(mail); err != nil {
		utils.WriteError(w, a.logger, err, "invalid email", http.StatusBadRequest)
		return
	}

	count, ttl, err := a.EmailClient.R.IncrementResend(r.Context(), email.VerifyPurpose, mail, 24*time.Hour)
	if err != nil {
		utils.WriteError(w, a.logger, err, "internal server error", http.StatusInternalServerError)
		return
	}
	if count > 3 {
		utils.WriteError(
			w,
			a.logger,
			email.ErrLimitReached,
			fmt.Sprintf("limit reached. Try again in %d seconds", int(ttl.Seconds())),
			http.StatusTooManyRequests,
		)
		return
	}
	id, err := a.EmailClient.SendConfirmationEmail(mail, "Confirm your email", 60)
	if err != nil {
		utils.WriteError(w, a.logger, err, "error sending confirmation email", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(SignUpWithConfirmationIDResponse{Message: "User registered successfully", Success: true, ConfirmationID: id}); err != nil {
		a.logger.Warn("failed to encode response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (a *AuthAPI) ResetPasswordEndpoint(w http.ResponseWriter, r *http.Request) {
	utils.WriteError(w, a.logger, errors.New("password reset disabled"), "Password reset is no longer supported", http.StatusGone)
}

func (a *AuthAPI) ResetPasswordConfirmEndpoint(w http.ResponseWriter, r *http.Request) {
	utils.WriteError(w, a.logger, errors.New("password reset disabled"), "Password reset is no longer supported", http.StatusGone)
}

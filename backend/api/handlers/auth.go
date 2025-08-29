package handlers

import (
	"encoding/gob"
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
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
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

// -----------------------------------
type EmailPassUserCreds struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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

type TokenResponse struct {
	Token   string `json:"token"`
	Success bool   `json:"success"`
}

type SessionUser struct {
	Provider     string
	UserID       string
	Email        string
	Name         string
	AvatarURL    string
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

// -----------------------------------

// GET /auth/me
func (a *AuthAPI) MeEndpoint(w http.ResponseWriter, r *http.Request) {
	// Read token from HttpOnly cookie
	c, err := r.Cookie("token")
	if err != nil || c == nil || c.Value == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	email, err := utils.DecodeJWT([]byte(a.RedisSecret), c.Value)
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

func returnCookieToken(origin string, w http.ResponseWriter, token string, cfg config.Config) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	secure := cfg.Env == "prod"
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   3600 * 24 * 21,
	})
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
	gob.Register(SessionUser{})
	logger.Debug("app url from config is", "app_url", config.Addr)
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
			fmt.Sprintf("%s/auth/google/callback", config.Addr),
			"openid", "email", "profile",
		),
	)

	return &AuthAPI{DB: db, logger: logger, Connection: connection, EmailClient: emailClient, RedisSecret: redisSecret, CookieStore: cookieStore, Environment: environment, SessionSecret: sessionSecret, Config: config}
}

// POST /auth/register
func (a *AuthAPI) RegisterEndpoint(w http.ResponseWriter, r *http.Request) {
	var u EmailPassUserCreds
	if err := utils.ReadJSON(r.Body, w, a.logger, &u); err != nil {
		return
	}

	_, err := utils.SignUpEmailPassword(r.Context(), a.Connection, u.Email, u.Password, "")
	if err != nil {
		if errors.Is(err, utils.ErrEmailTaken) {
			utils.WriteError(w, a.logger, err, "Email already in use", http.StatusConflict)
			return
		}
		if errors.Is(err, utils.ErrOAuthOnlyAccount) {
			utils.WriteError(w, a.logger, err, "This email is registered via Google. Please continue with Google.", http.StatusConflict)
			return
		}
		utils.WriteError(w, a.logger, err, "Could not register", http.StatusInternalServerError)
		return
	}

	id, err := a.EmailClient.SendConfirmationEmail(u.Email, "Confirm your email", 60)
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to send confirmation email", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(SignUpWithConfirmationIDResponse{Message: "User registered successfully", Success: true, ConfirmationID: id}); err != nil {
		a.logger.Warn("failed to encode response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
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
		now := time.Now()
		u.EmailVerifiedAt = &now
		return tx.Users.Update(r.Context(), u)
	})
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to verify email", http.StatusInternalServerError)
		return
	}

	token, err := utils.GenerateJWT([]byte(a.RedisSecret), verifiedEmail)
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to generate JWT", http.StatusInternalServerError)
		return
	}

	returnCookieToken(a.Config.APP_URL, w, token, a.Config)

	returnDefaultPositiveResponse(w, a.logger)
}

// POST /auth/login
func (a *AuthAPI) LoginEndpoint(w http.ResponseWriter, r *http.Request) {
	var u EmailPassUserCreds
	if err := utils.ReadJSON(r.Body, w, a.logger, &u); err != nil {
		return
	}

	var tokenOnSuccess string
	err := a.Connection.WithTx(r.Context(), func(tx *db.Connection) error {
		user, err := tx.Users.ByEmail(r.Context(), u.Email)
		if err != nil {
			return err
		}

		// Guard against users who registered via OAuth and do not have a password credential
		if user.PasswordCredential == nil || user.PasswordCredential.PasswordDisabled {
			return utils.ErrOAuthOnlyAccount
		}

		ok, err := utils.ComparePassword(u.Password, user.PasswordCredential.PasswordHash)
		if err != nil {
			return err
		}

		if !ok {
			return errors.New("invalid password")
		}

		token, err := utils.GenerateJWT([]byte(a.RedisSecret), u.Email)
		if err != nil {
			return err
		}

		tokenOnSuccess = token
		return nil
	})

	if err != nil {
		if errors.Is(err, utils.ErrOAuthOnlyAccount) {
			utils.WriteError(w, a.logger, err, "This account uses Google sign-in. Please continue with Google.", http.StatusConflict)
			return
		}
		utils.WriteError(w, a.logger, err, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	returnCookieToken(a.Config.APP_URL, w, tokenOnSuccess, a.Config)

	returnDefaultPositiveResponse(w, a.logger)
}

// GET /auth/{provider}
func (a *AuthAPI) ProviderBeginAuthEndpoint(w http.ResponseWriter, r *http.Request) {
	gothic.BeginAuthHandler(w, r)
}

// POST /auth/{provider}/callback
func (a *AuthAPI) ProviderCallbackEndpoint(w http.ResponseWriter, r *http.Request) {
	u, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		a.logger.Error("failed to complete auth", "error", err)
		utils.WriteError(w, a.logger, err, "Authentication failed", http.StatusUnauthorized)
		return
	}

	provider := strings.ToLower(strings.TrimSpace(u.Provider))
	subject := strings.TrimSpace(u.UserID)

	if provider == "" || subject == "" {
		http.Error(w, "invalid provider response", http.StatusBadRequest)
		return
	}

	email := utils.NormalizeEmail(u.Email)

	appUser := SessionUser{
		Provider:     provider,
		UserID:       subject,
		Email:        u.Email,
		Name:         utils.PickNonEmpty(u.Name, u.NickName),
		AvatarURL:    u.AvatarURL,
		AccessToken:  u.AccessToken,
		RefreshToken: u.RefreshToken,
		Expiry:       u.ExpiresAt,
	}

	now := time.Now()
	dbUser := &db.User{
		Email:           &email,
		Name:            &appUser.Name,
		EmailVerifiedAt: &now,
		AvatarURL:       &appUser.AvatarURL,
	}

	var signedInUser *db.User
	err = a.Connection.WithTx(r.Context(), func(tx *db.Connection) error {
		if ai, err := tx.Auth.FindAuthIdentity(r.Context(), provider, subject); err == nil {
			a.logger.Debug("found auth identity", "provider", provider, "subject", subject)
			signedUser, err := tx.Auth.FindUserByAuthIdentity(r.Context(), ai)
			if err != nil {
				return err
			}
			signedInUser = signedUser
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			a.logger.Error("failed to find auth identity", "error", err)
			return err
		}

		// todo add a check for users who already have a valid cookie

		if curr, err := tx.Users.ByEmail(r.Context(), email); err == nil {
			// User with this email already exists, link the new auth identity to them.
			if err := tx.Auth.LinkIdentity(r.Context(), curr.ID, provider, subject, &email); err != nil {
				if utils.IsUniqueViolation(err, "uniq_provider_subject") {
					ai, err2 := tx.Auth.FindAuthIdentity(r.Context(), provider, subject)
					if err2 != nil {
						return err2
					}

					signedUser, err := tx.Auth.FindUserByAuthIdentity(r.Context(), ai)
					if err != nil {
						return err
					}

					signedInUser = signedUser
					return nil
				}
				return err
			}

			signedInUser = curr
			return nil
		}

		if err := tx.Users.Create(r.Context(), dbUser); err != nil {
			if utils.IsUniqueViolation(err, "uniq_users_email") {
				u2, err2 := tx.Users.ByEmail(r.Context(), email)
				if err2 != nil {
					return err2
				}
				if err := tx.Auth.LinkIdentity(r.Context(), u2.ID, provider, subject, &email); err != nil {
					return err
				}
				signedInUser = u2
				return nil
			}
		}

		if err := tx.Auth.LinkIdentity(r.Context(), dbUser.ID, provider, subject, &email); err != nil {
			return err
		}
		signedInUser = dbUser
		return nil
	})

	if err != nil {
		a.logger.Error("failed to complete auth", "error", err)
		http.Error(w, "auth failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	token, err := utils.GenerateJWT([]byte(a.RedisSecret), *signedInUser.Email)
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to generate JWT", http.StatusInternalServerError)
		return
	}

	returnCookieToken(a.Config.APP_URL, w, token, a.Config)

	http.Redirect(w, r, a.Config.APP_URL+"/auth/ready", http.StatusFound)
}

// GET /auth/logout
func (a *AuthAPI) LogoutEndpoint(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
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

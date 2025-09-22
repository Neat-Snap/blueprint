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
	"github.com/markbates/goth/gothic"
	"gorm.io/gorm"
)

type AuthAPI struct {
	DB          *gorm.DB
	logger      logger.MultiLogger
	Connection  *db.Connection
	EmailClient *email.EmailClient
	RedisSecret string
	CookieStore *sessions.CookieStore
	Session     config.SessionConfig
	WorkOS      config.WorkOSConfig
	Config      config.Config
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

func sameSiteMode(mode string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "lax":
		return http.SameSiteLaxMode
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteDefaultMode
	}
}

// GET /auth/me
func (a *AuthAPI) MeEndpoint(w http.ResponseWriter, r *http.Request) {
	cookieName := a.Session.Token.Name
	if cookieName == "" {
		cookieName = "token"
	}
	c, err := r.Cookie(cookieName)
	if err != nil || c == nil || c.Value == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	email, err := utils.DecodeJWT([]byte(a.Session.TokenSecret), c.Value)
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

	cookieCfg := cfg.Session.Token
	if cookieCfg.Name == "" {
		cookieCfg.Name = "token"
	}
	if cookieCfg.Path == "" {
		cookieCfg.Path = "/"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieCfg.Name,
		Value:    token,
		Path:     cookieCfg.Path,
		Domain:   cookieCfg.Domain,
		HttpOnly: cookieCfg.HTTPOnly,
		Secure:   cookieCfg.Secure,
		SameSite: sameSiteMode(cookieCfg.SameSite),
		MaxAge:   cookieCfg.MaxAge,
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

func NewAuthAPI(db *gorm.DB, logger logger.MultiLogger, connection *db.Connection, emailClient *email.EmailClient, redisSecret string, sessionCfg config.SessionConfig, workosCfg config.WorkOSConfig, config config.Config) *AuthAPI {
	gob.Register(SessionUser{})
	logger.Info("app url from config is", "app_url", config.BACKEND_PUBLIC_URL)
	cookieStore := sessions.NewCookieStore([]byte(sessionCfg.Secret))
	storeOptions := sessions.Options{
		Path:     sessionCfg.Store.Path,
		Domain:   sessionCfg.Store.Domain,
		MaxAge:   sessionCfg.Store.MaxAge,
		HttpOnly: sessionCfg.Store.HTTPOnly,
		Secure:   sessionCfg.Store.Secure,
		SameSite: sameSiteMode(sessionCfg.Store.SameSite),
	}
	if storeOptions.Path == "" {
		storeOptions.Path = "/"
	}
	if storeOptions.MaxAge == 0 {
		storeOptions.MaxAge = 3600 * 8
	}
	cookieStore.Options = &storeOptions
	gothic.Store = cookieStore

	return &AuthAPI{DB: db, logger: logger, Connection: connection, EmailClient: emailClient, RedisSecret: redisSecret, CookieStore: cookieStore, Session: sessionCfg, WorkOS: workosCfg, Config: config}
}

// POST /auth/register
func (a *AuthAPI) RegisterEndpoint(w http.ResponseWriter, r *http.Request) {
	var u EmailPassUserCreds
	if err := utils.ReadJSON(r.Body, w, a.logger, &u); err != nil {
		return
	}

	policy := utils.PolicyFromConfig(a.Config)
	email, err := utils.ValidateEmail(u.Email)
	if err != nil {
		utils.WriteError(w, a.logger, err, "invalid email", http.StatusBadRequest)
		return
	}
	if err := utils.ValidatePassword(u.Password, policy); err != nil {
		utils.WriteError(w, a.logger, err, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = utils.SignUpEmailPassword(r.Context(), a.Connection, email, u.Password, "")
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

	id, err := a.EmailClient.SendConfirmationEmail(email, "Confirm your email", 60)
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

	token, err := utils.GenerateJWT([]byte(a.Session.TokenSecret), verifiedEmail)
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

	email, err := utils.ValidateEmail(u.Email)
	if err != nil {
		utils.WriteError(w, a.logger, err, "invalid email", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(u.Password) == "" {
		utils.WriteError(w, a.logger, errors.New("password required"), "password required", http.StatusBadRequest)
		return
	}

	var tokenOnSuccess string
	err = a.Connection.WithTx(r.Context(), func(tx *db.Connection) error {
		user, err := tx.Users.ByEmail(r.Context(), email)
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

		token, err := utils.GenerateJWT([]byte(a.Session.TokenSecret), email)
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
	u, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		a.logger.Error("failed to complete auth...", "error", err)
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

	name := utils.PickNonEmpty(u.Name, u.NickName)
	now := time.Now()
	dbUser := &db.User{
		Email:           &email,
		Name:            &name,
		EmailVerifiedAt: &now,
		AvatarURL:       &u.AvatarURL,
	}

	a.logger.Debug("got user from provider: ", "user", u)

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
			if err := tx.Auth.LinkIdentity(r.Context(), curr.ID, provider, subject, &email, &u.AccessToken, &u.RefreshToken); err != nil {
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
				if err := tx.Auth.LinkIdentity(r.Context(), u2.ID, provider, subject, &email, &u.AccessToken, &u.RefreshToken); err != nil {
					return err
				}
				signedInUser = u2
				return nil
			}
		}

		if err := tx.Auth.LinkIdentity(r.Context(), dbUser.ID, provider, subject, &email, &u.AccessToken, &u.RefreshToken); err != nil {
			return err
		}

		if err := tx.Preferences.Create(r.Context(), dbUser.ID); err != nil {
			return err
		}

		signedInUser = dbUser
		return nil
	})

	if err != nil {
		a.logger.Error("failed to complete auth with provider "+provider, "error", err)
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
		name := "My team"
		if signedInUser.Name != nil && *signedInUser.Name != "" {
			name = *signedInUser.Name + "'s team"
		}
		ws := &db.Team{
			Name:    name,
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

	token, err := utils.GenerateJWT([]byte(a.Session.TokenSecret), *signedInUser.Email)
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to generate JWT", http.StatusInternalServerError)
		return
	}

	returnCookieToken(a.Config.APP_URL, w, token, a.Config)

	http.Redirect(w, r, a.Config.APP_URL+"/auth/ready", http.StatusFound)
}

// GET /auth/logout
func (a *AuthAPI) LogoutEndpoint(w http.ResponseWriter, r *http.Request) {
	cookieCfg := a.Config.Session.Token
	if cookieCfg.Name == "" {
		cookieCfg.Name = "token"
	}
	if cookieCfg.Path == "" {
		cookieCfg.Path = "/"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieCfg.Name,
		Value:    "",
		Path:     cookieCfg.Path,
		Domain:   cookieCfg.Domain,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: cookieCfg.HTTPOnly,
		Secure:   cookieCfg.Secure,
		SameSite: sameSiteMode(cookieCfg.SameSite),
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
	mail = strings.ToLower(strings.TrimSpace(mail))
	u, err := a.Connection.Users.ByEmail(r.Context(), mail)
	if err != nil {
		utils.WriteError(w, a.logger, err, "user not found", http.StatusNotFound)
		return
	}

	if u.PasswordCredential == nil || u.PasswordCredential.PasswordDisabled {
		utils.WriteError(w, a.logger, utils.ErrOAuthOnlyAccount, "oauth only account", http.StatusUnauthorized)
		return
	}

	if ok, ttl, err := a.EmailClient.R.AllowOncePer(r.Context(), email.ResetPasswordPurpose, mail, 72*time.Hour); err != nil {
		utils.WriteError(w, a.logger, err, "internal server error", http.StatusInternalServerError)
		return
	} else if !ok {
		utils.WriteError(
			w,
			a.logger,
			email.ErrLimitReached,
			fmt.Sprintf("Password reset already requested recently. Try again in %d hours", int(ttl.Hours())),
			http.StatusTooManyRequests,
		)
		return
	}

	count, ttl, err := a.EmailClient.R.IncrementResend(r.Context(), email.ResetPasswordPurpose, mail, 4*time.Hour)
	if err != nil {
		utils.WriteError(w, a.logger, err, "internal server error", http.StatusInternalServerError)
		return
	}

	if count > 3 {
		utils.WriteError(w, a.logger, email.ErrLimitReached, fmt.Sprintf("limit reached. Try again in %d seconds", int(ttl.Seconds())), http.StatusTooManyRequests)
		return
	}

	if _, err := a.EmailClient.SendResetPasswordEmail(mail, "Reset your password", 60); err != nil {
		utils.WriteError(w, a.logger, err, "error sending reset password email", http.StatusInternalServerError)
		return
	}

	resp := struct {
		Message string `json:"message"`
		Success bool   `json:"success"`
	}{
		Message: "Reset password email sent successfully",
		Success: true,
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		a.logger.Warn("failed to encode response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (a *AuthAPI) ResetPasswordConfirmEndpoint(w http.ResponseWriter, r *http.Request) {
	var requestStruct struct {
		ResetID  string `json:"reset_password_id"`
		Code     string `json:"code"`
		Password string `json:"password"`
	}

	err := utils.ReadJSON(r.Body, w, a.logger, &requestStruct)
	if err != nil {
		return
	}

	policy := utils.PolicyFromConfig(a.Config)
	if err := utils.ValidatePassword(requestStruct.Password, policy); err != nil {
		utils.WriteError(w, a.logger, err, err.Error(), http.StatusBadRequest)
		return
	}

	mail_address, err := a.EmailClient.R.Verify(r.Context(), []byte(a.RedisSecret), email.ResetPasswordPurpose, requestStruct.ResetID, requestStruct.Code)
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

	err = utils.ResetPassword(r.Context(), a.Connection, mail_address, requestStruct.Password)
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to update password", http.StatusInternalServerError)
		return
	}

	token, err := utils.GenerateJWT([]byte(a.Session.TokenSecret), mail_address)
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	returnCookieToken(a.Config.APP_URL, w, token, a.Config)

	http.Redirect(w, r, a.Config.APP_URL+"/auth/ready?password_reset=true", http.StatusFound)
}

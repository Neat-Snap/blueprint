package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	mw "github.com/Neat-Snap/blueprint-backend/middleware"
	"github.com/Neat-Snap/blueprint-backend/utils"
	"github.com/Neat-Snap/blueprint-backend/workosclient"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/workos/workos-go/v5/pkg/usermanagement"
)

type AuthAPI struct {
	logger     logger.MultiLogger
	Connection *db.Connection
	WorkOS     *workosclient.Client
	Config     config.Config
}

func NewAuthAPI(logger logger.MultiLogger, connection *db.Connection, workos *workosclient.Client, cfg config.Config) *AuthAPI {
	return &AuthAPI{
		logger:     logger,
		Connection: connection,
		WorkOS:     workos,
		Config:     cfg,
	}
}

type registerRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type verifyEmailRequest struct {
	Code           string `json:"code"`
	ConfirmationID string `json:"confirmation_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
}

type passwordResetRequest struct {
	Email string `json:"email"`
}

type passwordResetConfirmRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type resendVerificationRequest struct {
	Email string `json:"email"`
}

func (a *AuthAPI) RegisterEndpoint(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := utils.ReadJSON(r.Body, w, a.logger, &req); err != nil {
		return
	}

	email, err := utils.ValidateEmail(req.Email)
	if err != nil {
		utils.WriteError(w, a.logger, err, "invalid email", http.StatusBadRequest)
		return
	}
	if err := utils.ValidatePassword(req.Password, utils.PolicyFromConfig(a.Config)); err != nil {
		utils.WriteError(w, a.logger, err, err.Error(), http.StatusBadRequest)
		return
	}

	first, err := utils.ValidateOptionalName(req.FirstName)
	if err != nil {
		utils.WriteError(w, a.logger, err, err.Error(), http.StatusBadRequest)
		return
	}
	last, err := utils.ValidateOptionalName(req.LastName)
	if err != nil {
		utils.WriteError(w, a.logger, err, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := a.WorkOS.CreateUser(r.Context(), usermanagement.CreateUserOpts{
		Email:     email,
		Password:  req.Password,
		FirstName: first,
		LastName:  last,
	})
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to create user", http.StatusBadGateway)
		return
	}

	if _, err := a.WorkOS.EnsureLocalUser(r.Context(), a.Connection, user); err != nil {
		utils.WriteError(w, a.logger, err, "failed to persist user", http.StatusInternalServerError)
		return
	}

	if _, err := a.WorkOS.SendVerificationEmail(r.Context(), user.ID); err != nil {
		a.logger.Warn("failed to send verification email", "error", err)
	}

	utils.WriteSuccess(w, a.logger, map[string]any{
		"success":         true,
		"message":         "User registered",
		"confirmation_id": user.ID,
	}, http.StatusCreated)
}

func (a *AuthAPI) LoginEndpoint(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := utils.ReadJSON(r.Body, w, a.logger, &req); err != nil {
		return
	}

	email, err := utils.ValidateEmail(req.Email)
	if err != nil {
		utils.WriteError(w, a.logger, err, "invalid email", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		utils.WriteError(w, a.logger, errors.New("password required"), "password required", http.StatusBadRequest)
		return
	}

	ip, ua := requestMetadata(r)
	authRes, err := a.WorkOS.AuthenticateWithPassword(r.Context(), email, req.Password, ip, ua)
	if err != nil {
		utils.WriteError(w, a.logger, err, "invalid credentials", http.StatusUnauthorized)
		return
	}

	claims, err := a.WorkOS.ParseAndValidateAccessToken(r.Context(), authRes.AccessToken)
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to validate access token", http.StatusUnauthorized)
		return
	}

	expiry, err := extractExpiry(claims)
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to read token expiry", http.StatusUnauthorized)
		return
	}
	sessionID, err := claimString(claims, "sid")
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to read session id", http.StatusUnauthorized)
		return
	}

	dbUser, err := a.WorkOS.EnsureLocalUser(r.Context(), a.Connection, authRes.User)
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to sync user", http.StatusInternalServerError)
		return
	}
	a.ensureDefaultTeam(r.Context(), dbUser)

	state := workosclient.SessionState{RefreshToken: authRes.RefreshToken, SessionID: sessionID}
	if err := a.WorkOS.SetSessionCookies(w, authRes.AccessToken, expiry, state); err != nil {
		utils.WriteError(w, a.logger, err, "failed to set cookies", http.StatusInternalServerError)
		return
	}

	utils.WriteSuccess(w, a.logger, map[string]any{
		"success": true,
		"user":    userPayload(dbUser),
	}, http.StatusOK)
}

func (a *AuthAPI) RefreshEndpoint(w http.ResponseWriter, r *http.Request) {
	session, err := a.WorkOS.SessionFromRequest(r)
	if err != nil {
		utils.WriteError(w, a.logger, err, "missing refresh token", http.StatusUnauthorized)
		return
	}

	ip, ua := requestMetadata(r)
	refreshRes, err := a.WorkOS.AuthenticateWithRefreshToken(r.Context(), session.RefreshToken, ip, ua)
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to refresh session", http.StatusUnauthorized)
		return
	}

	claims, err := a.WorkOS.ParseAndValidateAccessToken(r.Context(), refreshRes.AccessToken)
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to validate access token", http.StatusUnauthorized)
		return
	}
	expiry, err := extractExpiry(claims)
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to read token expiry", http.StatusUnauthorized)
		return
	}
	sessionID, err := claimString(claims, "sid")
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to read session id", http.StatusUnauthorized)
		return
	}
	workosID, err := claimString(claims, "sub")
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to read subject", http.StatusUnauthorized)
		return
	}

	dbUser, err := a.WorkOS.EnsureLocalUserByID(r.Context(), a.Connection, workosID)
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to sync user", http.StatusInternalServerError)
		return
	}

	state := workosclient.SessionState{RefreshToken: refreshRes.RefreshToken, SessionID: sessionID}
	if err := a.WorkOS.SetSessionCookies(w, refreshRes.AccessToken, expiry, state); err != nil {
		utils.WriteError(w, a.logger, err, "failed to set cookies", http.StatusInternalServerError)
		return
	}

	utils.WriteSuccess(w, a.logger, map[string]any{
		"success": true,
		"user":    userPayload(dbUser),
	}, http.StatusOK)
}

func (a *AuthAPI) LogoutEndpoint(w http.ResponseWriter, r *http.Request) {
	if session, err := a.WorkOS.SessionFromRequest(r); err == nil {
		if err := a.WorkOS.RevokeSession(r.Context(), session.SessionID); err != nil {
			a.logger.Warn("failed to revoke session", "error", err)
		}
	}
	a.WorkOS.ClearSessionCookies(w)
	utils.WriteSuccess(w, a.logger, utils.DefaultResponse{Success: true, Message: "Logged out"}, http.StatusOK)
}

func (a *AuthAPI) MeEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj, ok := r.Context().Value(mw.UserObjectContextKey).(*db.User)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	utils.WriteSuccess(w, a.logger, map[string]any{
		"user": userPayload(userObj),
	}, http.StatusOK)
}

func (a *AuthAPI) SendVerificationEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj, ok := r.Context().Value(mw.UserObjectContextKey).(*db.User)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if _, err := a.WorkOS.SendVerificationEmail(r.Context(), userObj.WorkOSUserID); err != nil {
		utils.WriteError(w, a.logger, err, "failed to send verification email", http.StatusInternalServerError)
		return
	}

	utils.WriteSuccess(w, a.logger, utils.DefaultResponse{Success: true, Message: "Verification email sent"}, http.StatusOK)
}

func (a *AuthAPI) ConfirmEmailEndpoint(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailRequest
	if err := utils.ReadJSON(r.Body, w, a.logger, &req); err != nil {
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		utils.WriteError(w, a.logger, errors.New("code required"), "code required", http.StatusBadRequest)
		return
	}

	target := strings.TrimSpace(req.ConfirmationID)
	if target == "" {
		target = strings.TrimSpace(req.UserID)
	}
	if target == "" {
		if userObj, ok := r.Context().Value(mw.UserObjectContextKey).(*db.User); ok && userObj != nil {
			target = userObj.WorkOSUserID
		}
	}
	if target == "" {
		utils.WriteError(w, a.logger, errors.New("user id required"), "user id required", http.StatusBadRequest)
		return
	}

	res, err := a.WorkOS.VerifyEmail(r.Context(), target, code)
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to verify email", http.StatusUnauthorized)
		return
	}

	updated, err := a.WorkOS.EnsureLocalUser(r.Context(), a.Connection, res.User)
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to sync user", http.StatusInternalServerError)
		return
	}
	a.ensureDefaultTeam(r.Context(), updated)

	utils.WriteSuccess(w, a.logger, map[string]any{
		"success": true,
		"message": "Email verified",
		"user":    userPayload(updated),
	}, http.StatusOK)
}

func (a *AuthAPI) ResetPasswordEndpoint(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequest
	if err := utils.ReadJSON(r.Body, w, a.logger, &req); err != nil {
		return
	}
	email, err := utils.ValidateEmail(req.Email)
	if err != nil {
		utils.WriteError(w, a.logger, err, "invalid email", http.StatusBadRequest)
		return
	}

	if err := usermanagement.SendPasswordResetEmail(r.Context(), usermanagement.SendPasswordResetEmailOpts{
		Email:            email,
		PasswordResetUrl: fmt.Sprintf("%s/auth/password/confirm", strings.TrimRight(a.Config.APP_URL, "/")),
	}); err != nil {
		utils.WriteError(w, a.logger, err, "failed to send reset email", http.StatusInternalServerError)
		return
	}

	utils.WriteSuccess(w, a.logger, utils.DefaultResponse{Success: true, Message: "Password reset email sent"}, http.StatusOK)
}

func (a *AuthAPI) ResetPasswordConfirmEndpoint(w http.ResponseWriter, r *http.Request) {
	var req passwordResetConfirmRequest
	if err := utils.ReadJSON(r.Body, w, a.logger, &req); err != nil {
		return
	}

	if strings.TrimSpace(req.Token) == "" {
		utils.WriteError(w, a.logger, errors.New("token required"), "token required", http.StatusBadRequest)
		return
	}
	if err := utils.ValidatePassword(req.Password, utils.PolicyFromConfig(a.Config)); err != nil {
		utils.WriteError(w, a.logger, err, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := usermanagement.ResetPassword(r.Context(), usermanagement.ResetPasswordOpts{Token: req.Token, NewPassword: req.Password}); err != nil {
		utils.WriteError(w, a.logger, err, "failed to reset password", http.StatusBadRequest)
		return
	}

	utils.WriteSuccess(w, a.logger, utils.DefaultResponse{Success: true, Message: "Password updated"}, http.StatusOK)
}

func (a *AuthAPI) ProviderBeginAuthEndpoint(w http.ResponseWriter, r *http.Request) {
	provider := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "provider")))
	params, err := a.providerAuthorizationParams(provider)
	if err != nil {
		http.Error(w, "provider not supported", http.StatusNotFound)
		return
	}

	redirectPath := a.sanitizeRedirect(r.URL.Query().Get("redirect"))
	if redirectPath == "" {
		redirectPath = a.defaultRedirectPath()
	}

	stateValue, err := a.WorkOS.EncodeOAuthState(workosclient.OAuthState{Redirect: redirectPath})
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to encode state", http.StatusInternalServerError)
		return
	}
	params.State = stateValue

	authURL, err := a.WorkOS.AuthorizationURL(params)
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to build authorization url", http.StatusBadGateway)
		return
	}

	http.Redirect(w, r, authURL, http.StatusFound)
}

func (a *AuthAPI) ProviderCallbackEndpoint(w http.ResponseWriter, r *http.Request) {
	provider := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "provider")))
	if _, err := a.providerAuthorizationParams(provider); err != nil {
		http.Error(w, "provider not supported", http.StatusNotFound)
		return
	}

	if errParam := strings.TrimSpace(r.URL.Query().Get("error")); errParam != "" {
		utils.WriteError(w, a.logger, errors.New(errParam), "authentication failed", http.StatusBadRequest)
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		utils.WriteError(w, a.logger, errors.New("missing code"), "missing code", http.StatusBadRequest)
		return
	}

	ip, ua := requestMetadata(r)
	authRes, err := a.WorkOS.AuthenticateWithCode(r.Context(), code, ip, ua)
	if err != nil {
		utils.WriteError(w, a.logger, err, "authentication failed", http.StatusUnauthorized)
		return
	}

	claims, err := a.WorkOS.ParseAndValidateAccessToken(r.Context(), authRes.AccessToken)
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to validate access token", http.StatusUnauthorized)
		return
	}

	expiry, err := extractExpiry(claims)
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to read token expiry", http.StatusUnauthorized)
		return
	}
	sessionID, err := claimString(claims, "sid")
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to read session id", http.StatusUnauthorized)
		return
	}

	dbUser, err := a.WorkOS.EnsureLocalUser(r.Context(), a.Connection, authRes.User)
	if err != nil {
		utils.WriteError(w, a.logger, err, "failed to sync user", http.StatusInternalServerError)
		return
	}
	a.ensureDefaultTeam(r.Context(), dbUser)

	stateRedirect := a.decodeRedirect(r.URL.Query().Get("state"))
	if stateRedirect == "" {
		stateRedirect = a.defaultRedirectPath()
	}

	sessionState := workosclient.SessionState{RefreshToken: authRes.RefreshToken, SessionID: sessionID}
	if err := a.WorkOS.SetSessionCookies(w, authRes.AccessToken, expiry, sessionState); err != nil {
		utils.WriteError(w, a.logger, err, "failed to set cookies", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, a.resolveRedirect(stateRedirect), http.StatusFound)
}

func (a *AuthAPI) ResendVerificationEndpoint(w http.ResponseWriter, r *http.Request) {
	var req resendVerificationRequest
	if err := utils.ReadJSON(r.Body, w, a.logger, &req); err != nil {
		return
	}
	email, err := utils.ValidateEmail(req.Email)
	if err != nil {
		utils.WriteError(w, a.logger, err, "invalid email", http.StatusBadRequest)
		return
	}

	users, err := usermanagement.ListUsers(r.Context(), usermanagement.ListUsersOpts{Email: email, Limit: 1})
	if err != nil || len(users.Data) == 0 {
		utils.WriteError(w, a.logger, errors.New("user not found"), "user not found", http.StatusNotFound)
		return
	}

	userID := users.Data[0].ID

	if _, err := a.WorkOS.SendVerificationEmail(r.Context(), userID); err != nil {
		utils.WriteError(w, a.logger, err, "failed to send verification email", http.StatusInternalServerError)
		return
	}

	utils.WriteSuccess(w, a.logger, map[string]any{
		"success":         true,
		"message":         "Verification email sent",
		"confirmation_id": userID,
	}, http.StatusOK)
}

func (a *AuthAPI) providerAuthorizationParams(provider string) (workosclient.AuthorizationParams, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))

	params := workosclient.AuthorizationParams{}

	if conn := a.providerConnection(provider); conn != "" {
		params.ConnectionID = conn
	} else if ident := providerIdentifier(provider); ident != "" {
		params.Provider = ident
	} else {
		return workosclient.AuthorizationParams{}, fmt.Errorf("unsupported provider: %s", provider)
	}

	return params, nil
}

func (a *AuthAPI) providerConnection(provider string) string {
	switch provider {
	case "google":
		return strings.TrimSpace(a.Config.WORKOS_GOOGLE_CONNECTION_ID)
	case "github":
		return strings.TrimSpace(a.Config.WORKOS_GITHUB_CONNECTION_ID)
	default:
		return ""
	}
}

func providerIdentifier(provider string) string {
	switch provider {
	case "google":
		return "GoogleOAuth"
	case "github":
		return "GitHubOAuth"
	default:
		return ""
	}
}

func (a *AuthAPI) defaultRedirectPath() string {
	if redirect := a.sanitizeRedirect(a.Config.WORKOS_DEFAULT_REDIRECT_PATH); redirect != "" {
		return redirect
	}
	return "/auth/ready"
}

func (a *AuthAPI) sanitizeRedirect(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "/") {
		return raw
	}

	target, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if target.Host == "" {
		return ""
	}

	base, err := url.Parse(a.Config.APP_URL)
	if err != nil || base.Host == "" {
		return ""
	}
	if !strings.EqualFold(base.Host, target.Host) {
		return ""
	}

	uri := target.Path
	if uri == "" {
		uri = "/"
	}
	if target.RawQuery != "" {
		uri += "?" + target.RawQuery
	}
	if target.Fragment != "" {
		uri += "#" + target.Fragment
	}
	return uri
}

func (a *AuthAPI) decodeRedirect(state string) string {
	state = strings.TrimSpace(state)
	if state == "" {
		return ""
	}
	decoded, err := a.WorkOS.DecodeOAuthState(state)
	if err != nil {
		a.logger.Warn("failed to decode oauth state", "error", err)
		return ""
	}
	return a.sanitizeRedirect(decoded.Redirect)
}

func (a *AuthAPI) resolveRedirect(path string) string {
	path = a.sanitizeRedirect(path)
	if path == "" {
		path = a.defaultRedirectPath()
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return strings.TrimRight(a.Config.APP_URL, "/") + path
}

func requestMetadata(r *http.Request) (string, string) {
	ip := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if ip != "" {
		parts := strings.Split(ip, ",")
		if len(parts) > 0 {
			ip = strings.TrimSpace(parts[0])
		}
	}
	if ip == "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			ip = host
		} else {
			ip = r.RemoteAddr
		}
	}
	return ip, r.UserAgent()
}

func extractExpiry(claims jwt.MapClaims) (time.Time, error) {
	raw, ok := claims["exp"]
	if !ok {
		return time.Time{}, errors.New("exp missing")
	}
	switch v := raw.(type) {
	case float64:
		return time.Unix(int64(v), 0), nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(i, 0), nil
	case int64:
		return time.Unix(v, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported exp type %T", raw)
	}
}

func claimString(claims jwt.MapClaims, key string) (string, error) {
	raw, ok := claims[key]
	if !ok {
		return "", fmt.Errorf("%s missing", key)
	}
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return "", fmt.Errorf("%s empty", key)
		}
		return v, nil
	default:
		return "", fmt.Errorf("%s invalid", key)
	}
}

func userPayload(u *db.User) map[string]any {
	payload := map[string]any{
		"id":             u.ID,
		"workos_id":      u.WorkOSUserID,
		"email_verified": u.EmailVerifiedAt != nil,
	}
	if u.Email != nil {
		payload["email"] = *u.Email
	}
	if u.Name != nil {
		payload["name"] = *u.Name
	}
	if u.AvatarURL != nil {
		payload["avatar_url"] = *u.AvatarURL
	}
	return payload
}

func (a *AuthAPI) ensureDefaultTeam(ctx context.Context, user *db.User) {
	if user == nil {
		return
	}
	if user.EmailVerifiedAt == nil {
		return
	}
	teams, err := a.Connection.Teams.ListForUser(ctx, user.ID)
	if err != nil {
		a.logger.Warn("failed to list user teams", "error", err)
		return
	}
	if len(teams) > 0 {
		return
	}

	name := "My team"
	if user.Name != nil && strings.TrimSpace(*user.Name) != "" {
		name = fmt.Sprintf("%s's team", strings.TrimSpace(*user.Name))
	}
	team := &db.Team{
		Name:    name,
		OwnerID: user.ID,
		Owner:   *user,
		Users:   []db.User{*user},
	}
	if err := a.Connection.Teams.Create(ctx, team); err != nil {
		a.logger.Warn("failed to create default team", "error", err)
		return
	}
	if err := a.Connection.Teams.AddMember(ctx, team.ID, user.ID, "owner"); err != nil {
		a.logger.Warn("failed to add owner to default team", "error", err)
	}
}

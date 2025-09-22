package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	mw "github.com/Neat-Snap/blueprint-backend/middleware"
	"github.com/Neat-Snap/blueprint-backend/utils"
	"github.com/Neat-Snap/blueprint-backend/workosclient"
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
	Code string `json:"code"`
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

	utils.WriteSuccess(w, a.logger, utils.DefaultResponse{Success: true, Message: "User registered"}, http.StatusCreated)
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
	userObj, ok := r.Context().Value(mw.UserObjectContextKey).(*db.User)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req verifyEmailRequest
	if err := utils.ReadJSON(r.Body, w, a.logger, &req); err != nil {
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		utils.WriteError(w, a.logger, errors.New("code required"), "code required", http.StatusBadRequest)
		return
	}

	res, err := a.WorkOS.VerifyEmail(r.Context(), userObj.WorkOSUserID, req.Code)
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

	if _, err := a.WorkOS.SendVerificationEmail(r.Context(), users.Data[0].ID); err != nil {
		utils.WriteError(w, a.logger, err, "failed to send verification email", http.StatusInternalServerError)
		return
	}

	utils.WriteSuccess(w, a.logger, utils.DefaultResponse{Success: true, Message: "Verification email sent"}, http.StatusOK)
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

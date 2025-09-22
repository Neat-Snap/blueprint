package handlers

import (
	"errors"
	"net/http"
	"slices"

	"strings"

	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/Neat-Snap/blueprint-backend/middleware"
	"github.com/Neat-Snap/blueprint-backend/utils"
	"github.com/Neat-Snap/blueprint-backend/utils/email"
)

type UsersAPI struct {
	logger      logger.MultiLogger
	Connection  *db.Connection
	EmailClient *email.EmailClient
	RedisSecret string
	Config      config.Config
}

func NewUsersAPI(logger logger.MultiLogger, connection *db.Connection, emailClient *email.EmailClient, redisSecret string, config config.Config) *UsersAPI {
	return &UsersAPI{logger: logger, Connection: connection, EmailClient: emailClient, RedisSecret: redisSecret, Config: config}
}

// PATCH /account/profile
func (h *UsersAPI) UpdateProfileEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)

	type Rreq struct {
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}

	var req Rreq

	err := utils.ReadJSON(r.Body, w, h.logger, &req)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}

	userObj.Name = &req.Name
	userObj.AvatarURL = &req.AvatarURL

	err = h.Connection.Users.Update(r.Context(), userObj)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to update user", http.StatusInternalServerError)
		return
	}

	resp := Rreq{
		Name:      *userObj.Name,
		AvatarURL: *userObj.AvatarURL,
	}

	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// POST /account/password/change
func (h *UsersAPI) ChangePasswordEndpoint(w http.ResponseWriter, r *http.Request) {
	utils.WriteError(w, h.logger, errors.New("password authentication disabled"), "Password authentication is no longer supported", http.StatusGone)
}

// POST /accounts/email/change
func (h *UsersAPI) ChangeEmailEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)

	type Rreq struct {
		Email string `json:"email"`
	}

	var req Rreq

	err := utils.ReadJSON(r.Body, w, h.logger, &req)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}

	newEmail := utils.NormalizeEmail(req.Email)
	if newEmail == "" || !strings.Contains(newEmail, "@") {
		utils.WriteError(w, h.logger, errors.New("invalid email"), "invalid email", http.StatusBadRequest)
		return
	}

	if existing, err := h.Connection.Users.ByEmail(r.Context(), newEmail); err == nil {
		if existing.ID != userObj.ID {
			utils.WriteError(w, h.logger, errors.New("email already in use"), "Email already in use", http.StatusConflict)
			return
		}
	}

	userObj.Email = &newEmail
	userObj.EmailVerified = false

	err = h.Connection.Users.Update(r.Context(), userObj)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to update user", http.StatusInternalServerError)
		return
	}

	id, err := h.EmailClient.SendConfirmationEmail(*userObj.Email, "Updating your email", 60)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to send confirmation email", http.StatusInternalServerError)
		return
	}

	resp := struct {
		ID string `json:"confirmation_id"`
	}{
		ID: id,
	}

	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// POST /accounts/email/confirm
func (h *UsersAPI) ConfirmEmailEndpoint(w http.ResponseWriter, r *http.Request) {
	var req VerifyUserEmailRequest
	err := utils.ReadJSON(r.Body, w, h.logger, &req)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}

	verifiedEmail, err := h.EmailClient.R.Verify(r.Context(), []byte(h.RedisSecret), email.VerifyPurpose, req.ConfirmationID, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, email.ErrNotFound), errors.Is(err, email.ErrExpired):
			utils.WriteError(w, h.logger, err, "Invalid or expired code", http.StatusBadRequest)
		case errors.Is(err, email.ErrConsumed):
			utils.WriteError(w, h.logger, err, "Code already used", http.StatusBadRequest)
		case errors.Is(err, email.ErrMismatch):
			utils.WriteError(w, h.logger, err, "Invalid code", http.StatusBadRequest)
		case errors.Is(err, email.ErrTooMany):
			utils.WriteError(w, h.logger, err, "Too many attempts, try again later", http.StatusTooManyRequests)
		default:
			utils.WriteError(w, h.logger, err, "Failed to verify email", http.StatusInternalServerError)
		}
		return
	}

	err = h.Connection.WithTx(r.Context(), func(tx *db.Connection) error {
		u, err := tx.Users.ByEmail(r.Context(), verifiedEmail)
		if err != nil {
			return err
		}
		u.EmailVerified = true
		return tx.Users.Update(r.Context(), u)
	})
	if err != nil {
		utils.WriteError(w, h.logger, err, "Failed to verify email", http.StatusInternalServerError)
		return
	}

	token, err := utils.GenerateJWT([]byte(h.Config.Session.TokenSecret), verifiedEmail)
	if err != nil {
		utils.WriteError(w, h.logger, err, "Failed to generate JWT", http.StatusInternalServerError)
		return
	}

	returnCookieToken(h.Config.APP_URL, w, token, nil, h.Config)

	returnDefaultPositiveResponse(w, h.logger)
}

// GET /account/preferences
func (h *UsersAPI) GetPreferencesEndpoint(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Context().Value(middleware.UserEmailContextKey).(string)

	preferences, err := h.Connection.Preferences.GetByEmail(r.Context(), userEmail)
	if err != nil {
		utils.WriteError(w, h.logger, err, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := struct {
		Theme    string `json:"theme"`
		Language string `json:"lang"`
	}{
		Theme:    preferences.Theme,
		Language: preferences.Language,
	}

	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// POST /account/preferences/theme
func (h *UsersAPI) UpdateUserThemeEndpoint(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Context().Value(middleware.UserEmailContextKey).(string)

	var req struct {
		Theme string `json:"theme"`
	}

	err := utils.ReadJSON(r.Body, w, h.logger, &req)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}

	allowedThemes := []string{"light", "dark", "system"}
	if !slices.Contains(allowedThemes, req.Theme) {
		h.logger.Debug("got theme", req.Theme)
		utils.WriteError(w, h.logger, err, "theme type not allowed", http.StatusBadRequest)
		return
	}

	preference, err := h.Connection.Preferences.GetByEmail(r.Context(), userEmail)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}

	preference.Theme = req.Theme

	if err = h.Connection.Preferences.Update(r.Context(), preference); err != nil {
		utils.WriteError(w, h.logger, err, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}{
		Success: true,
		Message: "user theme updated successfully",
	}

	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// POST /account/preferences/language
func (h *UsersAPI) UpdateUserLanguage(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Context().Value(middleware.UserEmailContextKey).(string)

	var req struct {
		Lang string `json:"lang"`
	}

	err := utils.ReadJSON(r.Body, w, h.logger, &req)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}

	allowedLanguages := []string{"en", "ru", "zh"}
	if !slices.Contains(allowedLanguages, req.Lang) {
		utils.WriteError(w, h.logger, err, "language not found", http.StatusBadRequest)
		return
	}

	preference, err := h.Connection.Preferences.GetByEmail(r.Context(), userEmail)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}

	preference.Language = req.Lang

	if err = h.Connection.Preferences.Update(r.Context(), preference); err != nil {
		utils.WriteError(w, h.logger, err, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}{
		Success: true,
		Message: "user language updated successfully",
	}

	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

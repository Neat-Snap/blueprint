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
	"github.com/workos/workos-go/v4/pkg/usermanagement"
	"github.com/workos/workos-go/v4/pkg/workos_errors"
)

type UsersAPI struct {
	logger         logger.MultiLogger
	Connection     *db.Connection
	EmailClient    *email.EmailClient
	RedisSecret    string
	Config         config.Config
	UserManagement *usermanagement.Client
}

func NewUsersAPI(logger logger.MultiLogger, connection *db.Connection, emailClient *email.EmailClient, redisSecret string, config config.Config, umClient *usermanagement.Client) *UsersAPI {
	return &UsersAPI{logger: logger, Connection: connection, EmailClient: emailClient, RedisSecret: redisSecret, Config: config, UserManagement: umClient}
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
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)

	type Rreq struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	var req Rreq

	err := utils.ReadJSON(r.Body, w, h.logger, &req)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}

	if userObj.Email == nil || *userObj.Email == "" {
		utils.WriteError(w, h.logger, errors.New("email missing"), "email missing", http.StatusBadRequest)
		return
	}
	if userObj.WorkOSUserID == nil || *userObj.WorkOSUserID == "" {
		utils.WriteError(w, h.logger, utils.ErrOAuthOnlyAccount, "oauth only account", http.StatusUnauthorized)
		return
	}

	policy := utils.PolicyFromConfig(h.Config)
	if err := utils.ValidatePassword(req.NewPassword, policy); err != nil {
		utils.WriteError(w, h.logger, err, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = h.UserManagement.AuthenticateWithPassword(r.Context(), usermanagement.AuthenticateWithPasswordOpts{
		ClientID: h.Config.WORKOS_CLIENT_ID,
		Email:    utils.NormalizeEmail(*userObj.Email),
		Password: req.CurrentPassword,
	})
	if err != nil {
		var httpErr workos_errors.HTTPError
		if errors.As(err, &httpErr) {
			if httpErr.Code == http.StatusUnauthorized || httpErr.Code == http.StatusBadRequest {
				utils.WriteError(w, h.logger, errors.New("invalid password"), "invalid password", http.StatusUnauthorized)
				return
			}
		}
		utils.WriteError(w, h.logger, err, "failed to verify password", http.StatusInternalServerError)
		return
	}

	updatedUser, err := h.UserManagement.UpdateUser(r.Context(), usermanagement.UpdateUserOpts{
		User:     *userObj.WorkOSUserID,
		Password: req.NewPassword,
	})
	if err != nil {
		var httpErr workos_errors.HTTPError
		if errors.As(err, &httpErr) {
			if httpErr.Code == http.StatusBadRequest {
				utils.WriteError(w, h.logger, err, httpErr.Message, http.StatusBadRequest)
				return
			}
		}
		utils.WriteError(w, h.logger, err, "failed to update password", http.StatusInternalServerError)
		return
	}

	err = h.Connection.WithTx(r.Context(), func(tx *db.Connection) error {
		user, err := tx.Users.ByID(r.Context(), userObj.ID)
		if err != nil {
			return err
		}
		applyWorkOSUser(user, updatedUser)
		return tx.Users.Update(r.Context(), user)
	})
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to update password", http.StatusInternalServerError)
		return
	}

	utils.WriteSuccess(w, h.logger, nil, http.StatusOK)
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

	if userObj.WorkOSUserID == nil || *userObj.WorkOSUserID == "" {
		utils.WriteError(w, h.logger, utils.ErrOAuthOnlyAccount, "oauth only account", http.StatusUnauthorized)
		return
	}

	updatedUser, err := h.UserManagement.UpdateUser(r.Context(), usermanagement.UpdateUserOpts{
		User:          *userObj.WorkOSUserID,
		Email:         newEmail,
		EmailVerified: false,
	})
	if err != nil {
		var httpErr workos_errors.HTTPError
		if errors.As(err, &httpErr) {
			if httpErr.Code == http.StatusConflict {
				utils.WriteError(w, h.logger, err, "Email already in use", http.StatusConflict)
				return
			}
			if httpErr.Code == http.StatusBadRequest {
				utils.WriteError(w, h.logger, err, httpErr.Message, http.StatusBadRequest)
				return
			}
		}
		utils.WriteError(w, h.logger, err, "failed to update user", http.StatusInternalServerError)
		return
	}

	err = h.Connection.WithTx(r.Context(), func(tx *db.Connection) error {
		user, err := tx.Users.ByID(r.Context(), userObj.ID)
		if err != nil {
			return err
		}
		applyWorkOSUser(user, updatedUser)
		if err := tx.Users.Update(r.Context(), user); err != nil {
			return err
		}
		return tx.Auth.DeleteAuthIdentity(r.Context(), user.ID)
	})
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to update user", http.StatusInternalServerError)
		return
	}

	if _, err := h.UserManagement.SendVerificationEmail(r.Context(), usermanagement.SendVerificationEmailOpts{User: *userObj.WorkOSUserID}); err != nil {
		utils.WriteError(w, h.logger, err, "failed to send confirmation email", http.StatusInternalServerError)
		return
	}

	resp := struct {
		ID string `json:"confirmation_id"`
	}{
		ID: *userObj.WorkOSUserID,
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

	resp, err := h.UserManagement.VerifyEmail(r.Context(), usermanagement.VerifyEmailOpts{User: req.ConfirmationID, Code: req.Code})
	if err != nil {
		var httpErr workos_errors.HTTPError
		if errors.As(err, &httpErr) {
			switch httpErr.Code {
			case http.StatusBadRequest, http.StatusUnauthorized:
				utils.WriteError(w, h.logger, err, "Invalid or expired code", http.StatusBadRequest)
				return
			case http.StatusTooManyRequests:
				utils.WriteError(w, h.logger, err, "Too many attempts, try again later", http.StatusTooManyRequests)
				return
			}
		}
		utils.WriteError(w, h.logger, err, "Failed to verify email", http.StatusInternalServerError)
		return
	}

	err = h.Connection.WithTx(r.Context(), func(tx *db.Connection) error {
		u, err := tx.Users.ByWorkOSID(r.Context(), resp.User.ID)
		if err != nil {
			return err
		}
		applyWorkOSUser(u, resp.User)
		return tx.Users.Update(r.Context(), u)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.WriteError(w, h.logger, err, "User not found", http.StatusNotFound)
			return
		}
		utils.WriteError(w, h.logger, err, "Failed to verify email", http.StatusInternalServerError)
		return
	}

	email := utils.NormalizeEmail(resp.User.Email)
	token, err := utils.GenerateJWT([]byte(h.Config.JWT_SECRET), email, h.Config.JWT_ISSUER, h.Config.JWT_AUDIENCE)
	if err != nil {
		utils.WriteError(w, h.logger, err, "Failed to generate JWT", http.StatusInternalServerError)
		return
	}

	returnCookieToken(h.Config.APP_URL, w, token, h.Config)

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

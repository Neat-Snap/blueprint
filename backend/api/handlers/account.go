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
	"github.com/Neat-Snap/blueprint-backend/workosclient"
	"github.com/workos/workos-go/v5/pkg/usermanagement"
)

type UsersAPI struct {
	logger     logger.MultiLogger
	Connection *db.Connection
	Config     config.Config
	WorkOS     *workosclient.Client
}

func NewUsersAPI(logger logger.MultiLogger, connection *db.Connection, cfg config.Config, workos *workosclient.Client) *UsersAPI {
	return &UsersAPI{logger: logger, Connection: connection, Config: cfg, WorkOS: workos}
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

	first, last := splitFullName(req.Name)
	_, err = usermanagement.UpdateUser(r.Context(), usermanagement.UpdateUserOpts{
		User:      userObj.WorkOSUserID,
		FirstName: first,
		LastName:  last,
	})
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to update profile", http.StatusBadGateway)
		return
	}

	updated, err := h.WorkOS.EnsureLocalUserByID(r.Context(), h.Connection, userObj.WorkOSUserID)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to sync user", http.StatusInternalServerError)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name != "" {
		updated.Name = &name
	}
	avatar := strings.TrimSpace(req.AvatarURL)
	if avatar != "" {
		updated.AvatarURL = &avatar
	} else {
		updated.AvatarURL = nil
	}

	if err := h.Connection.Users.Update(r.Context(), updated); err != nil {
		utils.WriteError(w, h.logger, err, "failed to persist profile", http.StatusInternalServerError)
		return
	}

	utils.WriteSuccess(w, h.logger, map[string]any{
		"name":       name,
		"avatar_url": avatar,
	}, http.StatusOK)
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
		utils.WriteError(w, h.logger, errors.New("email required"), "email not set", http.StatusBadRequest)
		return
	}

	if err := utils.ValidatePassword(req.NewPassword, utils.PolicyFromConfig(h.Config)); err != nil {
		utils.WriteError(w, h.logger, err, err.Error(), http.StatusBadRequest)
		return
	}

	ip, ua := requestMetadata(r)
	if _, err := h.WorkOS.AuthenticateWithPassword(r.Context(), *userObj.Email, req.CurrentPassword, ip, ua); err != nil {
		utils.WriteError(w, h.logger, err, "invalid current password", http.StatusUnauthorized)
		return
	}

	if _, err := usermanagement.UpdateUser(r.Context(), usermanagement.UpdateUserOpts{User: userObj.WorkOSUserID, Password: req.NewPassword}); err != nil {
		utils.WriteError(w, h.logger, err, "failed to update password", http.StatusBadGateway)
		return
	}

	utils.WriteSuccess(w, h.logger, utils.DefaultResponse{Success: true, Message: "Password updated"}, http.StatusOK)
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

	email, err := utils.ValidateEmail(req.Email)
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid email", http.StatusBadRequest)
		return
	}

	if _, err := usermanagement.UpdateUser(r.Context(), usermanagement.UpdateUserOpts{User: userObj.WorkOSUserID, Email: email}); err != nil {
		utils.WriteError(w, h.logger, err, "failed to update email", http.StatusBadGateway)
		return
	}

	updated, err := h.WorkOS.EnsureLocalUserByID(r.Context(), h.Connection, userObj.WorkOSUserID)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to sync user", http.StatusInternalServerError)
		return
	}

	if _, err := h.WorkOS.SendVerificationEmail(r.Context(), userObj.WorkOSUserID); err != nil {
		h.logger.Warn("failed to send verification email", "error", err)
	}

	utils.WriteSuccess(w, h.logger, map[string]any{
		"email":          email,
		"email_verified": updated.EmailVerifiedAt != nil,
	}, http.StatusOK)
}

// GET /account/preferences
func (h *UsersAPI) GetPreferencesEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)

	preferences, err := h.Connection.Preferences.Get(r.Context(), userObj.ID)
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
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)

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

	preference, err := h.Connection.Preferences.Get(r.Context(), userObj.ID)
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
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)

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

	preference, err := h.Connection.Preferences.Get(r.Context(), userObj.ID)
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

func splitFullName(full string) (string, string) {
	trimmed := strings.TrimSpace(full)
	if trimmed == "" {
		return "", ""
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

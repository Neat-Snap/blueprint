package handlers

import (
	"errors"
	"net/http"
	"time"

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

	pc, err := h.Connection.Auth.FindPasswordCredential(r.Context(), userObj.ID)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to find password credential", http.StatusInternalServerError)
		return
	}

	valid, err := utils.ComparePassword(req.CurrentPassword, pc.PasswordHash)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to compare password", http.StatusInternalServerError)
		return
	}

	if !valid {
		utils.WriteError(w, h.logger, errors.New("invalid password"), "invalid password", http.StatusUnauthorized)
		return
	}

	hashedNew, err := utils.HashPassword(req.NewPassword, utils.DefaultArgon)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to hash password", http.StatusInternalServerError)
		return
	}

	err = h.Connection.Auth.EnsurePasswordCredential(r.Context(), userObj.ID, hashedNew)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to ensure password credential", http.StatusInternalServerError)
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

	userObj.Email = &req.Email
	userObj.EmailVerifiedAt = nil

	err = h.Connection.Auth.DeleteAuthIdentity(r.Context(), userObj.ID)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to delete auth identity", http.StatusInternalServerError)
		return
	}

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
		now := time.Now()
		u.EmailVerifiedAt = &now
		return tx.Users.Update(r.Context(), u)
	})
	if err != nil {
		utils.WriteError(w, h.logger, err, "Failed to verify email", http.StatusInternalServerError)
		return
	}

	token, err := utils.GenerateJWT([]byte(h.RedisSecret), verifiedEmail)
	if err != nil {
		utils.WriteError(w, h.logger, err, "Failed to generate JWT", http.StatusInternalServerError)
		return
	}

	returnCookieToken(h.Config.APP_URL, w, token, h.Config)

	returnDefaultPositiveResponse(w, h.logger)
}

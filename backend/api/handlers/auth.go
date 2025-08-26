package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/Neat-Snap/blueprint-backend/utils"
	"gorm.io/gorm"
)

type AuthAPI struct {
	DB         *gorm.DB
	logger     logger.MultiLogger
	Connection *db.Connection
}

type SignUpUser struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewAuthAPI(db *gorm.DB, logger logger.MultiLogger, connection *db.Connection) *AuthAPI {
	return &AuthAPI{DB: db, logger: logger, Connection: connection}
}

func (a *AuthAPI) RegisterEndpoint(w http.ResponseWriter, r *http.Request) {
	var u SignUpUser
	if err := utils.ReadJSON(r.Body, w, a.logger, &u); err != nil {
		return
	}

	_, err := utils.SignUpEmailPassword(r.Context(), a.Connection, u.Email, u.Password, "")
	if err != nil {
		utils.WriteError(w, a.logger, err, "Failed to sign up user", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(utils.DefaultResponse{Message: "User registered successfully", Success: true}); err != nil {
		a.logger.Warn("failed to encode response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (a *AuthAPI) ConfirmEmailEndpoint(w http.ResponseWriter, r *http.Request) {

}

func (a *AuthAPI) LoginEndpoint(w http.ResponseWriter, r *http.Request) {

}

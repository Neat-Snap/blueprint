package handlers

import (
	"net/http"
	"strconv"

	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/Neat-Snap/blueprint-backend/middleware"
	"github.com/Neat-Snap/blueprint-backend/utils"
	"github.com/go-chi/chi/v5"
)

type NotificationsAPI struct {
	logger     logger.MultiLogger
	Connection *db.Connection
}

func NewNotificationsAPI(logger logger.MultiLogger, connection *db.Connection) *NotificationsAPI {
	return &NotificationsAPI{logger: logger, Connection: connection}
}

// GET /notifications
func (h *NotificationsAPI) ListEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	list, err := h.Connection.Notifications.ListForUser(r.Context(), userObj.ID)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to list notifications", http.StatusInternalServerError)
		return
	}
	utils.WriteSuccess(w, h.logger, list, http.StatusOK)
}

// PATCH /notifications/{id}/read
func (h *NotificationsAPI) MarkReadEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid notification id", http.StatusBadRequest)
		return
	}
	var n db.Notification
	if err := h.Connection.DBConn.WithContext(r.Context()).Where("id = ? AND user_id = ?", id, userObj.ID).First(&n).Error; err != nil {
		utils.WriteError(w, h.logger, err, "notification not found", http.StatusNotFound)
		return
	}
	if err := h.Connection.Notifications.MarkRead(r.Context(), uint(id)); err != nil {
		utils.WriteError(w, h.logger, err, "failed to mark notification read", http.StatusInternalServerError)
		return
	}
	utils.WriteSuccess(w, h.logger, map[string]any{"status": "ok"}, http.StatusOK)
}

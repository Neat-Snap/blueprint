package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/Neat-Snap/blueprint-backend/middleware"
	"github.com/Neat-Snap/blueprint-backend/utils"
)

type HandlersAPI struct {
	logger     logger.MultiLogger
	Connection *db.Connection
}

func NewDashboardAPI(logger logger.MultiLogger, connection *db.Connection) *HandlersAPI {
	return &HandlersAPI{logger: logger, Connection: connection}
}

// GET /dashboard/overview
func (h *HandlersAPI) OverViewEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	workspaces, err := h.Connection.Workspaces.ListForUser(r.Context(), userID)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get workspaces", http.StatusInternalServerError)
		return
	}

	var ownded int
	var workspaceResp []any
	for _, workspace := range workspaces {
		var role string
		if workspace.OwnerID == userID {
			role = "owner"
			ownded += 1
		} else {
			role = "member"
		}

		workspaceResp = append(workspaceResp, map[string]any{
			"id":   workspace.ID,
			"name": workspace.Name,
			"role": role,
		})
	}

	statsResp := map[string]any{
		"total_workspaces": len(workspaces),
		"owner_workspaces": ownded,
	}

	userResp := utils.UserConciseResponse{
		ID:    userObj.ID,
		Name:  *userObj.Name,
		Email: *userObj.Email,
	}

	finalResponse := map[string]any{
		"stats":      statsResp,
		"user":       userResp,
		"workspaces": workspaceResp,
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(finalResponse); err != nil {
		utils.WriteError(w, h.logger, err, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

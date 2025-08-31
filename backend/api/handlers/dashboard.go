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

	teams, err := h.Connection.Teams.ListForUser(r.Context(), userID)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get teams", http.StatusInternalServerError)
		return
	}

	var ownded int
	var teamResp []any
	for _, team := range teams {
		var role string
		if team.OwnerID == userID {
			role = "owner"
			ownded += 1
		} else {
			role = "member"
		}

		teamResp = append(teamResp, map[string]any{
			"id":   team.ID,
			"name": team.Name,
			"role": role,
		})
	}

	statsResp := map[string]any{
		"total_teams": len(teams),
		"owner_teams": ownded,
	}

	userResp := utils.UserConciseResponse{
		ID:    userObj.ID,
		Name:  *userObj.Name,
		Email: *userObj.Email,
	}

	finalResponse := map[string]any{
		"stats": statsResp,
		"user":  userResp,
		"teams": teamResp,
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(finalResponse); err != nil {
		utils.WriteError(w, h.logger, err, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

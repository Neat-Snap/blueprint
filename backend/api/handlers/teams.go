package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/Neat-Snap/blueprint-backend/middleware"
	"github.com/Neat-Snap/blueprint-backend/utils"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type TeamsAPI struct {
	logger     logger.MultiLogger
	Connection *db.Connection
}

// PATCH /teams/{id}/members/{user_id}/role
func (h *TeamsAPI) UpdateMemberRoleEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid team ID", http.StatusBadRequest)
		return
	}

	team, err := h.Connection.Teams.ByID(r.Context(), uint(teamID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get team", http.StatusInternalServerError)
		return
	}

	if team.OwnerID != userID {
		utils.WriteError(w, h.logger, nil, "user does not have access to change member roles", http.StatusForbidden)
		return
	}

	memberID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid member ID", http.StatusBadRequest)
		return
	}

	if uint(memberID) == team.OwnerID {
		utils.WriteError(w, h.logger, nil, "cannot change role of the team owner", http.StatusBadRequest)
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := utils.ReadJSON(r.Body, w, h.logger, &req); err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}
	if req.Role != "regular" && req.Role != "admin" {
		utils.WriteError(w, h.logger, fmt.Errorf("invalid role: %s", req.Role), "invalid role", http.StatusBadRequest)
		return
	}

	if err := h.Connection.Teams.AddMember(r.Context(), uint(teamID), uint(memberID), req.Role); err != nil {
		utils.WriteError(w, h.logger, err, "failed to update member role", http.StatusInternalServerError)
		return
	}

	utils.WriteSuccess(w, h.logger, map[string]any{"status": "updated"}, http.StatusOK)
}

// GET /teams/{id}/invitations
func (h *TeamsAPI) ListInvitationsEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid team ID", http.StatusBadRequest)
		return
	}
	team, err := h.Connection.Teams.ByID(r.Context(), uint(teamID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get team", http.StatusInternalServerError)
		return
	}
	rolesMap, _ := h.Connection.Teams.RolesForTeam(r.Context(), uint(teamID))
	role := rolesMap[userObj.ID]
	if team.OwnerID != userObj.ID && role != "admin" {
		utils.WriteError(w, h.logger, nil, "forbidden", http.StatusForbidden)
		return
	}
	list, err := h.Connection.Invitations.ListByTeam(r.Context(), uint(teamID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to list invitations", http.StatusInternalServerError)
		return
	}
	type item struct {
		ID        uint      `json:"id"`
		Email     string    `json:"email"`
		Role      string    `json:"role"`
		Token     string    `json:"token"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	resp := make([]item, 0, len(list))
	for _, i := range list {
		resp = append(resp, item{ID: i.ID, Email: i.Email, Role: i.Role, Token: i.Token, Status: i.Status, CreatedAt: i.CreatedAt, ExpiresAt: i.ExpiresAt})
	}
	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// DELETE /teams/{id}/invitations/{inv_id}
func (h *TeamsAPI) RevokeInvitationEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid team ID", http.StatusBadRequest)
		return
	}
	team, err := h.Connection.Teams.ByID(r.Context(), uint(teamID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get team", http.StatusInternalServerError)
		return
	}
	rolesMap, _ := h.Connection.Teams.RolesForTeam(r.Context(), uint(teamID))
	role := rolesMap[userObj.ID]
	if team.OwnerID != userObj.ID && role != "admin" {
		utils.WriteError(w, h.logger, nil, "forbidden", http.StatusForbidden)
		return
	}
	invID, err := strconv.Atoi(chi.URLParam(r, "inv_id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid invitation ID", http.StatusBadRequest)
		return
	}
	if err := h.Connection.Invitations.Revoke(r.Context(), uint(invID)); err != nil {
		utils.WriteError(w, h.logger, err, "failed to revoke invitation", http.StatusInternalServerError)
		return
	}
	utils.WriteSuccess(w, h.logger, map[string]any{"status": "revoked"}, http.StatusOK)
}

// POST /teams/{id}/invitations
func (h *TeamsAPI) CreateInvitationEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid team ID", http.StatusBadRequest)
		return
	}

	team, err := h.Connection.Teams.ByID(r.Context(), uint(teamID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get team", http.StatusInternalServerError)
		return
	}

	rolesMap, _ := h.Connection.Teams.RolesForTeam(r.Context(), uint(teamID))
	role := rolesMap[userObj.ID]
	if team.OwnerID != userObj.ID && role != "admin" {
		utils.WriteError(w, h.logger, nil, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := utils.ReadJSON(r.Body, w, h.logger, &req); err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" {
		utils.WriteError(w, h.logger, nil, "email is required", http.StatusBadRequest)
		return
	}
	if req.Role == "" {
		req.Role = "regular"
	}
	if req.Role != "regular" && req.Role != "admin" {
		utils.WriteError(w, h.logger, fmt.Errorf("invalid role: %s", req.Role), "invalid role", http.StatusBadRequest)
		return
	}

	inviteEmail := strings.ToLower(req.Email)
	u, uerr := h.Connection.Users.ByEmail(r.Context(), inviteEmail)
	if uerr != nil {
		if errors.Is(uerr, gorm.ErrRecordNotFound) {
			utils.WriteError(w, h.logger, nil, "user with this email does not exist", http.StatusBadRequest)
			return
		}
		utils.WriteError(w, h.logger, uerr, "failed to look up user", http.StatusInternalServerError)
		return
	}

	if rolesMap[u.ID] != "" {
		utils.WriteError(w, h.logger, nil, "user is already a team member", http.StatusBadRequest)
		return
	}

	if existing, lerr := h.Connection.Invitations.ListByTeam(r.Context(), uint(teamID)); lerr == nil {
		for _, e := range existing {
			if strings.EqualFold(e.Email, inviteEmail) && e.Status == "pending" && time.Now().Before(e.ExpiresAt) {
				utils.WriteError(w, h.logger, nil, "an active invitation already exists for this user", http.StatusBadRequest)
				return
			}
		}
	} else {
		utils.WriteError(w, h.logger, lerr, "failed to check existing invitations", http.StatusInternalServerError)
		return
	}

	token := generateToken()
	inv := &db.TeamInvitation{
		TeamID:    uint(teamID),
		Email:     inviteEmail,
		Token:     token,
		Role:      req.Role,
		Status:    "pending",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := h.Connection.Invitations.Create(r.Context(), inv); err != nil {
		utils.WriteError(w, h.logger, err, "failed to create team invitation", http.StatusInternalServerError)
		return
	}

	payload := map[string]any{
		"team_id":   team.ID,
		"team_name": team.Name,
		"token":     token,
		"role":      req.Role,
	}
	if b, perr := json.Marshal(payload); perr == nil {
		_ = h.Connection.Notifications.Create(r.Context(), &db.Notification{
			UserID: u.ID,
			Type:   "team_invite",
			Data:   string(b),
		})
	}

	resp := struct {
		Token string `json:"token"`
	}{Token: token}
	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// POST /teams/invitations/accept
func (h *TeamsAPI) AcceptInvitationEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)

	var req struct {
		Token string `json:"token"`
	}
	if err := utils.ReadJSON(r.Body, w, h.logger, &req); err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}
	inv, err := h.Connection.Invitations.ByToken(r.Context(), req.Token)
	if err != nil {
		utils.WriteError(w, h.logger, err, "invitation not found", http.StatusNotFound)
		return
	}
	if inv.Status != "pending" || time.Now().After(inv.ExpiresAt) {
		utils.WriteError(w, h.logger, nil, "invitation expired or used", http.StatusBadRequest)
		return
	}
	if userObj.Email == nil || !strings.EqualFold(*userObj.Email, inv.Email) {
		utils.WriteError(w, h.logger, nil, "invitation email mismatch", http.StatusForbidden)
		return
	}

	err = h.Connection.WithTx(r.Context(), func(tx *db.Connection) error {
		if err := tx.Teams.AddMember(r.Context(), inv.TeamID, userObj.ID, inv.Role); err != nil {
			return err
		}
		return tx.Invitations.MarkAccepted(r.Context(), inv.ID)
	})
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to accept invitation", http.StatusInternalServerError)
		return
	}

	// Fetch team to include helpful context in the response
	team, terr := h.Connection.Teams.ByID(r.Context(), inv.TeamID)
	if terr != nil {
		utils.WriteError(w, h.logger, terr, "failed to get team", http.StatusInternalServerError)
		return
	}

	utils.WriteSuccess(w, h.logger, map[string]any{
		"status":    "accepted",
		"team_id":   team.ID,
		"team_name": team.Name,
		"role":      inv.Role,
	}, http.StatusOK)
}

// POST /teams/invitations/check
func (h *TeamsAPI) CheckInvitationStatusEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)

	var req struct {
		Token string `json:"token"`
	}
	if err := utils.ReadJSON(r.Body, w, h.logger, &req); err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Token) == "" {
		utils.WriteError(w, h.logger, nil, "token is required", http.StatusBadRequest)
		return
	}

	inv, err := h.Connection.Invitations.ByToken(r.Context(), req.Token)
	if err != nil {
		utils.WriteError(w, h.logger, err, "invitation not found", http.StatusNotFound)
		return
	}

	if userObj.Email == nil || !strings.EqualFold(*userObj.Email, inv.Email) {
		utils.WriteError(w, h.logger, nil, "invitation email mismatch", http.StatusForbidden)
		return
	}

	team, terr := h.Connection.Teams.ByID(r.Context(), inv.TeamID)
	if terr != nil {
		utils.WriteError(w, h.logger, terr, "failed to get team", http.StatusInternalServerError)
		return
	}

	status := inv.Status
	if status == "pending" && time.Now().After(inv.ExpiresAt) {
		status = "expired"
	}

	resp := map[string]any{
		"status":     status,
		"team_id":    team.ID,
		"team_name":  team.Name,
		"role":       inv.Role,
		"expires_at": inv.ExpiresAt,
	}
	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

func isAllowedIcon(icon string) bool {
	allowed := []string{"briefcase", "building", "bolt", "beaker", "book", "calendar", "chart", "code", "compass", "cpu", "database"}
	for _, a := range allowed {
		if a == icon {
			return true
		}
	}
	return icon == ""
}

func generateToken() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// GET /teams/{id}/overview
func (h *TeamsAPI) GetTeamOverviewEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid team ID", http.StatusBadRequest)
		return
	}

	team, err := h.Connection.Teams.ByID(r.Context(), uint(teamID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get team", http.StatusInternalServerError)
		return
	}

	var hasAccess bool
	for _, user := range team.Users {
		if user.ID == userID {
			hasAccess = true
			break
		}
	}
	if !hasAccess {
		h.logger.Warn("user does not have access to team", "team_id", teamID, "user_id", userID)
		utils.WriteError(w, h.logger, nil, "team not found", http.StatusNotFound)
		return
	}

	stats := map[string]any{
		"members_count": len(team.Users),
	}

	resp := map[string]any{
		"team": map[string]any{
			"id":   team.ID,
			"name": team.Name,
			"icon": team.Icon,
		},
		"stats": stats,
	}

	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

func NewTeamsAPI(logger logger.MultiLogger, connection *db.Connection) *TeamsAPI {
	return &TeamsAPI{logger: logger, Connection: connection}
}

// GET /teams
func (h *TeamsAPI) GetTeamsEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)

	teams, err := h.Connection.Teams.ListForUser(r.Context(), userObj.ID)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get teams", http.StatusInternalServerError)
		return
	}

	type respPart struct {
		ID      uint   `json:"id"`
		Name    string `json:"name"`
		Icon    string `json:"icon"`
		OwnerID int    `json:"owner_id"`
	}

	var resp []respPart

	for _, ws := range teams {
		resp = append(resp, respPart{
			ID:      ws.ID,
			Name:    ws.Name,
			Icon:    ws.Icon,
			OwnerID: int(ws.OwnerID),
		})
	}
	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// POST /teams
func (h *TeamsAPI) CreateTeamEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)

	var req struct {
		Name string `json:"name"`
		Icon string `json:"icon"`
	}
	err := utils.ReadJSON(r.Body, w, h.logger, &req)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}

	if req.Icon != "" && !isAllowedIcon(req.Icon) {
		utils.WriteError(w, h.logger, nil, "invalid icon", http.StatusBadRequest)
		return
	}

	ws := &db.Team{
		Name:    req.Name,
		Icon:    req.Icon,
		OwnerID: userObj.ID,
		Owner:   *userObj,
		Users:   []db.User{*userObj},
	}

	err = h.Connection.Teams.Create(r.Context(), ws)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to create team", http.StatusInternalServerError)
		return
	}

	_ = h.Connection.Teams.AddMember(r.Context(), ws.ID, userObj.ID, "admin")

	resp := struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
		Icon string `json:"icon"`
		Role string `json:"role"`
	}{
		ID:   ws.ID,
		Name: ws.Name,
		Icon: ws.Icon,
		Role: "admin",
	}

	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// GET /teams/{id}
func (h *TeamsAPI) GetTeamEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid team ID", http.StatusBadRequest)
		return
	}

	team, err := h.Connection.Teams.ByID(r.Context(), uint(teamID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get team", http.StatusInternalServerError)
		return
	}

	var hasAccess bool

	members := []struct {
		ID    uint   `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Role  string `json:"role"`
	}{}

	rolesMap, _ := h.Connection.Teams.RolesForTeam(r.Context(), uint(teamID))
	for _, user := range team.Users {
		if user.ID == userID {
			hasAccess = true
		}
		role := rolesMap[user.ID]
		if role == "" {
			role = "regular"
		}
		var nameVal string
		if user.Name != nil {
			nameVal = *user.Name
		}
		var emailVal string
		if user.Email != nil {
			emailVal = *user.Email
		}
		members = append(members, struct {
			ID    uint   `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
			Role  string `json:"role"`
		}{
			ID:    user.ID,
			Name:  nameVal,
			Email: emailVal,
			Role:  role,
		})
	}

	if !hasAccess {
		h.logger.Warn("user does not have access to team", "team_id", teamID, "user_id", userID)
		utils.WriteError(w, h.logger, nil, "team not found", http.StatusNotFound)
		return
	}

	resp := struct {
		ID      uint   `json:"id"`
		Name    string `json:"name"`
		Icon    string `json:"icon"`
		OwnerID int    `json:"owner_id"`
		Members []struct {
			ID    uint   `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"members"`
	}{
		ID:      team.ID,
		Name:    team.Name,
		Icon:    team.Icon,
		OwnerID: int(team.OwnerID),
		Members: members,
	}

	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// PATCH /teams/{id}
func (h *TeamsAPI) UpdateTeamNameEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid team ID", http.StatusBadRequest)
		return
	}

	team, err := h.Connection.Teams.ByID(r.Context(), uint(teamID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get team", http.StatusInternalServerError)
		return
	}

	rolesMap, _ := h.Connection.Teams.RolesForTeam(r.Context(), uint(teamID))
	if team.OwnerID != userID && rolesMap[userID] != "admin" {
		h.logger.Warn("user does not have access to team", "team_id", teamID, "user_id", userID)
		utils.WriteError(w, h.logger, nil, "user does not have access to change the team info", http.StatusForbidden)
		return
	}

	var req struct {
		Name string `json:"name"`
		Icon string `json:"icon"`
	}

	err = utils.ReadJSON(r.Body, w, h.logger, &req)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}

	if req.Name != "" {
		team.Name = req.Name
	}
	if req.Icon != "" {
		if !isAllowedIcon(req.Icon) {
			utils.WriteError(w, h.logger, nil, "invalid icon", http.StatusBadRequest)
			return
		}
		team.Icon = req.Icon
	}

	err = h.Connection.Teams.Update(r.Context(), team)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to update team", http.StatusInternalServerError)
		return
	}

	resp := struct {
		Success bool   `json:"success"`
		Status  string `json:"status"`
	}{
		Status:  "success",
		Success: true,
	}

	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// DELETE /teams/{id}
func (h *TeamsAPI) DeleteTeamEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid team ID", http.StatusBadRequest)
		return
	}

	team, err := h.Connection.Teams.ByID(r.Context(), uint(teamID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get team", http.StatusInternalServerError)
		return
	}

	if team.OwnerID != userID {
		h.logger.Warn("user does not have access to team", "team_id", teamID, "user_id", userID)
		utils.WriteError(w, h.logger, nil, "user does not have access to delete the team", http.StatusForbidden)
		return
	}

	err = h.Connection.Teams.Delete(r.Context(), team)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to delete team", http.StatusInternalServerError)
		return
	}

	resp := struct {
		Success bool   `json:"success"`
		Status  string `json:"status"`
	}{
		Status:  "success",
		Success: true,
	}

	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// POST /teams/{id}/members
func (h *TeamsAPI) AddMemberEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid team ID", http.StatusBadRequest)
		return
	}

	team, err := h.Connection.Teams.ByID(r.Context(), uint(teamID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get team", http.StatusInternalServerError)
		return
	}

	rolesMap, _ := h.Connection.Teams.RolesForTeam(r.Context(), uint(teamID))
	if team.OwnerID != userID && rolesMap[userID] != "admin" {
		h.logger.Warn("user does not have access to team", "team_id", teamID, "user_id", userID)
		utils.WriteError(w, h.logger, nil, "user does not have access to add a member to the team", http.StatusForbidden)
		return
	}

	var req struct {
		UserID uint   `json:"user_id"`
		Role   string `json:"role"`
	}

	err = utils.ReadJSON(r.Body, w, h.logger, &req)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}

	if req.Role != "regular" && req.Role != "admin" {
		utils.WriteError(w, h.logger, fmt.Errorf("invalid role: %s", req.Role), "invalid role", http.StatusBadRequest)
		return
	}

	err = h.Connection.Teams.AddMember(r.Context(), uint(teamID), req.UserID, req.Role)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to add member to team", http.StatusInternalServerError)
		return
	}

	resp := struct {
		Success bool   `json:"success"`
		Status  string `json:"status"`
	}{
		Status:  "success",
		Success: true,
	}

	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// DELETE /teams/{id}/members/{user_id}
func (h *TeamsAPI) RemoveMemberEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid team ID", http.StatusBadRequest)
		return
	}

	team, err := h.Connection.Teams.ByID(r.Context(), uint(teamID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get team", http.StatusInternalServerError)
		return
	}

	rolesMap, _ := h.Connection.Teams.RolesForTeam(r.Context(), uint(teamID))
	if team.OwnerID != userID && rolesMap[userID] != "admin" {
		h.logger.Warn("user does not have access to team", "team_id", teamID, "user_id", userID)
		utils.WriteError(w, h.logger, nil, "user does not have access to remove a member from the team", http.StatusForbidden)
		return
	}

	memberID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid member ID", http.StatusBadRequest)
		return
	}

	if uint(memberID) == team.OwnerID {
		utils.WriteError(w, h.logger, nil, "cannot remove team owner", http.StatusBadRequest)
		return
	}
	err = h.Connection.Teams.RemoveMember(r.Context(), uint(teamID), uint(memberID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to remove member from team", http.StatusInternalServerError)
		return
	}

	resp := struct {
		Success bool   `json:"success"`
		Status  string `json:"status"`
	}{
		Status:  "success",
		Success: true,
	}

	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

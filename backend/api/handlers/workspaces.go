package handlers

import (
	"encoding/json"
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
)

type WorkspacesAPI struct {
	logger     logger.MultiLogger
	Connection *db.Connection
}

// PATCH /workspaces/{id}/members/{user_id}/role
func (h *WorkspacesAPI) UpdateMemberRoleEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	workspaceID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid workspace ID", http.StatusBadRequest)
		return
	}

	workspace, err := h.Connection.Workspaces.ByID(r.Context(), uint(workspaceID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get workspace", http.StatusInternalServerError)
		return
	}

	if workspace.OwnerID != userID {
		utils.WriteError(w, h.logger, nil, "user does not have access to change member roles", http.StatusForbidden)
		return
	}

	memberID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid member ID", http.StatusBadRequest)
		return
	}

	if uint(memberID) == workspace.OwnerID {
		utils.WriteError(w, h.logger, nil, "cannot change role of the workspace owner", http.StatusBadRequest)
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

	// Use upsert to set role
	if err := h.Connection.Workspaces.AddMember(r.Context(), uint(workspaceID), uint(memberID), req.Role); err != nil {
		utils.WriteError(w, h.logger, err, "failed to update member role", http.StatusInternalServerError)
		return
	}

	utils.WriteSuccess(w, h.logger, map[string]any{"status": "updated"}, http.StatusOK)
}

// GET /workspaces/{id}/invitations
func (h *WorkspacesAPI) ListInvitationsEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	workspaceID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid workspace ID", http.StatusBadRequest)
		return
	}
	workspace, err := h.Connection.Workspaces.ByID(r.Context(), uint(workspaceID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get workspace", http.StatusInternalServerError)
		return
	}
	rolesMap, _ := h.Connection.Workspaces.RolesForWorkspace(r.Context(), uint(workspaceID))
	role := rolesMap[userObj.ID]
	if workspace.OwnerID != userObj.ID && role != "admin" {
		utils.WriteError(w, h.logger, nil, "forbidden", http.StatusForbidden)
		return
	}
	list, err := h.Connection.Invitations.ListByWorkspace(r.Context(), uint(workspaceID))
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

// DELETE /workspaces/{id}/invitations/{inv_id}
func (h *WorkspacesAPI) RevokeInvitationEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	workspaceID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid workspace ID", http.StatusBadRequest)
		return
	}
	workspace, err := h.Connection.Workspaces.ByID(r.Context(), uint(workspaceID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get workspace", http.StatusInternalServerError)
		return
	}
	rolesMap, _ := h.Connection.Workspaces.RolesForWorkspace(r.Context(), uint(workspaceID))
	role := rolesMap[userObj.ID]
	if workspace.OwnerID != userObj.ID && role != "admin" {
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

// POST /workspaces/{id}/invitations
func (h *WorkspacesAPI) CreateInvitationEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)

	workspaceID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid workspace ID", http.StatusBadRequest)
		return
	}

	workspace, err := h.Connection.Workspaces.ByID(r.Context(), uint(workspaceID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get workspace", http.StatusInternalServerError)
		return
	}

	rolesMap, _ := h.Connection.Workspaces.RolesForWorkspace(r.Context(), uint(workspaceID))
	role := rolesMap[userObj.ID]
	if workspace.OwnerID != userObj.ID && role != "admin" {
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

	token := generateToken()
	inv := &db.WorkspaceInvitation{
		WorkspaceID: uint(workspaceID),
		Email:       strings.ToLower(req.Email),
		Token:       token,
		Role:        req.Role,
		Status:      "pending",
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
	}
	if err := h.Connection.Invitations.Create(r.Context(), inv); err != nil {
		utils.WriteError(w, h.logger, err, "failed to create invitation", http.StatusInternalServerError)
		return
	}

	// if user with this email already exists create a notification for them
	if u, err := h.Connection.Users.ByEmail(r.Context(), strings.ToLower(req.Email)); err == nil && u != nil {
		payload := map[string]any{
			"workspace_id":   workspace.ID,
			"workspace_name": workspace.Name,
			"token":          token,
			"role":           req.Role,
		}
		if b, perr := json.Marshal(payload); perr == nil {
			_ = h.Connection.Notifications.Create(r.Context(), &db.Notification{
				UserID: u.ID,
				Type:   "workspace_invite",
				Data:   string(b),
			})
		}
	}

	resp := struct {
		Token string `json:"token"`
	}{Token: token}
	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// POST /workspaces/invitations/accept
func (h *WorkspacesAPI) AcceptInvitationEndpoint(w http.ResponseWriter, r *http.Request) {
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
		if err := tx.Workspaces.AddMember(r.Context(), inv.WorkspaceID, userObj.ID, inv.Role); err != nil {
			return err
		}
		return tx.Invitations.MarkAccepted(r.Context(), inv.ID)
	})
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to accept invitation", http.StatusInternalServerError)
		return
	}

	utils.WriteSuccess(w, h.logger, map[string]any{"status": "accepted"}, http.StatusOK)
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

// GET /workspaces/{id}/overview
func (h *WorkspacesAPI) GetWorkspaceOverviewEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	workspaceID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid workspace ID", http.StatusBadRequest)
		return
	}

	workspace, err := h.Connection.Workspaces.ByID(r.Context(), uint(workspaceID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get workspace", http.StatusInternalServerError)
		return
	}

	var hasAccess bool
	for _, user := range workspace.Users {
		if user.ID == userID {
			hasAccess = true
			break
		}
	}
	if !hasAccess {
		h.logger.Warn("user does not have access to workspace", "workspace_id", workspaceID, "user_id", userID)
		utils.WriteError(w, h.logger, nil, "workspace not found", http.StatusNotFound)
		return
	}

	// Minimal per-workspace stats (extend as needed)
	stats := map[string]any{
		"members_count": len(workspace.Users),
	}

	resp := map[string]any{
		"workspace": map[string]any{
			"id":   workspace.ID,
			"name": workspace.Name,
			"icon": workspace.Icon,
		},
		"stats": stats,
	}

	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

func NewWorkspacesAPI(logger logger.MultiLogger, connection *db.Connection) *WorkspacesAPI {
	return &WorkspacesAPI{logger: logger, Connection: connection}
}

// GET /workspaces
func (h *WorkspacesAPI) GetWorkspacesEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)

	workspaces, err := h.Connection.Workspaces.ListForUser(r.Context(), userObj.ID)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get workspaces", http.StatusInternalServerError)
		return
	}

	type respPart struct {
		ID      uint   `json:"id"`
		Name    string `json:"name"`
		Icon    string `json:"icon"`
		OwnerID int    `json:"owner_id"`
	}

	var resp []respPart

	for _, ws := range workspaces {
		resp = append(resp, respPart{
			ID:      ws.ID,
			Name:    ws.Name,
			Icon:    ws.Icon,
			OwnerID: int(ws.OwnerID),
		})
	}
	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// POST /workspaces
func (h *WorkspacesAPI) CreateWorkspaceEndpoint(w http.ResponseWriter, r *http.Request) {
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

	ws := &db.WorkSpace{
		Name:    req.Name,
		Icon:    req.Icon,
		OwnerID: userObj.ID,
		Owner:   *userObj,
		Users:   []db.User{*userObj},
	}

	err = h.Connection.Workspaces.Create(r.Context(), ws)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to create workspace", http.StatusInternalServerError)
		return
	}

	_ = h.Connection.Workspaces.AddMember(r.Context(), ws.ID, userObj.ID, "admin")

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

// GET /workspaces/{id}
func (h *WorkspacesAPI) GetWorkspaceEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	workspaceID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid workspace ID", http.StatusBadRequest)
		return
	}

	workspace, err := h.Connection.Workspaces.ByID(r.Context(), uint(workspaceID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get workspace", http.StatusInternalServerError)
		return
	}

	var hasAccess bool

	members := []struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
		Role string `json:"role"`
	}{}

	rolesMap, _ := h.Connection.Workspaces.RolesForWorkspace(r.Context(), uint(workspaceID))
	for _, user := range workspace.Users {
		if user.ID == userID {
			hasAccess = true
		}
		role := rolesMap[user.ID]
		if role == "" {
			role = "regular"
		}
		members = append(members, struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
			Role string `json:"role"`
		}{
			ID:   user.ID,
			Name: *user.Name,
			Role: role,
		})
	}

	if !hasAccess {
		h.logger.Warn("user does not have access to workspace", "workspace_id", workspaceID, "user_id", userID)
		utils.WriteError(w, h.logger, nil, "workspace not found", http.StatusNotFound)
		return
	}

	resp := struct {
		ID      uint   `json:"id"`
		Name    string `json:"name"`
		Icon    string `json:"icon"`
		OwnerID int    `json:"owner_id"`
		Members []struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
			Role string `json:"role"`
		} `json:"members"`
	}{
		ID:      workspace.ID,
		Name:    workspace.Name,
		Icon:    workspace.Icon,
		OwnerID: int(workspace.OwnerID),
		Members: members,
	}

	utils.WriteSuccess(w, h.logger, resp, http.StatusOK)
}

// PATCH /workspaces/{id}
func (h *WorkspacesAPI) UpdateWorkspaceNameEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	workspaceID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid workspace ID", http.StatusBadRequest)
		return
	}

	workspace, err := h.Connection.Workspaces.ByID(r.Context(), uint(workspaceID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get workspace", http.StatusInternalServerError)
		return
	}

	rolesMap, _ := h.Connection.Workspaces.RolesForWorkspace(r.Context(), uint(workspaceID))
	if workspace.OwnerID != userID && rolesMap[userID] != "admin" {
		h.logger.Warn("user does not have access to workspace", "workspace_id", workspaceID, "user_id", userID)
		utils.WriteError(w, h.logger, nil, "user does not have access to change the workspace info", http.StatusForbidden)
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
		workspace.Name = req.Name
	}
	if req.Icon != "" {
		if !isAllowedIcon(req.Icon) {
			utils.WriteError(w, h.logger, nil, "invalid icon", http.StatusBadRequest)
			return
		}
		workspace.Icon = req.Icon
	}

	err = h.Connection.Workspaces.Update(r.Context(), workspace)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to update workspace", http.StatusInternalServerError)
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

// DELETE /workspaces/{id}
func (h *WorkspacesAPI) DeleteWorkspaceEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	workspaceID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid workspace ID", http.StatusBadRequest)
		return
	}

	workspace, err := h.Connection.Workspaces.ByID(r.Context(), uint(workspaceID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get workspace", http.StatusInternalServerError)
		return
	}

	if workspace.OwnerID != userID {
		h.logger.Warn("user does not have access to workspace", "workspace_id", workspaceID, "user_id", userID)
		utils.WriteError(w, h.logger, nil, "user does not have access to delete the workspace", http.StatusForbidden)
		return
	}

	err = h.Connection.Workspaces.Delete(r.Context(), workspace)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to delete workspace", http.StatusInternalServerError)
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

// POST /workspaces/{id}/members
func (h *WorkspacesAPI) AddMemberEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	workspaceID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid workspace ID", http.StatusBadRequest)
		return
	}

	workspace, err := h.Connection.Workspaces.ByID(r.Context(), uint(workspaceID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get workspace", http.StatusInternalServerError)
		return
	}

	rolesMap, _ := h.Connection.Workspaces.RolesForWorkspace(r.Context(), uint(workspaceID))
	if workspace.OwnerID != userID && rolesMap[userID] != "admin" {
		h.logger.Warn("user does not have access to workspace", "workspace_id", workspaceID, "user_id", userID)
		utils.WriteError(w, h.logger, nil, "user does not have access to add a member to the workspace", http.StatusForbidden)
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

	err = h.Connection.Workspaces.AddMember(r.Context(), uint(workspaceID), req.UserID, req.Role)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to add member to workspace", http.StatusInternalServerError)
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

// DELETE /workspaces/{id}/members/{user_id}
func (h *WorkspacesAPI) RemoveMemberEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(middleware.UserObjectContextKey).(*db.User)
	userID := userObj.ID

	workspaceID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid workspace ID", http.StatusBadRequest)
		return
	}

	workspace, err := h.Connection.Workspaces.ByID(r.Context(), uint(workspaceID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to get workspace", http.StatusInternalServerError)
		return
	}

	rolesMap, _ := h.Connection.Workspaces.RolesForWorkspace(r.Context(), uint(workspaceID))
	if workspace.OwnerID != userID && rolesMap[userID] != "admin" {
		h.logger.Warn("user does not have access to workspace", "workspace_id", workspaceID, "user_id", userID)
		utils.WriteError(w, h.logger, nil, "user does not have access to remove a member from the workspace", http.StatusForbidden)
		return
	}

	memberID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid member ID", http.StatusBadRequest)
		return
	}

	if uint(memberID) == workspace.OwnerID {
		utils.WriteError(w, h.logger, nil, "cannot remove workspace owner", http.StatusBadRequest)
		return
	}
	err = h.Connection.Workspaces.RemoveMember(r.Context(), uint(workspaceID), uint(memberID))
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to remove member from workspace", http.StatusInternalServerError)
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

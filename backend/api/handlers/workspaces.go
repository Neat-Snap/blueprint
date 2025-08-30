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

type WorkspacesAPI struct {
	logger     logger.MultiLogger
	Connection *db.Connection
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
		OwnerID int    `json:"owner_id"`
	}

	var resp []respPart

	for _, ws := range workspaces {
		resp = append(resp, respPart{
			ID:      ws.ID,
			Name:    ws.Name,
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
	}

	err := utils.ReadJSON(r.Body, w, h.logger, &req)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}

	ws := &db.WorkSpace{
		Name:    req.Name,
		OwnerID: userObj.ID,
		Owner:   *userObj,
		Users:   []db.User{*userObj},
	}

	err = h.Connection.Workspaces.Create(r.Context(), ws)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to create workspace", http.StatusInternalServerError)
		return
	}

	_ = h.Connection.Workspaces.AddMember(r.Context(), ws.ID, userObj.ID, "owner")

	resp := struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
		Role string `json:"role"`
	}{
		ID:   ws.ID,
		Name: req.Name,
		Role: "owner",
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

	for _, user := range workspace.Users {
		if user.ID == userID {
			hasAccess = true
		}
		members = append(members, struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
			Role string `json:"role"`
		}{
			ID:   user.ID,
			Name: *user.Name,
			Role: "member",
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
		OwnerID int    `json:"owner_id"`
		Members []struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
			Role string `json:"role"`
		} `json:"members"`
	}{
		ID:      workspace.ID,
		Name:    workspace.Name,
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

	if workspace.OwnerID != userID {
		h.logger.Warn("user does not have access to workspace", "workspace_id", workspaceID, "user_id", userID)
		utils.WriteError(w, h.logger, nil, "user does not have access to change the workspace info", http.StatusForbidden)
		return
	}

	var req struct {
		Name string `json:"name"`
	}

	err = utils.ReadJSON(r.Body, w, h.logger, &req)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}

	workspace.Name = req.Name

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

	if workspace.OwnerID != userID {
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

	if req.Role != "owner" && req.Role != "member" {
		utils.WriteError(w, h.logger, nil, "invalid role", http.StatusBadRequest)
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

	if workspace.OwnerID != userID {
		h.logger.Warn("user does not have access to workspace", "workspace_id", workspaceID, "user_id", userID)
		utils.WriteError(w, h.logger, nil, "user does not have access to remove a member from the workspace", http.StatusForbidden)
		return
	}

	memberID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil {
		utils.WriteError(w, h.logger, err, "invalid member ID", http.StatusBadRequest)
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

// POST /workspaces/{id}/owner
func (h *WorkspacesAPI) ReassignOwnerEndpoint(w http.ResponseWriter, r *http.Request) {
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
		utils.WriteError(w, h.logger, nil, "user does not have access to reassign the owner of the workspace", http.StatusForbidden)
		return
	}

	var req struct {
		UserID uint `json:"user_id"`
	}

	err = utils.ReadJSON(r.Body, w, h.logger, &req)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}

	err = h.Connection.Workspaces.ReassignOwner(r.Context(), uint(workspaceID), req.UserID)
	if err != nil {
		utils.WriteError(w, h.logger, err, "failed to reassign owner of workspace", http.StatusInternalServerError)
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

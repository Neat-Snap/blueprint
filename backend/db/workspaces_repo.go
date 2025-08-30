package db

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type workspacesRepo struct{ db *gorm.DB }

func (r *workspacesRepo) Create(ctx context.Context, w *WorkSpace) error {
	// Ensure OwnerID set per your non-null rule, else return error
	return r.db.WithContext(ctx).Create(w).Error
}

func (r *workspacesRepo) ByID(ctx context.Context, id uint) (*WorkSpace, error) {
	var w WorkSpace
	err := r.db.WithContext(ctx).
		Preload("Owner").
		Preload("Users").
		First(&w, id).Error
	return &w, err
}

func (r *workspacesRepo) AddMember(ctx context.Context, workspaceID, userID uint, role string) error {
	uw := UserWorkspace{UserID: userID, WorkspaceID: workspaceID, Role: role}
	// Upsert to avoid duplicate memberships
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "workspace_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"role": role}),
		}).
		Create(&uw).Error
}

func (r *workspacesRepo) RemoveMember(ctx context.Context, workspaceID, userID uint) error {
	return r.db.WithContext(ctx).Where("user_id=? AND workspace_id=?", userID, workspaceID).
		Delete(&UserWorkspace{}).Error
}

func (r *workspacesRepo) ListForUser(ctx context.Context, userID uint) ([]WorkSpace, error) {
	var ws []WorkSpace
	err := r.db.WithContext(ctx).
		Joins("JOIN user_workspaces uw ON uw.workspace_id = workspaces.id").
		Where("uw.user_id = ?", userID).
		Preload("Owner").
		Find(&ws).Error
	return ws, err
}

func (r *workspacesRepo) ReassignOwner(ctx context.Context, workspaceID, newOwnerID uint) error {
    return r.db.WithContext(ctx).
        Model(&WorkSpace{}).
        Where("id = ?", workspaceID).
        Update("owner_id", newOwnerID).Error
}

func (r *workspacesRepo) GetUserRole(ctx context.Context, workspaceID, userID uint) (string, error) {
    var uw UserWorkspace
    err := r.db.WithContext(ctx).
        Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
        First(&uw).Error
    if err != nil {
        return "", err
    }
    return uw.Role, nil
}

func (r *workspacesRepo) RolesForWorkspace(ctx context.Context, workspaceID uint) (map[uint]string, error) {
    var uws []UserWorkspace
    if err := r.db.WithContext(ctx).Where("workspace_id = ?", workspaceID).Find(&uws).Error; err != nil {
        return nil, err
    }
    res := make(map[uint]string, len(uws))
    for _, uw := range uws {
        res[uw.UserID] = uw.Role
    }
    return res, nil
}

func (r *workspacesRepo) Update(ctx context.Context, w *WorkSpace) error {
	return r.db.WithContext(ctx).Save(w).Error
}

func (r *workspacesRepo) Delete(ctx context.Context, w *WorkSpace) error {
	return r.db.WithContext(ctx).Delete(w).Error
}

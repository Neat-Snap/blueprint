package db

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type invitationsRepo struct{ db *gorm.DB }

func (r *invitationsRepo) Create(ctx context.Context, inv *WorkspaceInvitation) error {
	return r.db.WithContext(ctx).Create(inv).Error
}

func (r *invitationsRepo) ByToken(ctx context.Context, token string) (*WorkspaceInvitation, error) {
	var i WorkspaceInvitation
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&i).Error
	return &i, err
}

func (r *invitationsRepo) MarkAccepted(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&WorkspaceInvitation{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     "accepted",
			"updated_at": time.Now(),
		}).Error
}

func (r *invitationsRepo) ListByWorkspace(ctx context.Context, wsID uint) ([]WorkspaceInvitation, error) {
	var list []WorkspaceInvitation
	err := r.db.WithContext(ctx).Where("workspace_id = ? AND status = ?", wsID, "pending").Order("created_at DESC").Find(&list).Error
	return list, err
}

func (r *invitationsRepo) Revoke(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&WorkspaceInvitation{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     "revoked",
			"updated_at": time.Now(),
		}).Error
}

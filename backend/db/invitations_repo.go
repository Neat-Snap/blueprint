package db

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type invitationsRepo struct{ db *gorm.DB }

func (r *invitationsRepo) Create(ctx context.Context, inv *TeamInvitation) error {
	return r.db.WithContext(ctx).Create(inv).Error
}

func (r *invitationsRepo) ByToken(ctx context.Context, token string) (*TeamInvitation, error) {
	var i TeamInvitation
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&i).Error
	return &i, err
}

func (r *invitationsRepo) MarkAccepted(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&TeamInvitation{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     "accepted",
			"updated_at": time.Now(),
		}).Error
}

func (r *invitationsRepo) ListByTeam(ctx context.Context, teamID uint) ([]TeamInvitation, error) {
	var list []TeamInvitation
	err := r.db.WithContext(ctx).Where("team_id = ? AND status = ?", teamID, "pending").Order("created_at DESC").Find(&list).Error
	return list, err
}

func (r *invitationsRepo) Revoke(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&TeamInvitation{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     "revoked",
			"updated_at": time.Now(),
		}).Error
}

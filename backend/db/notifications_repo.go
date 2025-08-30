package db

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type notificationsRepo struct{ db *gorm.DB }

func (r *notificationsRepo) Create(ctx context.Context, n *Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *notificationsRepo) ListForUser(ctx context.Context, userID uint) ([]Notification, error) {
	var list []Notification
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&list).Error
	return list, err
}

func (r *notificationsRepo) MarkRead(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&Notification{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"read_at":    &now,
			"updated_at": now,
		}).Error
}

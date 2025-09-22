package db

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type authRepo struct{ db *gorm.DB }

func (r *authRepo) CreateSession(ctx context.Context, session *UserSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *authRepo) FindSessionByID(ctx context.Context, sessionID string) (*UserSession, error) {
	var session UserSession
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("session_id = ?", sessionID).
		First(&session).Error
	return &session, err
}

func (r *authRepo) TouchSession(ctx context.Context, sessionID string, lastUsedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&UserSession{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]any{"last_used_at": lastUsedAt}).
		Error
}

func (r *authRepo) UpdateSessionTokens(ctx context.Context, sessionID, refreshHash string, expiresAt time.Time, lastUsed time.Time) error {
	return r.db.WithContext(ctx).
		Model(&UserSession{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]any{
			"refresh_token_hash": refreshHash,
			"expires_at":         expiresAt,
			"last_used_at":       lastUsed,
		}).
		Error
}

func (r *authRepo) DeleteSessionByID(ctx context.Context, sessionID string) error {
	return r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Delete(&UserSession{}).
		Error
}

func (r *authRepo) DeleteSessionsForUser(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&UserSession{}).
		Error
}

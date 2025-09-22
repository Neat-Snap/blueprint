package db

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type authRepo struct{ db *gorm.DB }

func (r *authRepo) FindAuthIdentity(ctx context.Context, provider, subject string) (*AuthIdentity, error) {
	var ai AuthIdentity
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("provider = ? AND subject = ?", provider, subject).
		First(&ai).Error
	return &ai, err
}

func (r *authRepo) FindUserByAuthIdentity(ctx context.Context, ai *AuthIdentity) (*User, error) {
	userID := ai.UserID
	var u User
	err := r.db.WithContext(ctx).
		Preload("PasswordCredential").
		Preload("AuthIdentities").
		First(&u, userID).Error
	return &u, err
}

func (r *authRepo) LinkIdentity(ctx context.Context, userID uint, provider, subject string, providerEmail, accessToken, refreshToken *string) error {
	ai := AuthIdentity{
		UserID: userID, Provider: provider, Subject: subject, ProviderEmail: providerEmail, AccessToken: accessToken, RefreshToken: refreshToken,
	}
	// rely on UNIQUE(provider,subject)
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true, Columns: []clause.Column{{Name: "provider"}, {Name: "subject"}}}).
		Create(&ai).Error
}

func (r *authRepo) EnsurePasswordCredential(ctx context.Context, userID uint, hashed string) error {
	pc := PasswordCredential{UserID: userID, PasswordHash: hashed}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"password_hash": hashed}),
		}).
		Create(&pc).Error
}

func (r *authRepo) FindPasswordCredential(ctx context.Context, userID uint) (*PasswordCredential, error) {
	var pc PasswordCredential
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&pc).Error
	return &pc, err
}

func (r *authRepo) DeleteAuthIdentity(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&AuthIdentity{}).Error
}

package db

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// type UserPreferencesRepo interface {
// 	Get(ctx context.Context, userID uint) (*UserPreference, error)
// 	Update(ctx context.Context, userID uint, preference *UserPreference) error
// }

type preferencesRepo struct{ db *gorm.DB }

func (p *preferencesRepo) Create(ctx context.Context, userID uint) error {
	preference := UserPreference{UserID: userID, Theme: "system", Language: "en"}
	return p.db.WithContext(ctx).Create(&preference).Error
}

func (p *preferencesRepo) Get(ctx context.Context, userID uint) (*UserPreference, error) {
	var preferenceObj UserPreference
	err := p.db.WithContext(ctx).Where("user_id = ?", userID).First(&preferenceObj).Error
	if err != nil {
		return nil, err
	}

	return &preferenceObj, nil
}

func (p *preferencesRepo) GetByEmail(ctx context.Context, userEmail string) (*UserPreference, error) {
	if userEmail == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var u User
	err := p.db.WithContext(ctx).
		Where("LOWER(email) = ?", strings.ToLower(userEmail)).
		First(&u).Error

	if err != nil {
		return nil, err
	}

	var preference UserPreference
	err = p.db.WithContext(ctx).Where("user_id = ?", u.ID).First(&preference).Error
	return &preference, err
}

func (p *preferencesRepo) Update(ctx context.Context, preference *UserPreference) error {
	return p.db.WithContext(ctx).Save(preference).Error
}

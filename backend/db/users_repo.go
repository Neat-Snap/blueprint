package db

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

type usersRepo struct{ db *gorm.DB }

func (r *usersRepo) Create(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *usersRepo) ByID(ctx context.Context, id uint) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).
		First(&u, id).Error
	return &u, err
}

func (r *usersRepo) ByEmail(ctx context.Context, email string) (*User, error) {
	if email == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var u User
	err := r.db.WithContext(ctx).
		Where("LOWER(email) = ?", strings.ToLower(email)).
		First(&u).Error
	return &u, err
}

func (r *usersRepo) ByWorkOSID(ctx context.Context, workosID string) (*User, error) {
	if strings.TrimSpace(workosID) == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var u User
	err := r.db.WithContext(ctx).
		Where("work_os_user_id = ?", workosID).
		First(&u).Error
	return &u, err
}

func (r *usersRepo) Update(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Save(u).Error
}

func (r *usersRepo) SoftDelete(ctx context.Context, id uint) error {
	res := r.db.WithContext(ctx).Delete(&User{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

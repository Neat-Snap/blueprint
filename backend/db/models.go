package db

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Email           *string `gorm:"uniqueIndex:uniq_users_email,where:deleted_at IS NULL"`
	EmailVerifiedAt *time.Time

	Name      *string
	AvatarURL *string

	PasswordCredential *PasswordCredential `gorm:"constraint:OnDelete:CASCADE"`
	AuthIdentities     []AuthIdentity      `gorm:"constraint:OnDelete:CASCADE"`

	WorkSpaces []WorkSpace `gorm:"many2many:user_workspaces;joinForeignKey:UserID;joinReferences:WorkspaceID;constraint:OnDelete:CASCADE;"`
}

type PasswordCredential struct {
	ID                uint      `gorm:"primaryKey"`
	UserID            uint      `gorm:"uniqueIndex"`
	PasswordHash      string    `gorm:"type:text;not null" json:"-"`
	PasswordUpdatedAt time.Time `gorm:"autoUpdateTime"`
	PasswordDisabled  bool      `gorm:"default:false"`
}

type AuthIdentity struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	UserID   uint   `gorm:"index;not null"`
	Provider string `gorm:"type:varchar(32);not null;index:uniq_provider_subject,unique"`
	Subject  string `gorm:"type:varchar(191);not null;index:uniq_provider_subject,unique"`

	ProviderEmail *string
}

type WorkSpace struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Name string

	Users []User `gorm:"many2many:user_workspaces;joinForeignKey:WorkspaceID;joinReferences:UserID;constraint:OnDelete:CASCADE;"`

	Owner   User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	OwnerID uint `gorm:"index"`
}

func (WorkSpace) TableName() string { return "workspaces" }

type UserWorkspace struct {
	UserID      uint `gorm:"primaryKey;index"`
	WorkspaceID uint `gorm:"primaryKey;index"`

	Role string `gorm:"type:varchar(32);not null;default:'member'"`

	CreatedAt time.Time
}

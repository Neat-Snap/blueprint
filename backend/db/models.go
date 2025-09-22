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

	WorkOSUserID    string  `gorm:"type:varchar(191);uniqueIndex:uniq_users_workos_id,where:deleted_at IS NULL"`
	Email           *string `gorm:"uniqueIndex:uniq_users_email,where:deleted_at IS NULL"`
	EmailVerifiedAt *time.Time

	Name      *string
	AvatarURL *string

	Teams []Team `gorm:"many2many:user_teams;joinForeignKey:UserID;joinReferences:TeamID;constraint:OnDelete:CASCADE;"`
}

type UserPreference struct {
	ID uint `gorm:"primaryKey"`

	UserID uint  `gorm:"index;not null"`
	User   *User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	Theme    string `gorm:"type:varchar(32);not null;default:'system'"`
	Language string `gorm:"type:varchar(32);not null;default:'en'"`
}

type Notification struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	UserID uint  `gorm:"index;not null"`
	User   *User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	// type examples: "team_invite"
	Type string `gorm:"type:varchar(64);not null"`
	// json payload
	Data string `gorm:"type:text;not null"`

	ReadAt *time.Time `gorm:"index"`
}

type Team struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Name string
	Icon string `gorm:"type:varchar(64);default:''"`

	Users []User `gorm:"many2many:user_teams;joinForeignKey:TeamID;joinReferences:UserID;constraint:OnDelete:CASCADE;"`

	Owner   User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	OwnerID uint `gorm:"index"`
}

func (Team) TableName() string { return "teams" }

type UserTeam struct {
	UserID uint `gorm:"primaryKey;index"`
	TeamID uint `gorm:"primaryKey;index"`

	Role string `gorm:"type:varchar(32);not null;default:'regular'"`

	CreatedAt time.Time
}

type TeamInvitation struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	TeamID uint  `gorm:"index;not null"`
	Team   *Team `gorm:"foreignKey:TeamID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	Email     string    `gorm:"type:varchar(191);index;not null"`
	Token     string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	Role      string    `gorm:"type:varchar(32);not null;default:'regular'"`
	Status    string    `gorm:"type:varchar(32);not null;default:'pending'"`
	ExpiresAt time.Time `gorm:"index"`
}

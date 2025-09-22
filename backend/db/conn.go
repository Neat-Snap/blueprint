package db

import (
	"context"

	"gorm.io/gorm"
)

type Connection struct {
	DBConn        *gorm.DB
	Users         UsersRepo
	Teams         TeamsRepo
	Auth          AuthRepo
	Invitations   InvitationsRepo
	Notifications NotificationsRepo
	Preferences   UserPreferencesRepo
}

func NewConnection(db *gorm.DB) *Connection {
	return &Connection{
		DBConn:        db,
		Users:         &usersRepo{db: db},
		Teams:         &teamsRepo{db: db},
		Auth:          &authRepo{db: db},
		Invitations:   &invitationsRepo{db: db},
		Notifications: &notificationsRepo{db: db},
		Preferences:   &preferencesRepo{db: db},
	}
}

func (c *Connection) WithTx(ctx context.Context, fn func(tx *Connection) error) error {
	return c.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		localConn := &Connection{
			DBConn:        tx,
			Users:         &usersRepo{db: tx},
			Teams:         &teamsRepo{db: tx},
			Auth:          &authRepo{db: tx},
			Invitations:   &invitationsRepo{db: tx},
			Notifications: &notificationsRepo{db: tx},
			Preferences:   &preferencesRepo{db: tx},
		}
		return fn(localConn)
	})
}

type UsersRepo interface {
	Create(ctx context.Context, u *User) error
	ByID(ctx context.Context, id uint) (*User, error)
	ByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, u *User) error
	SoftDelete(ctx context.Context, id uint) error
}

type TeamsRepo interface {
	Create(ctx context.Context, w *Team) error
	ByID(ctx context.Context, id uint) (*Team, error)
	AddMember(ctx context.Context, teamID, userID uint, role string) error
	RemoveMember(ctx context.Context, teamID, userID uint) error
	ListForUser(ctx context.Context, userID uint) ([]Team, error)
	ReassignOwner(ctx context.Context, teamID, newOwnerID uint) error
	GetUserRole(ctx context.Context, teamID, userID uint) (string, error)
	RolesForTeam(ctx context.Context, teamID uint) (map[uint]string, error)
	Update(ctx context.Context, w *Team) error
	Delete(ctx context.Context, w *Team) error
}

type AuthRepo interface {
	FindAuthIdentity(ctx context.Context, provider, subject string) (*AuthIdentity, error)
	LinkIdentity(ctx context.Context, userID uint, provider, subject string, providerEmail, accessToken, refreshToken *string) error
	EnsurePasswordCredential(ctx context.Context, userID uint, hashed string) error
	FindUserByAuthIdentity(ctx context.Context, ai *AuthIdentity) (*User, error)
	FindPasswordCredential(ctx context.Context, userID uint) (*PasswordCredential, error)
	DeleteAuthIdentity(ctx context.Context, userID uint) error
	UpdateIdentityTokens(ctx context.Context, id uint, accessToken, refreshToken string) error
}

type InvitationsRepo interface {
	Create(ctx context.Context, inv *TeamInvitation) error
	ByToken(ctx context.Context, token string) (*TeamInvitation, error)
	MarkAccepted(ctx context.Context, id uint) error
	ListByTeam(ctx context.Context, teamID uint) ([]TeamInvitation, error)
	Revoke(ctx context.Context, id uint) error
}

type NotificationsRepo interface {
	Create(ctx context.Context, n *Notification) error
	ListForUser(ctx context.Context, userID uint) ([]Notification, error)
	MarkRead(ctx context.Context, id uint) error
}

type UserPreferencesRepo interface {
	Create(ctx context.Context, userID uint) error
	Get(ctx context.Context, userID uint) (*UserPreference, error)
	GetByEmail(ctx context.Context, userEmail string) (*UserPreference, error)
	Update(ctx context.Context, preference *UserPreference) error
}

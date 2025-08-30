package db

import (
	"context"

	"gorm.io/gorm"
)

type Connection struct {
	DBConn       *gorm.DB
	Users        UsersRepo
	Workspaces   WorkspacesRepo
	Auth         AuthRepo
	Invitations  InvitationsRepo
	Notifications NotificationsRepo
}

func NewConnection(db *gorm.DB) *Connection {
	return &Connection{
		DBConn:       db,
		Users:        &usersRepo{db: db},
		Workspaces:   &workspacesRepo{db: db},
		Auth:         &authRepo{db: db},
		Invitations:  &invitationsRepo{db: db},
		Notifications: &notificationsRepo{db: db},
	}
}

func (c *Connection) WithTx(ctx context.Context, fn func(tx *Connection) error) error {
	return c.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		localConn := &Connection{
			DBConn:       tx,
			Users:        &usersRepo{db: tx},
			Workspaces:   &workspacesRepo{db: tx},
			Auth:         &authRepo{db: tx},
			Invitations:  &invitationsRepo{db: tx},
			Notifications: &notificationsRepo{db: tx},
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

type WorkspacesRepo interface {
	Create(ctx context.Context, w *WorkSpace) error
	ByID(ctx context.Context, id uint) (*WorkSpace, error)
	AddMember(ctx context.Context, workspaceID, userID uint, role string) error
	RemoveMember(ctx context.Context, workspaceID, userID uint) error
	ListForUser(ctx context.Context, userID uint) ([]WorkSpace, error)
	ReassignOwner(ctx context.Context, workspaceID, newOwnerID uint) error
	GetUserRole(ctx context.Context, workspaceID, userID uint) (string, error)
	RolesForWorkspace(ctx context.Context, workspaceID uint) (map[uint]string, error)
	Update(ctx context.Context, w *WorkSpace) error
	Delete(ctx context.Context, w *WorkSpace) error
}

type AuthRepo interface {
	FindAuthIdentity(ctx context.Context, provider, subject string) (*AuthIdentity, error)
	LinkIdentity(ctx context.Context, userID uint, provider, subject string, providerEmail *string) error
	EnsurePasswordCredential(ctx context.Context, userID uint, hashed string) error
	FindUserByAuthIdentity(ctx context.Context, ai *AuthIdentity) (*User, error)
	FindPasswordCredential(ctx context.Context, userID uint) (*PasswordCredential, error)
	DeleteAuthIdentity(ctx context.Context, userID uint) error
}

type InvitationsRepo interface {
	Create(ctx context.Context, inv *WorkspaceInvitation) error
	ByToken(ctx context.Context, token string) (*WorkspaceInvitation, error)
	MarkAccepted(ctx context.Context, id uint) error
	ListByWorkspace(ctx context.Context, wsID uint) ([]WorkspaceInvitation, error)
	Revoke(ctx context.Context, id uint) error
}

type NotificationsRepo interface {
	Create(ctx context.Context, n *Notification) error
	ListForUser(ctx context.Context, userID uint) ([]Notification, error)
	MarkRead(ctx context.Context, id uint) error
}

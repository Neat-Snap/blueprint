package workosclient

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/utils"
	"github.com/workos/workos-go/v5/pkg/usermanagement"
	"gorm.io/gorm"
)

func (c *Client) EnsureLocalUser(ctx context.Context, conn *db.Connection, wUser usermanagement.User) (*db.User, error) {
	email := utils.NormalizeEmail(wUser.Email)
	var emailPtr *string
	if email != "" {
		emailPtr = &email
	}

	name := strings.TrimSpace(strings.TrimSpace(strings.Join([]string{wUser.FirstName, wUser.LastName}, " ")))
	if name == "" {
		name = strings.TrimSpace(wUser.FirstName)
	}
	if name == "" {
		name = strings.TrimSpace(wUser.LastName)
	}
	var namePtr *string
	if name != "" {
		namePtr = &name
	}

	avatar := strings.TrimSpace(wUser.ProfilePictureURL)
	var avatarPtr *string
	if avatar != "" {
		avatarPtr = &avatar
	}

	var verifiedAt *time.Time
	if wUser.EmailVerified {
		if t, err := time.Parse(time.RFC3339, wUser.UpdatedAt); err == nil {
			verifiedAt = &t
		} else {
			now := time.Now()
			verifiedAt = &now
		}
	}

	existing, err := conn.Users.ByWorkOSID(ctx, wUser.ID)
	if err == nil {
		changed := false
		if emailPtr != nil {
			if existing.Email == nil || *existing.Email != *emailPtr {
				existing.Email = emailPtr
				changed = true
			}
		}
		if namePtr != nil {
			if existing.Name == nil || *existing.Name != *namePtr {
				existing.Name = namePtr
				changed = true
			}
		}
		if avatarPtr != nil {
			if existing.AvatarURL == nil || *existing.AvatarURL != *avatarPtr {
				existing.AvatarURL = avatarPtr
				changed = true
			}
		}
		if !wUser.EmailVerified {
			if existing.EmailVerifiedAt != nil {
				existing.EmailVerifiedAt = nil
				changed = true
			}
		} else if verifiedAt != nil {
			if existing.EmailVerifiedAt == nil || !existing.EmailVerifiedAt.Equal(*verifiedAt) {
				existing.EmailVerifiedAt = verifiedAt
				changed = true
			}
		}

		if changed {
			if err := conn.Users.Update(ctx, existing); err != nil {
				return nil, err
			}
		}
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	user := &db.User{
		WorkOSUserID:    wUser.ID,
		Email:           emailPtr,
		Name:            namePtr,
		AvatarURL:       avatarPtr,
		EmailVerifiedAt: verifiedAt,
	}

	if err := conn.Users.Create(ctx, user); err != nil {
		return nil, err
	}
	if err := conn.Preferences.Create(ctx, user.ID); err != nil && !errors.Is(err, gorm.ErrDuplicatedKey) {
		// ignore duplicate preference creation
		return nil, err
	}
	return user, nil
}

func (c *Client) EnsureLocalUserByID(ctx context.Context, conn *db.Connection, workosID string) (*db.User, error) {
	if strings.TrimSpace(workosID) == "" {
		return nil, errors.New("workos id is empty")
	}
	existing, err := conn.Users.ByWorkOSID(ctx, workosID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	remote, err := c.GetUser(ctx, workosID)
	if err != nil {
		return nil, err
	}
	return c.EnsureLocalUser(ctx, conn, remote)
}

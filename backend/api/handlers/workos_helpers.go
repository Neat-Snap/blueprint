package handlers

import (
	"time"

	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/utils"
	"github.com/workos/workos-go/v4/pkg/usermanagement"
)

func applyWorkOSUser(local *db.User, remote usermanagement.User) {
	if local == nil {
		return
	}

	email := utils.NormalizeEmail(remote.Email)
	if email != "" {
		local.Email = &email
	}

	if remote.ID != "" {
		if local.WorkOSUserID == nil || *local.WorkOSUserID != remote.ID {
			id := remote.ID
			local.WorkOSUserID = &id
		}
	}

	local.WorkOSEmailVerified = remote.EmailVerified
	if remote.EmailVerified {
		if local.EmailVerifiedAt == nil {
			now := time.Now()
			local.EmailVerifiedAt = &now
		}
	} else {
		local.EmailVerifiedAt = nil
	}
}

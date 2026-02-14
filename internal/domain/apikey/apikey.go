package apikey

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type APIKey struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	KeyHash     string
	KeyPrefix   string
	Permissions []Permission
	ExpiresAt   *time.Time
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	LastUsedAt  *time.Time
	RevokedAt   *time.Time
	RevokedBy   *uuid.UUID
}

type Permission string

const (
	PermissionRead          Permission = "read"
	PermissionWrite         Permission = "write"
	PermissionDelete        Permission = "delete"
	errInvalidPermissionFmt            = "invalid permission: %s"
)

// Validate validates the permission
func (p Permission) Validate() error {
	switch p {
	case PermissionRead, PermissionWrite, PermissionDelete:
		return nil
	default:
		return fmt.Errorf(errInvalidPermissionFmt, p)
	}
}

type CreateAPIKeyInput struct {
	ProjectID   uuid.UUID
	Name        string
	KeyHash     string
	KeyPrefix   string
	Permissions []Permission
	ExpiresAt   *time.Time
	CreatedBy   uuid.UUID
}

type RevokeAPIKeyInput struct {
	ID        uuid.UUID
	RevokedBy uuid.UUID
}

// IsActive returns true if the API key is active (not expired, not revoked)
func (k *APIKey) IsActive() bool {
	if k.RevokedAt != nil {
		return false
	}

	if k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now()) {
		return false
	}

	return true
}

// HasPermission returns true if the API key has the given permission
func (k *APIKey) HasPermission(perm Permission) bool {
	for _, p := range k.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

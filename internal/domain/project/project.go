package project

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID           uuid.UUID
	ClientID     uuid.UUID
	Name         string
	S3BucketName string
	IsDefault    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateProjectInput struct {
	ClientID  uuid.UUID
	Name      string
	IsDefault bool
}

type UpdateProjectInput struct {
	Name *string
}

// Member represents a project member with their role
type Member struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	UserID    uuid.UUID
	Role      Role
	InvitedBy *uuid.UUID
	InvitedAt time.Time
}

type Role string

const (
	RoleViewer        Role = "viewer"
	RoleEditor        Role = "editor"
	RoleAdmin         Role = "admin"
	errInvalidRoleFmt      = "invalid role: %s"
)

// Validate validates the role
func (r Role) Validate() error {
	switch r {
	case RoleViewer, RoleEditor, RoleAdmin:
		return nil
	default:
		return fmt.Errorf(errInvalidRoleFmt, r)
	}
}

type AddMemberInput struct {
	ProjectID uuid.UUID
	UserID    uuid.UUID
	Role      Role
	InvitedBy uuid.UUID
}

type UpdateMemberRoleInput struct {
	ProjectID uuid.UUID
	UserID    uuid.UUID
	Role      Role
}

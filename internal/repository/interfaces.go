package repository

import (
	"context"
	"file-service/internal/domain/apikey"
	"file-service/internal/domain/file"
	"file-service/internal/domain/project"

	"github.com/google/uuid"
)

// Repository interfaces used by auth and middleware packages
// These are provider-side interfaces that concrete implementations must satisfy

type APIKeyRepository interface {
	GetByHash(ctx context.Context, hash string) (*apikey.APIKey, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

type ProjectRepository interface {
	GetMember(ctx context.Context, projectID, userID uuid.UUID) (*project.Member, error)
}

type FileRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*file.File, error)
}

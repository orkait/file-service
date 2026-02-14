package repository

import (
	"context"
	"file-service/internal/domain/apikey"
	"file-service/internal/domain/client"
	"file-service/internal/domain/file"
	"file-service/internal/domain/project"
	"file-service/internal/domain/share"
	"file-service/internal/domain/user"

	"github.com/google/uuid"
)

// UserRepository defines user data access operations
type UserRepository interface {
	Create(ctx context.Context, input user.CreateUserInput) (*user.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	GetByClientID(ctx context.Context, clientID uuid.UUID) ([]*user.User, error)
	Update(ctx context.Context, id uuid.UUID, input user.UpdateUserInput) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ClientRepository defines client data access operations
type ClientRepository interface {
	Create(ctx context.Context, input client.CreateClientInput) (*client.Client, error)
	GetByID(ctx context.Context, id uuid.UUID) (*client.Client, error)
	GetByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) (*client.Client, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ProjectRepository defines project data access operations
type ProjectRepository interface {
	Create(ctx context.Context, input project.CreateProjectInput) (*project.Project, error)
	GetByID(ctx context.Context, id uuid.UUID) (*project.Project, error)
	GetByClientID(ctx context.Context, clientID uuid.UUID) ([]*project.Project, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*project.Project, error)
	GetDefaultByClientID(ctx context.Context, clientID uuid.UUID) (*project.Project, error)
	Update(ctx context.Context, id uuid.UUID, input project.UpdateProjectInput) error
	Delete(ctx context.Context, id uuid.UUID) error

	AddMember(ctx context.Context, input project.AddMemberInput) (*project.Member, error)
	GetMember(ctx context.Context, projectID, userID uuid.UUID) (*project.Member, error)
	GetMembers(ctx context.Context, projectID uuid.UUID) ([]*project.Member, error)
	CountAdminsByProject(ctx context.Context, projectID uuid.UUID) (int, error)
	UpdateMemberRole(ctx context.Context, input project.UpdateMemberRoleInput) error
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
}

// FileRepository defines file data access operations
type FileRepository interface {
	Create(ctx context.Context, input file.CreateFileInput) (*file.File, error)
	GetByID(ctx context.Context, id uuid.UUID) (*file.File, error)
	GetByProjectAndS3Key(ctx context.Context, projectID uuid.UUID, s3Key string) (*file.File, error)
	List(ctx context.Context, filter file.ListFilesFilter) ([]*file.File, error)
	DeleteByProjectAndPrefix(ctx context.Context, projectID uuid.UUID, prefix string) (int64, error)
	CountByProjectAndPrefix(ctx context.Context, projectID uuid.UUID, prefix string) (int64, error)
	Update(ctx context.Context, id uuid.UUID, input file.UpdateFileInput) error
	Delete(ctx context.Context, id uuid.UUID) error

	CreateFolder(ctx context.Context, input file.CreateFolderInput) (*file.Folder, error)
	GetFolder(ctx context.Context, id uuid.UUID) (*file.Folder, error)
	GetFolderByPath(ctx context.Context, projectID uuid.UUID, s3Prefix string) (*file.Folder, error)
	ListFolders(ctx context.Context, projectID uuid.UUID, parentFolderID *uuid.UUID) ([]*file.Folder, error)
	DeleteFolder(ctx context.Context, id uuid.UUID) error
}

// APIKeyRepository defines API key data access operations
type APIKeyRepository interface {
	Create(ctx context.Context, input apikey.CreateAPIKeyInput) (*apikey.APIKey, error)
	GetByID(ctx context.Context, id uuid.UUID) (*apikey.APIKey, error)
	GetByHash(ctx context.Context, keyHash string) (*apikey.APIKey, error)
	ListByProjectID(ctx context.Context, projectID uuid.UUID) ([]*apikey.APIKey, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	Revoke(ctx context.Context, input apikey.RevokeAPIKeyInput) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ShareLinkRepository defines share link data access operations
type ShareLinkRepository interface {
	Create(ctx context.Context, input share.CreateShareLinkInput) (*share.ShareLink, error)
	GetByID(ctx context.Context, id uuid.UUID) (*share.ShareLink, error)
	GetByToken(ctx context.Context, token string) (*share.ShareLink, error)
	ListByFileID(ctx context.Context, fileID uuid.UUID) ([]*share.ShareLink, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

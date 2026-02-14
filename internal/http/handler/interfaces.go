package handler

import (
	"context"
	"file-service/internal/domain/apikey"
	"file-service/internal/domain/client"
	"file-service/internal/domain/file"
	"file-service/internal/domain/project"
	"file-service/internal/domain/share"
	"file-service/internal/domain/user"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

// Consumer-side interfaces defined by handlers
// Each interface contains only the methods needed by the specific handler

// AuthHandler interfaces
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	Create(ctx context.Context, input user.CreateUserInput) (*user.User, error)
}

type TransactionExecutor interface {
	SignupTransaction(ctx context.Context, email, passwordHash string) (*user.User, *client.Client, *project.Project, error)
	RollbackSignup(ctx context.Context, clientID uuid.UUID) error
}

type TokenGenerator interface {
	Generate(userID, clientID uuid.UUID, email string) (string, error)
}

// FileHandler interfaces
type FileRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*file.File, error)
	GetByProjectAndS3Key(ctx context.Context, projectID uuid.UUID, s3Key string) (*file.File, error)
	Create(ctx context.Context, input file.CreateFileInput) (*file.File, error)
	Update(ctx context.Context, id uuid.UUID, input file.UpdateFileInput) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter file.ListFilesFilter) ([]*file.File, error)
	DeleteByProjectAndPrefix(ctx context.Context, projectID uuid.UUID, prefix string) (int64, error)
	CountByProjectAndPrefix(ctx context.Context, projectID uuid.UUID, prefix string) (int64, error)
}

type FolderRepository interface {
	GetFolder(ctx context.Context, id uuid.UUID) (*file.Folder, error)
	GetFolderByPath(ctx context.Context, projectID uuid.UUID, path string) (*file.Folder, error)
	CreateFolder(ctx context.Context, input file.CreateFolderInput) (*file.Folder, error)
	ListFolders(ctx context.Context, projectID uuid.UUID, parentID *uuid.UUID) ([]*file.Folder, error)
	DeleteFolder(ctx context.Context, id uuid.UUID) error
}

type ProjectGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*project.Project, error)
}

// ProjectHandler interfaces
type ProjectRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*project.Project, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*project.Project, error)
	Create(ctx context.Context, input project.CreateProjectInput) (*project.Project, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ProjectMemberRepository interface {
	AddMember(ctx context.Context, input project.AddMemberInput) (*project.Member, error)
	GetMembers(ctx context.Context, projectID uuid.UUID) ([]*project.Member, error)
	GetMember(ctx context.Context, projectID, userID uuid.UUID) (*project.Member, error)
	UpdateMemberRole(ctx context.Context, input project.UpdateMemberRoleInput) error
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
	CountAdminsByProject(ctx context.Context, projectID uuid.UUID) (int, error)
}

type UserGetter interface {
	GetByEmail(ctx context.Context, email string) (*user.User, error)
}

// ShareHandler interfaces
type ShareRepository interface {
	Create(ctx context.Context, input share.CreateShareLinkInput) (*share.ShareLink, error)
	GetByToken(ctx context.Context, token string) (*share.ShareLink, error)
}

type FileGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*file.File, error)
}

// APIKeyHandler interfaces
type APIKeyRepository interface {
	Create(ctx context.Context, input apikey.CreateAPIKeyInput) (*apikey.APIKey, error)
	GetByID(ctx context.Context, id uuid.UUID) (*apikey.APIKey, error)
	ListByProjectID(ctx context.Context, projectID uuid.UUID) ([]*apikey.APIKey, error)
	Revoke(ctx context.Context, input apikey.RevokeAPIKeyInput) error
}

// Storage interfaces (used by multiple handlers)
type StorageOperations interface {
	GeneratePresignedUploadURL(ctx context.Context, bucket, key, contentType string) (string, error)
	GeneratePresignedDownloadURL(ctx context.Context, bucket, key string) (string, error)
	DeleteObject(ctx context.Context, bucket, key string) error
	DeleteFolder(ctx context.Context, bucket, prefix string) error
	DeleteBucket(ctx context.Context, bucket string) error
	ListObjects(ctx context.Context, bucket, prefix string, limit int) (*s3.ListObjectsV2Output, error)
}

type BucketCreator interface {
	CreateBucket(ctx context.Context, name, region string) error
}

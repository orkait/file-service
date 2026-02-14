package types

import (
	"context"
	"file-service/internal/domain/client"
	"file-service/internal/domain/project"
	"file-service/internal/domain/user"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// AuditLogger defines audit logging operations
type AuditLogger interface {
	LogFromContext(c echo.Context, resourceType string, resourceID *uuid.UUID, action string, status string, metadata map[string]any) error
	LogError(c echo.Context, resourceType string, resourceID *uuid.UUID, action string, err error) error
}

// TransactionManager handles database transactions for signup
type TransactionManager interface {
	SignupTransaction(ctx context.Context, email, passwordHash string) (*user.User, *client.Client, *project.Project, error)
	RollbackSignup(ctx context.Context, clientID uuid.UUID) error
}

// BucketCreator handles S3 bucket operations
type BucketCreator interface {
	CreateBucket(ctx context.Context, bucketName, region string) error
	DeleteBucket(ctx context.Context, bucketName string) error
	DeleteFolder(ctx context.Context, bucketName, prefix string) error
	DeleteObject(ctx context.Context, bucketName, key string) error
	GeneratePresignedUploadURL(ctx context.Context, bucketName, key, contentType string) (string, error)
	GeneratePresignedDownloadURL(ctx context.Context, bucketName, key string) (string, error)
	ListObjects(ctx context.Context, bucketName, prefix string, maxKeys int) (*s3.ListObjectsV2Output, error)
}

// CSRFTokenManager handles CSRF token operations
type CSRFTokenManager interface {
	GetOrCreateToken(userID uuid.UUID) (string, error)
	Middleware() echo.MiddlewareFunc
}

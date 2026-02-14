package file

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	FolderID   *uuid.UUID
	Name       string
	S3Key      string
	SizeBytes  int64
	MimeType   string
	UploadedBy *uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Folder struct {
	ID             uuid.UUID
	ProjectID      uuid.UUID
	ParentFolderID *uuid.UUID
	Name           string
	S3Prefix       string
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
}

type CreateFileInput struct {
	ProjectID  uuid.UUID
	FolderID   *uuid.UUID
	Name       string
	S3Key      string
	SizeBytes  int64
	MimeType   string
	UploadedBy *uuid.UUID
}

type CreateFolderInput struct {
	ProjectID      uuid.UUID
	ParentFolderID *uuid.UUID
	Name           string
	S3Prefix       string
	CreatedBy      *uuid.UUID
}

type UpdateFileInput struct {
	Name      *string
	SizeBytes *int64
	MimeType  *string
}

type ListFilesFilter struct {
	ProjectID uuid.UUID
	FolderID  *uuid.UUID
	Limit     int
	Offset    int
}

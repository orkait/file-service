package models

import "time"

type Client struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Email               string     `json:"email"`
	PasswordHash        string     `json:"-"`
	Status              string     `json:"status"`
	PausedAt            *time.Time `json:"paused_at,omitempty"`
	ScheduledDeletionAt *time.Time `json:"scheduled_deletion_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type Project struct {
	ID          string    `json:"id"`
	ClientID    string    `json:"client_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type APIKey struct {
	ID          string     `json:"id"`
	ClientID    string     `json:"client_id"`
	ProjectID   *string    `json:"project_id,omitempty"`
	KeyHash     string     `json:"-"`
	KeyPrefix   string     `json:"key_prefix"`
	Name        string     `json:"name"`
	Permissions []string   `json:"permissions"`
	IsActive    bool       `json:"is_active"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type Asset struct {
	ID               string    `json:"id"`
	ClientID         string    `json:"client_id"`
	ProjectID        string    `json:"project_id"`
	FolderPath       string    `json:"folder_path"`
	Filename         string    `json:"filename"`
	OriginalFilename string    `json:"original_filename"`
	FileSize         int64     `json:"file_size"`
	MimeType         string    `json:"mime_type,omitempty"`
	S3Key            string    `json:"s3_key"`
	PresignedURL     string    `json:"presigned_url,omitempty"`
	Version          int       `json:"version"`
	IsLatest         bool      `json:"is_latest"`
	ParentAssetID    *string   `json:"parent_asset_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ProjectMember struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	ClientID  string    `json:"client_id"`
	Role      string    `json:"role"`
	InvitedBy *string   `json:"invited_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

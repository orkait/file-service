package app

import "io"

// UploadFileRequest represents a file upload request
type UploadFileRequest struct {
	Reader     io.Reader
	FileName   string
	FolderPath string
}

// UploadFileResponse represents a file upload response
type UploadFileResponse struct {
	ObjectKey string
	Message   string
}

// ListFilesRequest represents a list files request
type ListFilesRequest struct {
	FolderPath    string
	NextPageToken string
	PageSize      int
	IsFolder      bool
}

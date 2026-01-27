package s3

import (
	"io"
	"time"
)

type ObjectDetails struct {
	Name         string    `json:"name"`
	IsFolder     bool      `json:"isFolder"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
	DownloadLink string    `json:"downloadLink,omitempty"`
}

// CreateFolderRequest represents the request body structure for creating a folder
type CreateFolderRequest struct {
	FolderName string `json:"folderName"`
}

type ListFilesResponse struct {
	Files               *[]ObjectDetails `json:"data"`
	NextPageToken       string           `json:"nextPageToken,omitempty"` // for pagination purposes only if
	IsLastPage          bool             `json:"isLastPage,omitempty"`    // for pagination purposes only if
	NoOfRecordsReturned int32            `json:"noOfRecordsReturned,omitempty"`
	FilesCount          int32            `json:"filesCount,omitempty"`
	FoldersCount        int32            `json:"foldersCount,omitempty"`
}

type SuccessResponse struct {
	Status       string      `json:"status"`
	ResponseCode int         `json:"response_code"`
	Data         interface{} `json:"data"`
}

type FailureResponse struct {
	Status       string `json:"status"`
	ResponseCode int    `json:"response_code"`
	ErrorMessage string `json:"error_message"`
}

type S3UploadPayload struct {
	Bucket     string `json:"bucket"`
	FolderPath string `json:"folderPath"`
}

// FileInfo represents the details of a file or folder
type FileInfo struct {
	Name     string `json:"name"`
	IsFolder bool   `json:"isFolder"`
	// Add more fields as per your requirements
}

type FilterOptions struct {
	SizeRange          string
	TimeRange          string
	FileTypes          []string
	FilenameQuery      string
	FilenameFilterType string
	FileSize           int64
	FileSizeFilterType string
}

type FilterSizeRange struct {
	MinSize int64
	MaxSize int64
}

// ========== Batch Operations Types ==========

// FileUploadInput represents input for a single file in batch upload
type FileUploadInput struct {
	Reader    io.Reader
	FileName  string
	ObjectKey string
}

// BatchUploadResult represents result of a single file in batch upload
type BatchUploadResult struct {
	FileName  string `json:"fileName"`
	ObjectKey string `json:"objectKey,omitempty"`
	Error     string `json:"error,omitempty"`
	Success   bool   `json:"success"`
}

// BatchUploadResponse represents the complete batch upload response
type BatchUploadResponse struct {
	Uploaded      []BatchUploadResult `json:"uploaded"`
	Failed        []BatchUploadResult `json:"failed"`
	TotalUploaded int                 `json:"totalUploaded"`
	TotalFailed   int                 `json:"totalFailed"`
}

// BatchDownloadRequest represents the batch download request body
type BatchDownloadRequest struct {
	Paths []string `json:"paths"`
}

// BatchDownloadResult represents result of a single file in batch download
type BatchDownloadResult struct {
	Path     string `json:"path"`
	FileName string `json:"fileName"`
	URL      string `json:"url,omitempty"`
	Error    string `json:"error,omitempty"`
	Success  bool   `json:"success"`
}

// BatchDownloadResponse represents the complete batch download response
type BatchDownloadResponse struct {
	Files        []BatchDownloadResult `json:"files"`
	TotalSuccess int                   `json:"totalSuccess"`
	TotalFailed  int                   `json:"totalFailed"`
}

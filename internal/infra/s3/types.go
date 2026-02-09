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

type ListFilesResponse struct {
	Files               *[]ObjectDetails `json:"data"`
	NextPageToken       string           `json:"nextPageToken,omitempty"`
	IsLastPage          bool             `json:"isLastPage,omitempty"`
	NoOfRecordsReturned int32            `json:"noOfRecordsReturned,omitempty"`
	FilesCount          int32            `json:"filesCount,omitempty"`
	FoldersCount        int32            `json:"foldersCount,omitempty"`
}

type FileUploadInput struct {
	Reader    io.Reader
	FileName  string
	ObjectKey string
}

type BatchUploadResult struct {
	FileName  string `json:"fileName"`
	ObjectKey string `json:"objectKey,omitempty"`
	Error     string `json:"error,omitempty"`
	Success   bool   `json:"success"`
}

type BatchUploadResponse struct {
	Uploaded      []BatchUploadResult `json:"uploaded"`
	Failed        []BatchUploadResult `json:"failed"`
	TotalUploaded int                 `json:"totalUploaded"`
	TotalFailed   int                 `json:"totalFailed"`
}

type BatchDownloadResult struct {
	Path     string `json:"path"`
	FileName string `json:"fileName"`
	URL      string `json:"url,omitempty"`
	Error    string `json:"error,omitempty"`
	Success  bool   `json:"success"`
}

type BatchDownloadResponse struct {
	Files        []BatchDownloadResult `json:"files"`
	TotalSuccess int                   `json:"totalSuccess"`
	TotalFailed  int                   `json:"totalFailed"`
}

type BatchDownloadRequest struct {
	Paths []string `json:"paths"`
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

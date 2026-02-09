package app

import (
	"context"
	"file-service/config"
	"file-service/internal/infra/cache"
	"file-service/internal/infra/s3"
	"file-service/internal/rbac"
	"file-service/internal/transport/echo"
	"fmt"
	"log"
	"time"
)

// Service represents the file management application
type Service struct {
	config      *config.Config
	s3Client    *s3.Client
	urlCache    *cache.URLCache
	rbacChecker *rbac.Checker
	server      *echo.Server
}

// NewService creates and initializes a new Service instance
// This is a convenience wrapper around InitializeService
func NewService() (*Service, error) {
	return InitializeService()
}

// Start starts the service and all background tasks
func (s *Service) Start() error {
	// Start cache cleanup goroutine
	go s.startCacheCleanup()

	// Start HTTP server
	log.Println("Starting file service...")
	return s.server.Start()
}

// startCacheCleanup runs a background task to clear expired cache entries
func (s *Service) startCacheCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.urlCache.Clear()
	}
}

// Shutdown gracefully shuts down the service
func (s *Service) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// UploadFile uploads a single file to S3
func (s *Service) UploadFile(ctx context.Context, req *UploadFileRequest) (*UploadFileResponse, error) {
	objectKey := s3.BuildObjectKey(req.FolderPath, req.FileName)

	if err := s.s3Client.UploadFile(req.Reader, objectKey); err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &UploadFileResponse{
		ObjectKey: objectKey,
		Message:   fmt.Sprintf("File uploaded successfully with object key: %s", objectKey),
	}, nil
}

// ListFiles lists files and folders in a given path
func (s *Service) ListFiles(ctx context.Context, req *ListFilesRequest) (*s3.ListFilesResponse, error) {
	return s.s3Client.ListFiles(req.FolderPath, req.NextPageToken, req.PageSize, req.IsFolder, s.urlCache)
}

// GenerateDownloadLink generates a presigned download URL
func (s *Service) GenerateDownloadLink(ctx context.Context, path string) (string, error) {
	return s.s3Client.GenerateDownloadLink(path, s.urlCache)
}

// DeleteFile deletes a file from S3
func (s *Service) DeleteFile(ctx context.Context, path string) error {
	return s.s3Client.DeleteObject(path)
}

// DeleteFolder deletes a folder and its contents recursively
func (s *Service) DeleteFolder(ctx context.Context, path string) error {
	return s.s3Client.DeleteFolder(path)
}

// CreateFolder creates a new folder in S3
func (s *Service) CreateFolder(ctx context.Context, folderPath string) error {
	return s.s3Client.CreateFolder(folderPath)
}

// BatchUploadFiles uploads multiple files concurrently
func (s *Service) BatchUploadFiles(ctx context.Context, files []s3.FileUploadInput, maxWorkers int) *s3.BatchUploadResponse {
	return s.s3Client.BatchUploadFiles(ctx, files, maxWorkers)
}

// BatchGenerateDownloadLinks generates multiple download URLs concurrently
func (s *Service) BatchGenerateDownloadLinks(ctx context.Context, paths []string, maxWorkers int) *s3.BatchDownloadResponse {
	return s.s3Client.BatchGenerateDownloadLinks(ctx, paths, s.urlCache, maxWorkers)
}

// ListAllFolders lists all folders in a given path
func (s *Service) ListAllFolders(ctx context.Context, folderPath string) []s3.ObjectDetails {
	return s.s3Client.ListAllFolders(folderPath)
}

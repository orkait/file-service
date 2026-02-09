package app

import (
	"file-service/config"
	"file-service/internal/infra/cache"
	"file-service/internal/infra/s3"
	"file-service/internal/rbac"
	"file-service/internal/rbac/presets"
	"file-service/internal/transport/echo"
	"fmt"
)

// InitializeService wires up all dependencies and returns a configured Service
func InitializeService() (*Service, error) {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize S3 client
	s3Client, err := s3.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Initialize URL cache
	urlCache := cache.NewURLCache()

	// Initialize RBAC checker with file management preset
	rbacChecker := rbac.MustNew(presets.FileManagement())

	// Initialize Echo server
	server := echo.NewServer(cfg, s3Client, urlCache, rbacChecker)

	return &Service{
		config:      cfg,
		s3Client:    s3Client,
		urlCache:    urlCache,
		rbacChecker: rbacChecker,
		server:      server,
	}, nil
}

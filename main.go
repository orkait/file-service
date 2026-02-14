package main

import (
	"context"
	"file-service/internal/audit"
	"file-service/internal/auth"
	"file-service/internal/config"
	"file-service/internal/http"
	"file-service/internal/http/middleware"
	"file-service/internal/repository/postgres"
	"file-service/internal/storage/s3"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

const (
	envFilePath      = ".env"
	serverAddrPrefix = ":"
	signalBufferSize = 1
	logOutputFlags   = log.LstdFlags | log.Lshortfile
)

var shutdownSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
}

func main() {
	if err := godotenv.Load(envFilePath); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	log.SetOutput(os.Stderr)
	log.SetFlags(logOutputFlags)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Println("Configuration loaded successfully")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := postgres.New(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Database connection established")

	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	fileRepo := postgres.NewFileRepository(db)
	apiKeyRepo := postgres.NewAPIKeyRepository(db)
	shareRepo := postgres.NewShareLinkRepository(db)

	s3Client, err := s3.NewClient(&cfg.AWS, cfg.App.PresignedURLExpiry)
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}

	log.Println("S3 client initialized")

	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.ExpiryDuration)
	apiKeyService := auth.NewAPIKeyService(apiKeyRepo)
	authMiddleware := auth.NewMiddleware(jwtService, apiKeyService, apiKeyRepo)
	rbacMiddleware := auth.NewRBACMiddleware(projectRepo, fileRepo)
	csrfMiddleware := middleware.NewCSRFMiddleware(ctx)
	defer csrfMiddleware.Stop()

	// Create audit logger with adapter
	auditLoggerImpl := audit.NewLogger(db.Pool)
	auditLogger := &auditLoggerAdapter{logger: auditLoggerImpl}

	serverDeps := &http.ServerDependencies{
		Config:         cfg,
		DB:             db,
		UserRepo:       userRepo,
		ProjectRepo:    projectRepo,
		FileRepo:       fileRepo,
		APIKeyRepo:     apiKeyRepo,
		ShareRepo:      shareRepo,
		S3Client:       s3Client,
		BucketRegion:   cfg.AWS.Region,
		JWTService:     jwtService,
		APIKeyService:  apiKeyService,
		AuthMiddleware: authMiddleware,
		RBACMiddleware: rbacMiddleware,
		AuditLogger:    auditLogger,
		CSRFMiddleware: csrfMiddleware,
	}

	server := http.NewServer(serverDeps)

	go func() {
		log.Printf("Starting HTTP server on port %s", cfg.Server.Port)
		if err := server.Start(serverAddrPrefix + cfg.Server.Port); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, signalBufferSize)
	signal.Notify(quit, shutdownSignals...)
	<-quit

	log.Println("Shutting down server...")

	// Cancel context to stop background tasks
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

// auditLoggerAdapter adapts audit.Logger to types.AuditLogger interface
type auditLoggerAdapter struct {
	logger *audit.Logger
}

func (a *auditLoggerAdapter) LogFromContext(c echo.Context, resourceType string, resourceID *uuid.UUID, action string, status string, metadata map[string]any) error {
	return a.logger.LogFromContext(c, audit.ResourceType(resourceType), resourceID, audit.Action(action), audit.Status(status), metadata)
}

func (a *auditLoggerAdapter) LogError(c echo.Context, resourceType string, resourceID *uuid.UUID, action string, err error) error {
	return a.logger.LogError(c, audit.ResourceType(resourceType), resourceID, audit.Action(action), err)
}

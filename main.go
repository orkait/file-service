package main

import (
	"context"
	"file-service/internal/auth"
	"file-service/internal/config"
	"file-service/internal/http"
	"file-service/internal/repository/postgres"
	"file-service/internal/storage/s3"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
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

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

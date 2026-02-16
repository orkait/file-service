package main

import (
	"context"
	"file-service/config"
	"file-service/pkg/cache"
	"file-service/pkg/s3"
	"file-service/routes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	} else {
		port = ":" + port
	}

	return port
}

func main() {
	e := echo.New()

	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		fmt.Println("Error loading environment variables")
	}

	log.SetOutput(os.Stderr)
	// Apply rate limiter middleware
	rateLimiterConfig := middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{Rate: 10, Burst: 30, ExpiresIn: 3 * time.Minute},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			id := ctx.RealIP()
			return id, nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusForbidden, nil)
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, nil)
		},
	}

	// Apply rate limiter middleware
	e.Use(middleware.RateLimiterWithConfig(rateLimiterConfig))

	// Apply CORS middleware
	e.Use(middleware.CORS())

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}

	// Create S3 client once and reuse
	s3Client, err := s3.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create S3 client: %s", err)
	}

	urlCache := cache.NewURLCache()

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Spawn goroutine to clear cache with cancellation support
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				urlCache.Clear()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Register routes
	routes.RegisterRoutes(e, s3Client, urlCache)

	// Graceful shutdown
	go func() {
		if err := e.Start(getPort()); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %s", err)
		}
	}()

	log.Println("Server Started!!!")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	cancel() // Stop cache cleanup goroutine

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %s", err)
	}

	log.Println("Server exited")
}

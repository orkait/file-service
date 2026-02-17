package main

import (
	"context"
	"file-management-service/config"
	"file-management-service/pkg/cache"
	"file-management-service/pkg/metrics"
	"file-management-service/pkg/profiling"
	"file-management-service/routes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Global variable to hold the configuration
var AppConfig *config.Config

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

	// ── Performance profiling setup (for stress testing) ──────────────
	// Set memory limits for 1GB RAM environments
	profiling.SetMemoryLimit()

	// Register pprof and health endpoints if profiling is enabled
	if profiling.IsProfilingEnabled() {
		log.Println("Profiling ENABLED — /debug/pprof/*, /health, /metrics/* endpoints active")
		profiling.RegisterPprofRoutes(e)
		profiling.RegisterHealthRoutes(e)
		metrics.RegisterMetricsRoute(e)
		// Apply metrics middleware BEFORE other middlewares to capture all requests
		e.Use(metrics.MetricsMiddleware())
	}

	// ── Rate limiter (configurable — disable for stress tests) ───────
	disableRateLimiter := strings.ToLower(os.Getenv("DISABLE_RATE_LIMITER"))
	if disableRateLimiter == "true" || disableRateLimiter == "1" {
		log.Println("Rate limiter DISABLED (DISABLE_RATE_LIMITER=true)")
	} else {
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
		e.Use(middleware.RateLimiterWithConfig(rateLimiterConfig))
	}

	// Apply CORS middleware
	e.Use(middleware.CORS())

	config, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}

	// Assign the configuration to the global variable
	AppConfig = config

	cache := cache.NewURLCache()

	// spawn a goroutine to clear the cache every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			cache.Clear()
		}
	}()

	// Register routes
	routes.RegisterRoutes(e, AppConfig, cache)

	// Start server in goroutine for graceful shutdown
	go func() {
		if err := e.Start(getPort()); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server startup failed:", err)
		}
	}()

	log.Println("Server Started!")

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited gracefully")
}

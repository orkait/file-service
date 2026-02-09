package echo

import (
	"context"
	"file-service/config"
	"file-service/internal/infra/cache"
	"file-service/internal/infra/s3"
	"file-service/internal/rbac"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Server wraps the Echo server with dependencies
type Server struct {
	echo        *echo.Echo
	config      *config.Config
	s3Client    *s3.Client
	urlCache    *cache.URLCache
	rbacChecker *rbac.Checker
}

// NewServer creates a new Echo server with middleware and routes
func NewServer(cfg *config.Config, s3Client *s3.Client, urlCache *cache.URLCache, rbacChecker *rbac.Checker) *Server {
	e := echo.New()

	// Apply rate limiter middleware
	rateLimiterConfig := middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      10,
				Burst:     30,
				ExpiresIn: 3 * time.Minute,
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			return ctx.RealIP(), nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusForbidden, nil)
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, nil)
		},
	}

	e.Use(middleware.RateLimiterWithConfig(rateLimiterConfig))
	e.Use(middleware.CORS())

	server := &Server{
		echo:        e,
		config:      cfg,
		s3Client:    s3Client,
		urlCache:    urlCache,
		rbacChecker: rbacChecker,
	}

	server.registerRoutes()

	return server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return s.echo.Start(":" + port)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

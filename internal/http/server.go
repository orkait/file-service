package http

import (
	"context"
	"file-service/internal/auth"
	"file-service/internal/config"
	"file-service/internal/domain/apikey"
	"file-service/internal/http/handler"
	"file-service/internal/http/middleware"
	"file-service/internal/rbac/presets"
	"file-service/internal/repository/postgres"
	"file-service/internal/types"
	stdhttp "net/http"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

const (
	jsonKeyStatus    = "status"
	statusOK         = "ok"
	requestBodyLimit = "1M"
)

type ServerDependencies struct {
	Config         *config.Config
	DB             types.TransactionManager
	UserRepo       *postgres.UserRepository
	ProjectRepo    *postgres.ProjectRepository
	FileRepo       *postgres.FileRepository
	APIKeyRepo     *postgres.APIKeyRepository
	ShareRepo      *postgres.ShareLinkRepository
	S3Client       types.BucketCreator
	BucketRegion   string
	JWTService     *auth.JWTService
	APIKeyService  *auth.APIKeyService
	AuthMiddleware *auth.Middleware
	RBACMiddleware *auth.RBACMiddleware
	AuditLogger    types.AuditLogger
	CSRFMiddleware types.CSRFTokenManager
}

type Server struct {
	echo *echo.Echo
	deps *ServerDependencies
}

func NewServer(deps *ServerDependencies) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Set custom HTTP error handler
	e.HTTPErrorHandler = CustomHTTPErrorHandler

	e.Server.ReadTimeout = deps.Config.Server.ReadTimeout
	e.Server.WriteTimeout = deps.Config.Server.WriteTimeout

	// Request ID middleware (first, so all logs have request ID)
	e.Use(middleware.RequestID())
	e.Use(middleware.SecurityHeaders()) // Add security headers to all responses
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.BodyLimit(requestBodyLimit))
	// Note: echomiddleware.Secure() removed - SecurityHeaders() provides comprehensive security headers

	// Global rate limiting
	globalRateLimiter := middleware.NewGlobalRateLimiter()
	e.Use(globalRateLimiter.Middleware())

	// Strict rate limiting for auth endpoints
	strictRateLimiter := middleware.NewStrictRateLimiter()

	authHandler := handler.NewAuthHandler(deps.UserRepo, deps.DB, deps.S3Client, deps.JWTService, deps.BucketRegion, deps.AuditLogger, deps.CSRFMiddleware)
	projectHandler := handler.NewProjectHandler(deps.ProjectRepo, deps.ProjectRepo, deps.UserRepo, deps.S3Client, deps.S3Client, deps.BucketRegion, deps.AuditLogger)
	fileHandler := handler.NewFileHandler(deps.FileRepo, deps.FileRepo, deps.ProjectRepo, deps.S3Client, deps.AuditLogger)
	apiKeyHandler := handler.NewAPIKeyHandler(deps.APIKeyRepo, deps.AuditLogger)
	shareHandler := handler.NewShareHandler(deps.ShareRepo, deps.FileRepo, deps.ProjectRepo, deps.S3Client, deps.AuditLogger)

	// Auth endpoints with strict rate limiting
	e.POST("/auth/signup", authHandler.Signup, strictRateLimiter.Middleware())
	e.POST("/auth/login", authHandler.Login, strictRateLimiter.Middleware())
	e.GET("/shares/:token/download-url", shareHandler.GetDownloadURLByShareToken)
	e.GET("/health", healthCheck)

	api := e.Group("/api")
	jwtAPI := api.Group("")
	jwtAPI.Use(deps.AuthMiddleware.RequireJWT())
	jwtAPI.Use(deps.CSRFMiddleware.Middleware()) // Add CSRF protection for JWT-authenticated requests

	jwtAPI.GET("/projects", projectHandler.ListProjects)
	jwtAPI.POST("/projects", projectHandler.CreateProject)
	jwtAPI.GET("/projects/:id", projectHandler.GetProject, deps.RBACMiddleware.RequireProjectRole(presets.RoleViewer))
	jwtAPI.DELETE("/projects/:id", projectHandler.DeleteProject, deps.RBACMiddleware.RequireProjectRole(presets.RoleAdmin))
	jwtAPI.POST("/projects/:project_id/members", projectHandler.AddMember, deps.RBACMiddleware.RequireProjectRole(presets.RoleAdmin))
	jwtAPI.GET("/projects/:project_id/members", projectHandler.ListMembers, deps.RBACMiddleware.RequireProjectRole(presets.RoleViewer))
	jwtAPI.PUT("/projects/:project_id/members/:user_id", projectHandler.UpdateMemberRole, deps.RBACMiddleware.RequireProjectRole(presets.RoleAdmin))
	jwtAPI.DELETE("/projects/:project_id/members/:user_id", projectHandler.RemoveMember, deps.RBACMiddleware.RequireProjectRole(presets.RoleAdmin))

	jwtAPI.POST("/projects/:project_id/api-keys", apiKeyHandler.CreateAPIKey, deps.RBACMiddleware.RequireProjectRole(presets.RoleAdmin))
	jwtAPI.GET("/projects/:project_id/api-keys", apiKeyHandler.ListAPIKeys, deps.RBACMiddleware.RequireProjectRole(presets.RoleAdmin))
	jwtAPI.DELETE("/projects/:project_id/api-keys/:id", apiKeyHandler.RevokeAPIKey, deps.RBACMiddleware.RequireProjectRole(presets.RoleAdmin))

	jwtAPI.POST("/projects/:project_id/files/upload-url", fileHandler.GetUploadURL, deps.RBACMiddleware.RequireProjectRole(presets.RoleEditor))
	jwtAPI.GET("/files/:id", fileHandler.GetFile, deps.RBACMiddleware.RequireProjectRoleForFile(presets.RoleViewer))
	jwtAPI.GET("/files/:id/download-url", fileHandler.GetDownloadURL, deps.RBACMiddleware.RequireProjectRoleForFile(presets.RoleViewer))
	jwtAPI.GET("/projects/:project_id/files", fileHandler.ListFiles, deps.RBACMiddleware.RequireProjectRole(presets.RoleViewer))
	jwtAPI.DELETE("/files/:id", fileHandler.DeleteFile, deps.RBACMiddleware.RequireProjectRoleForFile(presets.RoleEditor))
	jwtAPI.POST("/projects/:project_id/folders", fileHandler.CreateFolder, deps.RBACMiddleware.RequireProjectRole(presets.RoleEditor))
	jwtAPI.GET("/projects/:project_id/folders", fileHandler.ListFolders, deps.RBACMiddleware.RequireProjectRole(presets.RoleViewer))
	jwtAPI.DELETE("/projects/:project_id/folders/:id", fileHandler.DeleteFolder, deps.RBACMiddleware.RequireProjectRole(presets.RoleEditor))

	jwtAPI.POST("/files/:id/share-link", shareHandler.CreateShareLink, deps.RBACMiddleware.RequireProjectRoleForFile(presets.RoleEditor))

	apiKeyRead := api.Group("/key")
	apiKeyRead.Use(deps.AuthMiddleware.RequireAPIKey(apikey.PermissionRead))
	apiKeyRead.GET("/projects/:project_id/files", fileHandler.ListFiles)
	apiKeyRead.GET("/files/:id", fileHandler.GetFile)
	apiKeyRead.GET("/files/:id/download-url", fileHandler.GetDownloadURL)
	apiKeyRead.GET("/projects/:project_id/folders", fileHandler.ListFolders)

	apiKeyWrite := api.Group("/key")
	apiKeyWrite.Use(deps.AuthMiddleware.RequireAPIKey(apikey.PermissionWrite))
	apiKeyWrite.POST("/projects/:project_id/files/upload-url", fileHandler.GetUploadURL)
	apiKeyWrite.POST("/projects/:project_id/folders", fileHandler.CreateFolder)

	apiKeyDelete := api.Group("/key")
	apiKeyDelete.Use(deps.AuthMiddleware.RequireAPIKey(apikey.PermissionDelete))
	apiKeyDelete.DELETE("/files/:id", fileHandler.DeleteFile)
	apiKeyDelete.DELETE("/projects/:project_id/folders/:id", fileHandler.DeleteFolder)

	return &Server{
		echo: e,
		deps: deps,
	}
}

func (s *Server) Start(address string) error {
	return s.echo.Start(address)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

func healthCheck(c echo.Context) error {
	return c.JSON(stdhttp.StatusOK, map[string]string{
		jsonKeyStatus: statusOK,
	})
}

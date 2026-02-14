package http

import (
	"context"
	"file-service/internal/auth"
	"file-service/internal/config"
	"file-service/internal/domain/apikey"
	"file-service/internal/domain/client"
	"file-service/internal/domain/project"
	"file-service/internal/domain/user"
	"file-service/internal/http/handler"
	"file-service/internal/rbac/presets"
	"file-service/internal/repository"
	"file-service/internal/storage/s3"
	stdhttp "net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	jsonKeyStatus    = "status"
	statusOK         = "ok"
	requestBodyLimit = "1M"
)

type ServerDependencies struct {
	Config *config.Config
	DB     interface {
		SignupTransaction(ctx context.Context, email, passwordHash string) (*user.User, *client.Client, *project.Project, error)
		RollbackSignup(ctx context.Context, clientID uuid.UUID) error
	}
	UserRepo       repository.UserRepository
	ProjectRepo    repository.ProjectRepository
	FileRepo       repository.FileRepository
	APIKeyRepo     repository.APIKeyRepository
	ShareRepo      repository.ShareLinkRepository
	S3Client       *s3.Client
	BucketRegion   string
	JWTService     *auth.JWTService
	APIKeyService  *auth.APIKeyService
	AuthMiddleware *auth.Middleware
	RBACMiddleware *auth.RBACMiddleware
}

type Server struct {
	echo *echo.Echo
	deps *ServerDependencies
}

func NewServer(deps *ServerDependencies) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Server.ReadTimeout = deps.Config.Server.ReadTimeout
	e.Server.WriteTimeout = deps.Config.Server.WriteTimeout

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit(requestBodyLimit))
	e.Use(middleware.Secure())

	authHandler := handler.NewAuthHandler(deps.UserRepo, deps.DB, deps.S3Client, deps.JWTService, deps.BucketRegion)
	projectHandler := handler.NewProjectHandler(deps.ProjectRepo, deps.UserRepo, deps.S3Client, deps.BucketRegion)
	fileHandler := handler.NewFileHandler(deps.FileRepo, deps.ProjectRepo, deps.S3Client)
	apiKeyHandler := handler.NewAPIKeyHandler(deps.APIKeyRepo)
	shareHandler := handler.NewShareHandler(deps.ShareRepo, deps.FileRepo, deps.ProjectRepo, deps.S3Client)

	e.POST("/auth/signup", authHandler.Signup)
	e.POST("/auth/login", authHandler.Login)
	e.GET("/shares/:token/download-url", shareHandler.GetDownloadURLByShareToken)
	e.GET("/health", healthCheck)

	api := e.Group("/api")
	jwtAPI := api.Group("")
	jwtAPI.Use(deps.AuthMiddleware.RequireJWT())

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

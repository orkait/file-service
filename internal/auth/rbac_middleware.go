package auth

import (
	"file-service/internal/rbac"
	"file-service/internal/rbac/presets"
	"file-service/internal/repository"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type RBACMiddleware struct {
	rbacChecker *rbac.Checker
	projectRepo repository.ProjectRepository
	fileRepo    repository.FileRepository
}

func NewRBACMiddleware(projectRepo repository.ProjectRepository, fileRepo repository.FileRepository) *RBACMiddleware {
	return &RBACMiddleware{
		rbacChecker: rbac.MustNew(presets.FileManagement()),
		projectRepo: projectRepo,
		fileRepo:    fileRepo,
	}
}

func (m *RBACMiddleware) resolveJWTSubject(c echo.Context, projectID uuid.UUID) (*rbac.AuthSubject, error) {
	userID, err := GetUserID(c)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, msgUserNotAuthenticated)
	}

	member, err := m.projectRepo.GetMember(c.Request().Context(), projectID, userID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, msgProjectNotFound)
	}

	subject := &rbac.AuthSubject{
		Type:     rbac.AuthTypeJWT,
		UserRole: rbac.Role(member.Role),
	}

	return subject, nil
}

func (m *RBACMiddleware) RequireProjectRole(minRole rbac.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if GetAuthType(c) == AuthTypeAPIKey {
				return next(c)
			}

			projectID, err := extractProjectID(c)
			if err != nil {
				return respondError(c, http.StatusBadRequest, msgProjectIDRequired)
			}

			subject, err := m.resolveJWTSubject(c, projectID)
			if err != nil {
				return handleHTTPError(c, err)
			}

			if err := m.rbacChecker.RequireRole(subject, minRole); err != nil {
				return respondError(c, http.StatusForbidden, err.Error())
			}

			return next(c)
		}
	}
}

func (m *RBACMiddleware) RequireProjectRoleForFile(minRole rbac.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if GetAuthType(c) == AuthTypeAPIKey {
				return next(c)
			}

			fileIDStr := c.Param(paramID)
			if fileIDStr == "" {
				return respondError(c, http.StatusBadRequest, msgFileIDRequired)
			}

			fileID, err := uuid.Parse(fileIDStr)
			if err != nil {
				return respondError(c, http.StatusBadRequest, msgInvalidFileID)
			}

			file, err := m.fileRepo.GetByID(c.Request().Context(), fileID)
			if err != nil {
				return respondError(c, http.StatusNotFound, msgFileNotFound)
			}

			subject, err := m.resolveJWTSubject(c, file.ProjectID)
			if err != nil {
				return respondError(c, http.StatusNotFound, msgFileNotFound)
			}

			if err := m.rbacChecker.RequireRole(subject, minRole); err != nil {
				return respondError(c, http.StatusForbidden, err.Error())
			}

			c.Set(ContextKeyProjectID, file.ProjectID)

			return next(c)
		}
	}
}

func extractProjectID(c echo.Context) (uuid.UUID, error) {
	if projectID, ok := c.Get(ContextKeyProjectID).(uuid.UUID); ok && projectID != uuid.Nil {
		return projectID, nil
	}

	if id := c.Param(paramProjectID); id != "" {
		return uuid.Parse(id)
	}
	if id := c.Param(paramID); id != "" {
		return uuid.Parse(id)
	}

	return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, msgProjectIDNotFound)
}

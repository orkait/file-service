package auth

import (
	"file-service/internal/domain/apikey"
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
			authType := GetAuthType(c)

			if authType == AuthTypeAPIKey {
				// For API keys, validate project scope AND permissions BEFORE proceeding
				if err := m.validateAPIKeyProjectScope(c); err != nil {
					return err
				}
				if err := m.validateAPIKeyPermission(c, minRole); err != nil {
					return err
				}
				return next(c)
			}

			// For JWT, validate role
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

// validateAPIKeyProjectScope validates that the API key's project matches the requested project
// This MUST be called before any data access to prevent information disclosure
func (m *RBACMiddleware) validateAPIKeyProjectScope(c echo.Context) error {
	// Get API key's project ID from context (set by auth middleware)
	keyProjectID, err := GetProjectID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, msgAPIKeyContextMissing)
	}

	// Extract requested project ID from route
	requestedProjectID, err := extractProjectID(c)
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgProjectIDRequired)
	}

	// Validate scope BEFORE handler executes
	if keyProjectID != requestedProjectID {
		return respondError(c, http.StatusForbidden, msgAPIKeyScopeDenied)
	}

	return nil
}

// validateAPIKeyPermission validates that the API key has the required permission
// based on the minimum role required for the operation
func (m *RBACMiddleware) validateAPIKeyPermission(c echo.Context, minRole rbac.Role) error {
	// Get API key from context
	keyRaw := c.Get(ContextKeyAPIKey)
	if keyRaw == nil {
		return respondError(c, http.StatusUnauthorized, msgAPIKeyContextMissing)
	}

	key, ok := keyRaw.(*apikey.APIKey)
	if !ok || key == nil {
		return respondError(c, http.StatusUnauthorized, msgAPIKeyContextMissing)
	}

	// Map role to required permission
	// Viewer = Read, Editor = Write, Admin = Delete (or higher)
	var requiredPermission apikey.Permission
	switch minRole {
	case presets.RoleViewer:
		requiredPermission = apikey.PermissionRead
	case presets.RoleEditor:
		requiredPermission = apikey.PermissionWrite
	case presets.RoleAdmin:
		// Admin operations require delete permission
		requiredPermission = apikey.PermissionDelete
	default:
		return respondError(c, http.StatusForbidden, msgAPIKeyPermissionDenied)
	}

	// Check if API key has the required permission
	if !key.HasPermission(requiredPermission) {
		return respondError(c, http.StatusForbidden, msgAPIKeyPermissionDenied)
	}

	return nil
}

func (m *RBACMiddleware) RequireProjectRoleForFile(minRole rbac.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
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

			authType := GetAuthType(c)

			if authType == AuthTypeAPIKey {
				// For API keys, validate project scope AND permissions BEFORE proceeding
				keyProjectID, getErr := GetProjectID(c)
				if getErr != nil {
					return respondError(c, http.StatusUnauthorized, msgAPIKeyContextMissing)
				}

				if file.ProjectID != keyProjectID {
					// Return same error as not found to prevent enumeration
					return respondError(c, http.StatusNotFound, msgFileNotFound)
				}

				// Validate API key has required permission
				if err := m.validateAPIKeyPermission(c, minRole); err != nil {
					return respondError(c, http.StatusForbidden, msgAPIKeyPermissionDenied)
				}

				c.Set(ContextKeyProjectID, file.ProjectID)
				return next(c)
			}

			// For JWT, validate role
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

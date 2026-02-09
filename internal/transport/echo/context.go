package echo

import (
	"file-service/internal/rbac"
	"fmt"

	"github.com/labstack/echo/v4"
)

// extractAuthSubject builds an AuthSubject from Echo context values
func extractAuthSubject(c echo.Context, checker *rbac.Checker) (*rbac.AuthSubject, error) {
	authType := getAuthType(c)

	switch authType {
	case rbac.AuthTypeJWT:
		role, err := getUserRole(c, checker)
		if err != nil {
			return nil, err
		}
		return &rbac.AuthSubject{
			Type:     rbac.AuthTypeJWT,
			UserRole: role,
		}, nil

	case rbac.AuthTypeAPIKey:
		permissions, err := getAPIKeyPermissions(c)
		if err != nil {
			return nil, err
		}
		return &rbac.AuthSubject{
			Type:        rbac.AuthTypeAPIKey,
			Permissions: permissions,
		}, nil

	default:
		return nil, fmt.Errorf("unknown or missing auth type in context")
	}
}

// getAuthType determines the authentication method from context
func getAuthType(c echo.Context) rbac.AuthType {
	authType := c.Get(ContextKeyAuthType)
	if authType == nil {
		return ""
	}
	if str, ok := authType.(string); ok {
		return rbac.AuthType(str)
	}
	return ""
}

// getUserRole extracts and validates the user role from context
func getUserRole(c echo.Context, checker *rbac.Checker) (rbac.Role, error) {
	userRole := c.Get(ContextKeyUserRole)
	if userRole == nil {
		return "", fmt.Errorf("user role not found in context")
	}
	if str, ok := userRole.(string); ok {
		return checker.ValidateRole(str)
	}
	return "", fmt.Errorf("user role in context is not a string")
}

// getAPIKeyPermissions extracts API key permissions from context
func getAPIKeyPermissions(c echo.Context) ([]rbac.Permission, error) {
	permissions := c.Get(ContextKeyAPIKeyPermissions)
	if permissions == nil {
		return nil, fmt.Errorf("API key permissions not found in context")
	}
	if perms, ok := permissions.([]rbac.Permission); ok {
		return perms, nil
	}
	return nil, fmt.Errorf("API key permissions in context are not []Permission")
}

// SetAuthSubject stores an AuthSubject in the Echo context
func SetAuthSubject(c echo.Context, subject *rbac.AuthSubject) {
	if subject == nil {
		return
	}
	switch subject.Type {
	case rbac.AuthTypeJWT:
		c.Set(ContextKeyAuthType, string(rbac.AuthTypeJWT))
		c.Set(ContextKeyUserRole, string(subject.UserRole))
	case rbac.AuthTypeAPIKey:
		c.Set(ContextKeyAuthType, string(rbac.AuthTypeAPIKey))
		c.Set(ContextKeyAPIKeyPermissions, subject.Permissions)
	}
}

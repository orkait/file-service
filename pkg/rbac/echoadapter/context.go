package echoadapter

import (
	"file-service/pkg/rbac"
	"fmt"

	"github.com/labstack/echo/v4"
)

// Context keys used by the authentication middleware.
const (
	ContextKeyAuthType          = "auth_type"
	ContextKeyUserRole          = "user_role"
	ContextKeyAPIKeyPermissions = "api_key_permissions"
)

// ExtractAuthSubject builds an AuthSubject from Echo context values.
// It takes a checker because role validation is config-driven.
func ExtractAuthSubject(c echo.Context, checker *rbac.RBACChecker) (*rbac.AuthSubject, error) {
	authType := GetAuthType(c)

	switch authType {
	case rbac.AuthTypeJWT:
		role, err := GetUserRole(c, checker)
		if err != nil {
			return nil, err
		}
		return &rbac.AuthSubject{
			Type:     rbac.AuthTypeJWT,
			UserRole: role,
		}, nil

	case rbac.AuthTypeAPIKey:
		permissions, err := GetAPIKeyPermissions(c)
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

// GetAuthType determines the authentication method from context.
func GetAuthType(c echo.Context) rbac.AuthType {
	authType := c.Get(ContextKeyAuthType)
	if authType == nil {
		return ""
	}
	if str, ok := authType.(string); ok {
		return rbac.AuthType(str)
	}
	return ""
}

// GetUserRole extracts and validates the user role from context.
func GetUserRole(c echo.Context, checker *rbac.RBACChecker) (rbac.Role, error) {
	userRole := c.Get(ContextKeyUserRole)
	if userRole == nil {
		return "", fmt.Errorf("user role not found in context")
	}
	if str, ok := userRole.(string); ok {
		return checker.ValidateRole(str)
	}
	return "", fmt.Errorf("user role in context is not a string")
}

// GetAPIKeyPermissions extracts API key permissions from context.
func GetAPIKeyPermissions(c echo.Context) ([]rbac.Permission, error) {
	permissions := c.Get(ContextKeyAPIKeyPermissions)
	if permissions == nil {
		return nil, fmt.Errorf("API key permissions not found in context")
	}
	if perms, ok := permissions.([]rbac.Permission); ok {
		return perms, nil
	}
	return nil, fmt.Errorf("API key permissions in context are not []Permission")
}

// SetAuthSubject stores an AuthSubject in the Echo context.
// This is a helper for testing and middleware.
func SetAuthSubject(c echo.Context, subject *rbac.AuthSubject) {
	if subject == nil {
		return
	}
	if subject.Type == rbac.AuthTypeJWT {
		c.Set(ContextKeyAuthType, string(rbac.AuthTypeJWT))
		c.Set(ContextKeyUserRole, string(subject.UserRole))
	} else if subject.Type == rbac.AuthTypeAPIKey {
		c.Set(ContextKeyAuthType, string(rbac.AuthTypeAPIKey))
		c.Set(ContextKeyAPIKeyPermissions, subject.Permissions)
	}
}

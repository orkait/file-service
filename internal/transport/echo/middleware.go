package echo

import (
	"file-service/internal/rbac"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

const (
	ContextKeyAuthType          = "auth_type"
	ContextKeyUserRole          = "user_role"
	ContextKeyAPIKeyPermissions = "api_key_permissions"
)

// RequireAction creates middleware that enforces an action on a resource
func RequireAction(checker *rbac.Checker, resource rbac.Resource, action rbac.Action) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			subject, err := extractAuthSubject(c, checker)
			if err != nil {
				log.Printf("rbac: auth extraction failed: %v", err)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Unauthorized",
				})
			}

			if err := checker.Authorize(subject, resource, action); err != nil {
				log.Printf("rbac: authorization denied: %v", err)
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Forbidden",
				})
			}

			return next(c)
		}
	}
}

// RequireRole creates middleware that enforces a minimum role requirement
func RequireRole(checker *rbac.Checker, minRole rbac.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			subject, err := extractAuthSubject(c, checker)
			if err != nil {
				log.Printf("rbac: auth extraction failed: %v", err)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Unauthorized",
				})
			}

			if err := checker.RequireRole(subject, minRole); err != nil {
				log.Printf("rbac: role check denied: %v", err)
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Forbidden",
				})
			}

			return next(c)
		}
	}
}

// RequirePermission creates middleware for permission checks
func RequirePermission(checker *rbac.Checker, resource rbac.Resource, permission rbac.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			subject, err := extractAuthSubject(c, checker)
			if err != nil {
				log.Printf("rbac: auth extraction failed: %v", err)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Unauthorized",
				})
			}

			switch subject.Type {
			case rbac.AuthTypeAPIKey:
				if !checker.HasPermission(subject.Permissions, permission) {
					log.Printf("rbac: API key lacks permission %q", permission)
					return c.JSON(http.StatusForbidden, map[string]string{
						"error": "Forbidden",
					})
				}
			case rbac.AuthTypeJWT:
				action := checker.PermissionToAction(permission)
				if action == "" {
					log.Printf("rbac: no action mapping for permission %q", permission)
					return c.JSON(http.StatusInternalServerError, map[string]string{
						"error": "Internal server error",
					})
				}
				if err := checker.Authorize(subject, resource, action); err != nil {
					log.Printf("rbac: authorization denied: %v", err)
					return c.JSON(http.StatusForbidden, map[string]string{
						"error": "Forbidden",
					})
				}
			default:
				log.Printf("rbac: unknown auth type %q", subject.Type)
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Forbidden",
				})
			}

			return next(c)
		}
	}
}

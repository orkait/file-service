package echoadapter

import (
	"log"
	"net/http"

	"file-service/pkg/rbac"

	"github.com/labstack/echo/v4"
)

// RequireAction creates middleware that enforces an action on a resource.
// Returns 403 Forbidden if the subject cannot perform the action.
func RequireAction(checker *rbac.RBACChecker, resource rbac.Resource, action rbac.Action) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			subject, err := ExtractAuthSubject(c, checker)
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

// RequireRole creates middleware that enforces a minimum role requirement.
// Returns 403 Forbidden if the user doesn't have the required role.
func RequireRole(checker *rbac.RBACChecker, minRole rbac.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			subject, err := ExtractAuthSubject(c, checker)
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

// RequirePermission creates middleware for permission checks.
// For API keys it checks the permission directly; for JWT users it maps the
// permission to a resource/action check on the specified resource.
func RequirePermission(checker *rbac.RBACChecker, resource rbac.Resource, permission rbac.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			subject, err := ExtractAuthSubject(c, checker)
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

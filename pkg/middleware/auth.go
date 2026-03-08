package middleware

import (
	"file-service/pkg/auth"
	"file-service/pkg/repository"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// JWTAuth middleware validates JWT access tokens
func JWTAuth(secret string, clientRepo *repository.ClientRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization format"})
			}

			token := parts[1]
			claims, err := auth.ValidateToken(token, secret)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
			}

			if clientRepo != nil {
				active, err := clientRepo.IsClientActive(claims.ClientID)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to verify account status"})
				}
				if !active {
					if c.Request().Method == http.MethodDelete && c.Path() == "/api/clients/me" {
						// Allow paused clients to complete force-delete with an already-issued token.
						c.Set("client_id", claims.ClientID)
						c.Set("email", claims.Email)
						return next(c)
					}
					return c.JSON(http.StatusForbidden, map[string]string{"error": "account is paused"})
				}
			}

			// Store client info in context
			c.Set("client_id", claims.ClientID)
			c.Set("email", claims.Email)

			return next(c)
		}
	}
}

// APIKeyAuth middleware validates API keys
func APIKeyAuth(apiKeyRepo *repository.APIKeyRepository, clientRepo *repository.ClientRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := c.Request().Header.Get("X-API-Key")
			if apiKey == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing API key"})
			}

			keyData, err := apiKeyRepo.ValidateAPIKey(apiKey)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid API key"})
			}

			if clientRepo != nil {
				active, err := clientRepo.IsClientActive(keyData.ClientID)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to verify account status"})
				}
				if !active {
					return c.JSON(http.StatusForbidden, map[string]string{"error": "account is paused"})
				}
			}

			// Store API key info in context
			c.Set("client_id", keyData.ClientID)
			c.Set("project_id", keyData.ProjectID)
			c.Set("permissions", keyData.Permissions)

			return next(c)
		}
	}
}

// CheckPermission middleware checks if API key has required permission
func CheckPermission(required string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			permissions, ok := c.Get("permissions").([]string)
			if !ok {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "no permissions found"})
			}

			hasPermission := false
			for _, perm := range permissions {
				if perm == required || perm == "admin" {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "insufficient permissions"})
			}

			return next(c)
		}
	}
}

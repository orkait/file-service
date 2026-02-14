package auth

import (
	"context"
	"file-service/internal/domain/apikey"
	"file-service/internal/repository"
	apperrors "file-service/pkg/errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Middleware struct {
	jwtService    *JWTService
	apiKeyService *APIKeyService
	apiKeyRepo    repository.APIKeyRepository
}

const apiKeyLastUsedUpdateTimeout = 500 * time.Millisecond

func NewMiddleware(jwtService *JWTService, apiKeyService *APIKeyService, apiKeyRepo repository.APIKeyRepository) *Middleware {
	return &Middleware{
		jwtService:    jwtService,
		apiKeyService: apiKeyService,
		apiKeyRepo:    apiKeyRepo,
	}
}

func (m *Middleware) RequireJWT() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := extractBearerToken(c)
			if token == "" {
				return respondError(c, http.StatusUnauthorized, msgMissingAuthorization)
			}

			claims, err := m.jwtService.Verify(token)
			if err != nil {
				return respondError(c, http.StatusUnauthorized, msgInvalidOrExpiredToken)
			}

			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyClientID, claims.ClientID)
			c.Set(ContextKeyAuthType, AuthTypeJWT)

			return next(c)
		}
	}
}

func (m *Middleware) RequireAPIKey(permission apikey.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKeyString := extractAPIKey(c)
			if apiKeyString == "" {
				return respondError(c, http.StatusUnauthorized, msgMissingAPIKey)
			}

			key, err := m.verifyAPIKey(c.Request().Context(), apiKeyString)
			if err != nil {
				return respondError(c, http.StatusUnauthorized, msgInvalidAPIKey)
			}

			if err := m.apiKeyService.ValidatePermissions(key, permission); err != nil {
				return respondError(c, http.StatusForbidden, err.Error())
			}

			c.Set(ContextKeyProjectID, key.ProjectID)
			c.Set(ContextKeyAuthType, AuthTypeAPIKey)
			c.Set(ContextKeyAPIKey, key)

			updateCtx, cancel := context.WithTimeout(context.Background(), apiKeyLastUsedUpdateTimeout)
			defer cancel()
			if err := m.apiKeyRepo.UpdateLastUsed(updateCtx, key.ID); err != nil {
				c.Logger().Warnf("failed to update API key last_used_at for key %s: %v", key.ID, err)
			}

			return next(c)
		}
	}
}

func extractBearerToken(c echo.Context) string {
	authHeader := c.Request().Header.Get(headerAuthorization)
	if authHeader == "" {
		return ""
	}

	parts := strings.Fields(authHeader)
	if len(parts) != authHeaderParts || strings.ToLower(parts[0]) != bearerScheme {
		return ""
	}

	return parts[1]
}

func extractAPIKey(c echo.Context) string {
	return strings.TrimSpace(c.Request().Header.Get(headerAPIKey))
}

func (m *Middleware) verifyAPIKey(ctx context.Context, keyString string) (*apikey.APIKey, error) {
	if !strings.HasPrefix(keyString, apiKeyPrefix) {
		return nil, apperrors.Unauthorized(msgInvalidAPIKeyFormat)
	}

	hash := HashKey(keyString)
	key, err := m.apiKeyRepo.GetByHash(ctx, hash)
	if err != nil {
		return nil, apperrors.Unauthorized(msgInvalidAPIKey)
	}

	if !key.IsActive() {
		if key.RevokedAt != nil {
			return nil, apperrors.Revoked(msgAPIKeyRevoked)
		}
		return nil, apperrors.Expired(msgAPIKeyExpired)
	}

	return key, nil
}

func GetUserID(c echo.Context) (uuid.UUID, error) {
	userID := c.Get(ContextKeyUserID)
	if userID == nil {
		return uuid.Nil, apperrors.Unauthorized(msgUserNotAuthenticated)
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, apperrors.InternalServer(msgInvalidUserIDCtx, nil)
	}

	return id, nil
}

func GetClientID(c echo.Context) (uuid.UUID, error) {
	clientID := c.Get(ContextKeyClientID)
	if clientID == nil {
		return uuid.Nil, apperrors.Unauthorized(msgClientNotFound)
	}

	id, ok := clientID.(uuid.UUID)
	if !ok {
		return uuid.Nil, apperrors.InternalServer(msgInvalidClientIDCtx, nil)
	}

	return id, nil
}

func GetProjectID(c echo.Context) (uuid.UUID, error) {
	projectID := c.Get(ContextKeyProjectID)
	if projectID == nil {
		return uuid.Nil, apperrors.Unauthorized(msgProjectNotFound)
	}

	id, ok := projectID.(uuid.UUID)
	if !ok {
		return uuid.Nil, apperrors.InternalServer(msgInvalidProjectIDCtx, nil)
	}

	return id, nil
}

func GetAuthType(c echo.Context) AuthType {
	authType := c.Get(ContextKeyAuthType)
	if authType == nil {
		return ""
	}

	t, ok := authType.(AuthType)
	if !ok {
		return ""
	}

	return t
}

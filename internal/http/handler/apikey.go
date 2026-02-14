package handler

import (
	"file-service/internal/auth"
	"file-service/internal/domain/apikey"
	"file-service/internal/types"
	"file-service/pkg/token"
	"file-service/pkg/validator"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type APIKeyHandler struct {
	apiKeyRepo  APIKeyRepository
	auditLogger types.AuditLogger
}

func NewAPIKeyHandler(apiKeyRepo APIKeyRepository, auditLogger types.AuditLogger) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyRepo:  apiKeyRepo,
		auditLogger: auditLogger,
	}
}

type CreateAPIKeyRequest struct {
	Name        string     `json:"name"`
	Permissions []string   `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type APIKeyResponse struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Name        string     `json:"name"`
	KeyPrefix   string     `json:"key_prefix"`
	Permissions []string   `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

type CreateAPIKeyResponse struct {
	APIKey APIKeyResponse `json:"api_key"`
	Key    string         `json:"key"`
}

func (h *APIKeyHandler) CreateAPIKey(c echo.Context) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, err.Error())
	}

	projectID, err := uuid.Parse(c.Param(paramProjectID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidProjectIDParam)
	}

	var req CreateAPIKeyRequest
	if err := bindStrictJSON(c, &req); err != nil {
		return handleHTTPError(c, err)
	}
	req.Name = strings.TrimSpace(req.Name)

	if err = validator.APIKeyName(req.Name); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}
	if len(req.Permissions) == 0 {
		return respondError(c, http.StatusBadRequest, msgPermissionsRequired)
	}
	if req.ExpiresAt != nil && !req.ExpiresAt.After(time.Now().UTC()) {
		return respondError(c, http.StatusBadRequest, msgAPIKeyExpiryInPast)
	}

	seen := make(map[string]struct{}, len(req.Permissions))
	permissions := make([]apikey.Permission, 0, len(req.Permissions))
	for _, permission := range req.Permissions {
		permission = strings.ToLower(strings.TrimSpace(permission))
		if _, dup := seen[permission]; dup {
			continue
		}
		seen[permission] = struct{}{}
		perm := apikey.Permission(permission)
		if err = perm.Validate(); err != nil {
			return respondError(c, http.StatusBadRequest, err.Error())
		}
		permissions = append(permissions, perm)
	}

	plainKey, err := token.GenerateAPIKey()
	if err != nil {
		return respondError(c, http.StatusInternalServerError, msgGenerateAPIKeyFail)
	}

	keyPrefix := token.ExtractPrefix(plainKey, apiKeyPrefixLength)
	keyHash := auth.HashKey(plainKey)

	keyRecord, err := h.apiKeyRepo.Create(c.Request().Context(), apikey.CreateAPIKeyInput{
		ProjectID:   projectID,
		Name:        req.Name,
		KeyHash:     keyHash,
		KeyPrefix:   keyPrefix,
		Permissions: permissions,
		ExpiresAt:   req.ExpiresAt,
		CreatedBy:   userID,
	})
	if err != nil {
		if h.auditLogger != nil {
			_ = h.auditLogger.LogError(c, "api_key", nil, "create", err)
		}
		return respondError(c, http.StatusInternalServerError, msgCreateAPIKeyFail)
	}

	// Log successful API key creation
	if h.auditLogger != nil {
		metadata := map[string]any{
			"api_key_id":  keyRecord.ID.String(),
			"project_id":  projectID.String(),
			"name":        req.Name,
			"permissions": req.Permissions,
			"key_prefix":  keyPrefix,
		}
		_ = h.auditLogger.LogFromContext(c, "api_key", &keyRecord.ID, "create", "success", metadata)
	}

	return c.JSON(http.StatusCreated, CreateAPIKeyResponse{
		APIKey: toAPIKeyResponse(keyRecord),
		Key:    plainKey,
	})
}

func (h *APIKeyHandler) ListAPIKeys(c echo.Context) error {
	projectID, err := uuid.Parse(c.Param(paramProjectID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidProjectIDParam)
	}

	keys, err := h.apiKeyRepo.ListByProjectID(c.Request().Context(), projectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, msgListAPIKeysFail)
	}

	out := make([]APIKeyResponse, 0, len(keys))
	for _, key := range keys {
		out = append(out, toAPIKeyResponse(key))
	}

	return c.JSON(http.StatusOK, out)
}

func (h *APIKeyHandler) RevokeAPIKey(c echo.Context) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, err.Error())
	}

	projectID, err := uuid.Parse(c.Param(paramProjectID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidProjectIDParam)
	}

	keyID, err := uuid.Parse(c.Param(paramID))
	if err != nil {
		return respondError(c, http.StatusBadRequest, msgInvalidAPIKeyID)
	}

	keyRecord, err := h.apiKeyRepo.GetByID(c.Request().Context(), keyID)
	if err != nil {
		return respondError(c, http.StatusNotFound, msgAPIKeyNotFound)
	}
	if keyRecord.ProjectID != projectID {
		return respondError(c, http.StatusForbidden, msgAPIKeyProjectMismatch)
	}

	if err := h.apiKeyRepo.Revoke(c.Request().Context(), apikey.RevokeAPIKeyInput{
		ID:        keyID,
		RevokedBy: userID,
	}); err != nil {
		if h.auditLogger != nil {
			_ = h.auditLogger.LogError(c, "api_key", &keyID, "revoke", err)
		}
		return respondError(c, http.StatusInternalServerError, msgRevokeAPIKeyFail)
	}

	// Log successful API key revocation
	if h.auditLogger != nil {
		metadata := map[string]any{
			"api_key_id": keyID.String(),
			"project_id": projectID.String(),
			"name":       keyRecord.Name,
		}
		_ = h.auditLogger.LogFromContext(c, "api_key", &keyID, "revoke", "success", metadata)
	}

	return respondMessage(c, http.StatusOK, msgAPIKeyRevoked)
}

func toAPIKeyResponse(key *apikey.APIKey) APIKeyResponse {
	perms := make([]string, 0, len(key.Permissions))
	for _, permission := range key.Permissions {
		perms = append(perms, string(permission))
	}

	return APIKeyResponse{
		ID:          key.ID.String(),
		ProjectID:   key.ProjectID.String(),
		Name:        key.Name,
		KeyPrefix:   key.KeyPrefix,
		Permissions: perms,
		ExpiresAt:   key.ExpiresAt,
		CreatedAt:   key.CreatedAt,
		RevokedAt:   key.RevokedAt,
		LastUsedAt:  key.LastUsedAt,
	}
}

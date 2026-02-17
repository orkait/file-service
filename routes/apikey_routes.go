package routes

import (
	"file-service/pkg/repository"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type APIKeyRoutes struct {
	apiKeyRepo *repository.APIKeyRepository
}

func NewAPIKeyRoutes(apiKeyRepo *repository.APIKeyRepository) *APIKeyRoutes {
	return &APIKeyRoutes{apiKeyRepo: apiKeyRepo}
}

// CreateAPIKey generates a new API key
func (ar *APIKeyRoutes) CreateAPIKey(c echo.Context) error {
	clientID := c.Get("client_id").(string)

	var req struct {
		ProjectID   string   `json:"project_id"`
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
		ExpiresIn   *int     `json:"expires_in_days"` // optional, days until expiration
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.ProjectID == "" || req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id and name required"})
	}

	if len(req.Permissions) == 0 {
		req.Permissions = []string{"read"} // default permission
	}

	var expiresAt *time.Time
	if req.ExpiresIn != nil {
		exp := time.Now().AddDate(0, 0, *req.ExpiresIn)
		expiresAt = &exp
	}

	key, apiKey, err := ar.apiKeyRepo.CreateAPIKey(clientID, req.ProjectID, req.Name, req.Permissions, expiresAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create API key"})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"api_key": key, // Only shown once!
		"details": apiKey,
	})
}

// GetAPIKeys retrieves all API keys for authenticated client
func (ar *APIKeyRoutes) GetAPIKeys(c echo.Context) error {
	clientID := c.Get("client_id").(string)

	keys, err := ar.apiKeyRepo.GetAPIKeysByClientID(clientID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get API keys"})
	}

	return c.JSON(http.StatusOK, keys)
}

// RevokeAPIKey deactivates an API key
func (ar *APIKeyRoutes) RevokeAPIKey(c echo.Context) error {
	clientID := c.Get("client_id").(string)
	keyID := c.Param("id")

	err := ar.apiKeyRepo.RevokeAPIKey(keyID, clientID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "API key not found"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "API key revoked"})
}

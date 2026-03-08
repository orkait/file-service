package routes

import (
	"file-service/pkg/auth"
	"file-service/pkg/repository"
	"file-service/pkg/s3"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

const accountDeletionGraceDays = 90

type ClientRoutes struct {
	clientRepo *repository.ClientRepository
	assetRepo  *repository.AssetRepository
	s3Client   *s3.S3
}

func NewClientRoutes(clientRepo *repository.ClientRepository, assetRepo *repository.AssetRepository, s3Client *s3.S3) *ClientRoutes {
	return &ClientRoutes{
		clientRepo: clientRepo,
		assetRepo:  assetRepo,
		s3Client:   s3Client,
	}
}

func parseForceDelete(c echo.Context) bool {
	candidates := []string{
		c.QueryParam("force_delete"),
		c.QueryParam("force-delete"),
		c.QueryParam("forceDelete"),
	}

	for _, raw := range candidates {
		if raw == "" {
			continue
		}
		value, err := strconv.ParseBool(raw)
		if err == nil {
			return value
		}
	}

	return false
}

func parseIntQueryParam(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func normalizeClientEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (cr *ClientRoutes) deleteClientAndResources(clientID string) error {
	keys, err := cr.assetRepo.GetAllS3KeysByClientID(clientID)
	if err != nil {
		return err
	}

	for _, key := range keys {
		if err := cr.s3Client.DeleteObject(key); err != nil {
			return fmt.Errorf("failed to delete object %s: %w", key, err)
		}
	}

	if err := cr.clientRepo.DeleteClient(clientID); err != nil {
		return err
	}

	return nil
}

// ListClients returns active clients for collaboration/invite lookups.
func (cr *ClientRoutes) ListClients(c echo.Context) error {
	limit := parseIntQueryParam(c.QueryParam("limit"), 50)
	offset := parseIntQueryParam(c.QueryParam("offset"), 0)
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	q := strings.TrimSpace(c.QueryParam("q"))
	clients, err := cr.clientRepo.ListClients(limit, offset, q)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list clients"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"clients": clients,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetMyClient returns the authenticated client account.
func (cr *ClientRoutes) GetMyClient(c echo.Context) error {
	clientID := c.Get("client_id").(string)

	client, err := cr.clientRepo.GetClientByID(clientID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "client not found"})
	}

	return c.JSON(http.StatusOK, client)
}

// GetClientByID returns an active client by id.
func (cr *ClientRoutes) GetClientByID(c echo.Context) error {
	requesterID := c.Get("client_id").(string)
	clientID := strings.TrimSpace(c.Param("id"))
	if clientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "client id required"})
	}

	client, err := cr.clientRepo.GetClientByID(clientID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "client not found"})
	}

	if client.Status != "active" && client.ID != requesterID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "client not found"})
	}

	return c.JSON(http.StatusOK, client)
}

// UpdateMyClient updates authenticated client's profile (name/email/password).
func (cr *ClientRoutes) UpdateMyClient(c echo.Context) error {
	clientID := c.Get("client_id").(string)

	var req struct {
		Name     *string `json:"name"`
		Email    *string `json:"email"`
		Password *string `json:"password"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	current, err := cr.clientRepo.GetClientByID(clientID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "client not found"})
	}

	needsProfileUpdate := false
	newName := current.Name
	newEmail := current.Email

	if req.Name != nil {
		trimmedName := strings.TrimSpace(*req.Name)
		if trimmedName == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "name cannot be empty"})
		}
		newName = trimmedName
		needsProfileUpdate = true
	}

	if req.Email != nil {
		normalizedEmail := normalizeClientEmail(*req.Email)
		if normalizedEmail == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "email cannot be empty"})
		}
		newEmail = normalizedEmail
		needsProfileUpdate = true
	}

	if !needsProfileUpdate && req.Password == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "nothing to update"})
	}

	if needsProfileUpdate {
		_, err := cr.clientRepo.UpdateClientProfile(clientID, newName, newEmail)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return c.JSON(http.StatusConflict, map[string]string{"error": "email already exists"})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update client profile"})
		}
	}

	if req.Password != nil {
		trimmedPassword := strings.TrimSpace(*req.Password)
		if trimmedPassword == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "password cannot be empty"})
		}

		passwordHash, err := auth.HashPassword(trimmedPassword)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to process password"})
		}

		if err := cr.clientRepo.UpdateClientPassword(clientID, passwordHash); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update password"})
		}
	}

	updated, err := cr.clientRepo.GetClientByID(clientID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch updated client"})
	}

	return c.JSON(http.StatusOK, updated)
}

// DeleteMyAccount pauses account for 90 days or force-deletes it immediately.
func (cr *ClientRoutes) DeleteMyAccount(c echo.Context) error {
	clientID := c.Get("client_id").(string)
	forceDelete := parseForceDelete(c)

	client, err := cr.clientRepo.GetClientByID(clientID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "client not found"})
	}

	if forceDelete {
		if err := cr.deleteClientAndResources(clientID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to force delete account"})
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "account and associated data deleted"})
	}

	if client.Status == "paused" && client.ScheduledDeletionAt != nil {
		return c.JSON(http.StatusOK, map[string]any{
			"message":               "account already paused for deletion",
			"scheduled_deletion_at": client.ScheduledDeletionAt.UTC(),
		})
	}

	scheduledDeletionAt := time.Now().UTC().AddDate(0, 0, accountDeletionGraceDays)
	if err := cr.clientRepo.PauseClientForDeletion(clientID, scheduledDeletionAt); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to pause account"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message":               "account paused and scheduled for deletion",
		"scheduled_deletion_at": scheduledDeletionAt,
		"force_delete_hint":     "Use DELETE /api/clients/me?force_delete=true for immediate deletion",
	})
}

// CleanupDuePausedClients permanently deletes paused accounts whose grace period expired.
func (cr *ClientRoutes) CleanupDuePausedClients() (int, error) {
	clients, err := cr.clientRepo.GetClientsDueForDeletion(time.Now().UTC())
	if err != nil {
		return 0, err
	}

	deleted := 0
	failures := 0
	for _, client := range clients {
		if err := cr.deleteClientAndResources(client.ID); err != nil {
			failures++
			continue
		}
		deleted++
	}

	if failures > 0 {
		return deleted, fmt.Errorf("failed to delete %d account(s)", failures)
	}

	return deleted, nil
}

package repository

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"file-service/pkg/models"
	"fmt"
	"time"
)

type APIKeyRepository struct {
	db *sql.DB
}

func NewAPIKeyRepository(db *sql.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// GenerateAPIKey creates a new API key with format: orka-storage-<projectid>-<uuid>
func GenerateAPIKey(projectID string) string {
	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("orka-storage-%s-%s", projectID, randomStr)
}

// HashAPIKey creates SHA256 hash of API key
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// CreateAPIKey creates a new API key
func (r *APIKeyRepository) CreateAPIKey(clientID, projectID, name string, permissions []string, expiresAt *time.Time) (string, *models.APIKey, error) {
	// Generate the actual key
	key := GenerateAPIKey(projectID)
	keyHash := HashAPIKey(key)
	keyPrefix := key[:20] + "..."

	permJSON, _ := json.Marshal(permissions)

	query := `
		INSERT INTO api_keys (client_id, project_id, key_hash, key_prefix, name, permissions, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, client_id, project_id, key_prefix, name, permissions, is_active, created_at, expires_at
	`

	var apiKey models.APIKey
	var permStr string
	var projID sql.NullString

	err := r.db.QueryRow(query, clientID, projectID, keyHash, keyPrefix, name, permJSON, expiresAt).Scan(
		&apiKey.ID,
		&apiKey.ClientID,
		&projID,
		&apiKey.KeyPrefix,
		&apiKey.Name,
		&permStr,
		&apiKey.IsActive,
		&apiKey.CreatedAt,
		&apiKey.ExpiresAt,
	)

	if err != nil {
		return "", nil, fmt.Errorf("failed to create API key: %w", err)
	}

	if projID.Valid {
		apiKey.ProjectID = &projID.String
	}

	json.Unmarshal([]byte(permStr), &apiKey.Permissions)

	return key, &apiKey, nil
}

// ValidateAPIKey checks if API key is valid and returns associated data
func (r *APIKeyRepository) ValidateAPIKey(key string) (*models.APIKey, error) {
	keyHash := HashAPIKey(key)

	query := `
		SELECT id, client_id, project_id, key_prefix, name, permissions, is_active, last_used_at, created_at, expires_at
		FROM api_keys
		WHERE key_hash = $1 AND is_active = true
	`

	var apiKey models.APIKey
	var permStr string
	var projID sql.NullString
	var lastUsed sql.NullTime

	err := r.db.QueryRow(query, keyHash).Scan(
		&apiKey.ID,
		&apiKey.ClientID,
		&projID,
		&apiKey.KeyPrefix,
		&apiKey.Name,
		&permStr,
		&apiKey.IsActive,
		&lastUsed,
		&apiKey.CreatedAt,
		&apiKey.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid API key")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate API key: %w", err)
	}

	if projID.Valid {
		apiKey.ProjectID = &projID.String
	}
	if lastUsed.Valid {
		apiKey.LastUsedAt = &lastUsed.Time
	}

	json.Unmarshal([]byte(permStr), &apiKey.Permissions)

	// Check expiration
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("API key expired")
	}

	// Update last used timestamp
	go r.updateLastUsed(apiKey.ID)

	return &apiKey, nil
}

func (r *APIKeyRepository) updateLastUsed(keyID string) {
	query := `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`
	r.db.Exec(query, keyID)
}

// GetAPIKeysByClientID retrieves all API keys for a client
func (r *APIKeyRepository) GetAPIKeysByClientID(clientID string) ([]models.APIKey, error) {
	query := `
		SELECT id, client_id, project_id, key_prefix, name, permissions, is_active, last_used_at, created_at, expires_at
		FROM api_keys
		WHERE client_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		var apiKey models.APIKey
		var permStr string
		var projID sql.NullString
		var lastUsed sql.NullTime

		err := rows.Scan(
			&apiKey.ID,
			&apiKey.ClientID,
			&projID,
			&apiKey.KeyPrefix,
			&apiKey.Name,
			&permStr,
			&apiKey.IsActive,
			&lastUsed,
			&apiKey.CreatedAt,
			&apiKey.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}

		if projID.Valid {
			apiKey.ProjectID = &projID.String
		}
		if lastUsed.Valid {
			apiKey.LastUsedAt = &lastUsed.Time
		}

		json.Unmarshal([]byte(permStr), &apiKey.Permissions)
		keys = append(keys, apiKey)
	}

	return keys, nil
}

// RevokeAPIKey deactivates an API key
func (r *APIKeyRepository) RevokeAPIKey(keyID, clientID string) error {
	query := `UPDATE api_keys SET is_active = false WHERE id = $1 AND client_id = $2`
	result, err := r.db.Exec(query, keyID, clientID)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

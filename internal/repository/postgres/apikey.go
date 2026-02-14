package postgres

import (
	"context"
	"file-service/internal/domain/apikey"
	apperrors "file-service/pkg/errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type APIKeyRepository struct {
	db *DB
}

func NewAPIKeyRepository(db *DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, input apikey.CreateAPIKeyInput) (*apikey.APIKey, error) {
	query := `
		INSERT INTO api_keys (project_id, name, key_hash, key_prefix, permissions, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, project_id, name, key_hash, key_prefix, permissions, expires_at, created_by, created_at, last_used_at, revoked_at, revoked_by
	`

	k := &apikey.APIKey{}
	err := r.db.Pool.QueryRow(ctx, query, input.ProjectID, input.Name, input.KeyHash, input.KeyPrefix, input.Permissions, input.ExpiresAt, input.CreatedBy).Scan(
		&k.ID, &k.ProjectID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.Permissions, &k.ExpiresAt, &k.CreatedBy, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt, &k.RevokedBy,
	)

	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.Conflict("API key already exists")
		}
		return nil, errFailedCreateAPIKey(err)
	}

	return k, nil
}

func (r *APIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*apikey.APIKey, error) {
	query := `
		SELECT id, project_id, name, key_hash, key_prefix, permissions, expires_at, created_by, created_at, last_used_at, revoked_at, revoked_by
		FROM api_keys WHERE id = $1
	`

	k := &apikey.APIKey{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&k.ID, &k.ProjectID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.Permissions, &k.ExpiresAt, &k.CreatedBy, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt, &k.RevokedBy,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound(errAPIKeyNotFound)
		}
		return nil, errFailedGetAPIKey(err)
	}

	return k, nil
}

func (r *APIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*apikey.APIKey, error) {
	query := `
		SELECT id, project_id, name, key_hash, key_prefix, permissions, expires_at, created_by, created_at, last_used_at, revoked_at, revoked_by
		FROM api_keys WHERE key_hash = $1
	`

	k := &apikey.APIKey{}
	err := r.db.Pool.QueryRow(ctx, query, keyHash).Scan(
		&k.ID, &k.ProjectID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.Permissions, &k.ExpiresAt, &k.CreatedBy, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt, &k.RevokedBy,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound(errAPIKeyNotFound)
		}
		return nil, errFailedGetAPIKey(err)
	}

	return k, nil
}

func (r *APIKeyRepository) ListByProjectID(ctx context.Context, projectID uuid.UUID) ([]*apikey.APIKey, error) {
	query := `
		SELECT id, project_id, name, key_hash, key_prefix, permissions, expires_at, created_by, created_at, last_used_at, revoked_at, revoked_by
		FROM api_keys WHERE project_id = $1 ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, errFailedListAPIKeys(err)
	}
	defer rows.Close()

	var keys []*apikey.APIKey
	for rows.Next() {
		k := &apikey.APIKey{}
		if err := rows.Scan(&k.ID, &k.ProjectID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.Permissions, &k.ExpiresAt, &k.CreatedBy, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt, &k.RevokedBy); err != nil {
			return nil, errFailedScanAPIKey(err)
		}
		keys = append(keys, k)
	}

	return keys, rows.Err()
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := "UPDATE api_keys SET last_used_at = $1 WHERE id = $2"
	_, err := r.db.Pool.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return errFailedUpdateLastUsed(err)
	}

	return nil
}

func (r *APIKeyRepository) Revoke(ctx context.Context, input apikey.RevokeAPIKeyInput) error {
	query := "UPDATE api_keys SET revoked_at = NOW(), revoked_by = $1 WHERE id = $2"
	result, err := r.db.Pool.Exec(ctx, query, input.RevokedBy, input.ID)
	if err != nil {
		return errFailedRevokeAPIKey(err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound(errAPIKeyNotFound)
	}

	return nil
}

func (r *APIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM api_keys WHERE id = $1"
	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return errFailedDeleteAPIKey(err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound(errAPIKeyNotFound)
	}

	return nil
}

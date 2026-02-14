package postgres

import (
	"context"
	"file-service/internal/domain/client"
	apperrors "file-service/pkg/errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ClientRepository struct {
	db *DB
}

func NewClientRepository(db *DB) *ClientRepository {
	return &ClientRepository{db: db}
}

func (r *ClientRepository) Create(ctx context.Context, input client.CreateClientInput) (*client.Client, error) {
	query := `
		INSERT INTO clients (owner_user_id)
		VALUES ($1)
		RETURNING id, owner_user_id, created_at, updated_at
	`

	c := &client.Client{}
	err := r.db.Pool.QueryRow(ctx, query, input.OwnerUserID).Scan(
		&c.ID,
		&c.OwnerUserID,
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	if err != nil {
		return nil, errFailedCreateClient(err)
	}

	return c, nil
}

func (r *ClientRepository) GetByID(ctx context.Context, id uuid.UUID) (*client.Client, error) {
	query := `
		SELECT id, owner_user_id, created_at, updated_at
		FROM clients
		WHERE id = $1
	`

	c := &client.Client{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.OwnerUserID,
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound(errClientNotFound)
		}
		return nil, errFailedGetClient(err)
	}

	return c, nil
}

func (r *ClientRepository) GetByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) (*client.Client, error) {
	query := `
		SELECT id, owner_user_id, created_at, updated_at
		FROM clients
		WHERE owner_user_id = $1
	`

	c := &client.Client{}
	err := r.db.Pool.QueryRow(ctx, query, ownerUserID).Scan(
		&c.ID,
		&c.OwnerUserID,
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound(errClientNotFound)
		}
		return nil, errFailedGetClient(err)
	}

	return c, nil
}

func (r *ClientRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM clients WHERE id = $1"
	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return errFailedDeleteClient(err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound(errClientNotFound)
	}

	return nil
}

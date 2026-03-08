package repository

import (
	"database/sql"
	"file-service/pkg/models"
	"fmt"
	"time"
)

type ClientRepository struct {
	db *sql.DB
}

func NewClientRepository(db *sql.DB) *ClientRepository {
	return &ClientRepository{db: db}
}

func assignClientLifecycleFields(client *models.Client, pausedAt sql.NullTime, scheduledDeletionAt sql.NullTime) {
	if pausedAt.Valid {
		t := pausedAt.Time
		client.PausedAt = &t
	}
	if scheduledDeletionAt.Valid {
		t := scheduledDeletionAt.Time
		client.ScheduledDeletionAt = &t
	}
}

// CreateClient creates a new client
func (r *ClientRepository) CreateClient(name, email, passwordHash string) (*models.Client, error) {
	query := `
		INSERT INTO clients (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, status, paused_at, scheduled_deletion_at, created_at, updated_at
	`

	var client models.Client
	var pausedAt sql.NullTime
	var scheduledDeletionAt sql.NullTime
	err := r.db.QueryRow(query, name, email, passwordHash).Scan(
		&client.ID,
		&client.Name,
		&client.Email,
		&client.Status,
		&pausedAt,
		&scheduledDeletionAt,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	assignClientLifecycleFields(&client, pausedAt, scheduledDeletionAt)

	return &client, nil
}

// GetClientByEmail retrieves client by email
func (r *ClientRepository) GetClientByEmail(email string) (*models.Client, error) {
	query := `
		SELECT id, name, email, password_hash, status, paused_at, scheduled_deletion_at, created_at, updated_at
		FROM clients
		WHERE email = $1
	`

	var client models.Client
	var pausedAt sql.NullTime
	var scheduledDeletionAt sql.NullTime
	err := r.db.QueryRow(query, email).Scan(
		&client.ID,
		&client.Name,
		&client.Email,
		&client.PasswordHash,
		&client.Status,
		&pausedAt,
		&scheduledDeletionAt,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("client not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	assignClientLifecycleFields(&client, pausedAt, scheduledDeletionAt)

	return &client, nil
}

// GetClientByID retrieves client by ID
func (r *ClientRepository) GetClientByID(id string) (*models.Client, error) {
	query := `
		SELECT id, name, email, status, paused_at, scheduled_deletion_at, created_at, updated_at
		FROM clients
		WHERE id = $1
	`

	var client models.Client
	var pausedAt sql.NullTime
	var scheduledDeletionAt sql.NullTime
	err := r.db.QueryRow(query, id).Scan(
		&client.ID,
		&client.Name,
		&client.Email,
		&client.Status,
		&pausedAt,
		&scheduledDeletionAt,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("client not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	assignClientLifecycleFields(&client, pausedAt, scheduledDeletionAt)

	return &client, nil
}

// UpdateClientPassword updates password hash for a client.
func (r *ClientRepository) UpdateClientPassword(clientID, passwordHash string) error {
	query := `
		UPDATE clients
		SET password_hash = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.Exec(query, passwordHash, clientID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("client not found")
	}

	return nil
}

func (r *ClientRepository) IsClientActive(clientID string) (bool, error) {
	query := `
		SELECT status
		FROM clients
		WHERE id = $1
	`

	var status string
	err := r.db.QueryRow(query, clientID).Scan(&status)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get client status: %w", err)
	}

	return status == "active", nil
}

func (r *ClientRepository) PauseClientForDeletion(clientID string, scheduledDeletionAt time.Time) error {
	query := `
		UPDATE clients
		SET status = 'paused',
			paused_at = COALESCE(paused_at, NOW()),
			scheduled_deletion_at = $1,
			updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.Exec(query, scheduledDeletionAt.UTC(), clientID)
	if err != nil {
		return fmt.Errorf("failed to pause client: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("client not found")
	}

	return nil
}

func (r *ClientRepository) GetClientsDueForDeletion(now time.Time) ([]models.Client, error) {
	query := `
		SELECT id, name, email, status, paused_at, scheduled_deletion_at, created_at, updated_at
		FROM clients
		WHERE status = 'paused'
			AND scheduled_deletion_at IS NOT NULL
			AND scheduled_deletion_at <= $1
		ORDER BY scheduled_deletion_at ASC
	`

	rows, err := r.db.Query(query, now.UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to query clients due for deletion: %w", err)
	}
	defer rows.Close()

	clients := make([]models.Client, 0)
	for rows.Next() {
		var client models.Client
		var pausedAt sql.NullTime
		var scheduledDeletionAt sql.NullTime
		if err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.Email,
			&client.Status,
			&pausedAt,
			&scheduledDeletionAt,
			&client.CreatedAt,
			&client.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan client due for deletion: %w", err)
		}
		assignClientLifecycleFields(&client, pausedAt, scheduledDeletionAt)
		clients = append(clients, client)
	}

	return clients, nil
}

func (r *ClientRepository) DeleteClient(clientID string) error {
	query := `DELETE FROM clients WHERE id = $1`

	result, err := r.db.Exec(query, clientID)
	if err != nil {
		return fmt.Errorf("failed to delete client: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("client not found")
	}

	return nil
}

func (r *ClientRepository) ListClients(limit int, offset int, q string) ([]models.Client, error) {
	query := `
		SELECT id, name, email, status, paused_at, scheduled_deletion_at, created_at, updated_at
		FROM clients
		WHERE status = 'active'
			AND (
				$1 = '' OR
				name ILIKE '%' || $1 || '%' OR
				email ILIKE '%' || $1 || '%'
			)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}
	defer rows.Close()

	clients := make([]models.Client, 0)
	for rows.Next() {
		var client models.Client
		var pausedAt sql.NullTime
		var scheduledDeletionAt sql.NullTime

		if err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.Email,
			&client.Status,
			&pausedAt,
			&scheduledDeletionAt,
			&client.CreatedAt,
			&client.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan client: %w", err)
		}
		assignClientLifecycleFields(&client, pausedAt, scheduledDeletionAt)
		clients = append(clients, client)
	}

	return clients, nil
}

func (r *ClientRepository) UpdateClientProfile(clientID string, name string, email string) (*models.Client, error) {
	query := `
		UPDATE clients
		SET name = $1,
			email = $2,
			updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, email, status, paused_at, scheduled_deletion_at, created_at, updated_at
	`

	var client models.Client
	var pausedAt sql.NullTime
	var scheduledDeletionAt sql.NullTime
	if err := r.db.QueryRow(query, name, email, clientID).Scan(
		&client.ID,
		&client.Name,
		&client.Email,
		&client.Status,
		&pausedAt,
		&scheduledDeletionAt,
		&client.CreatedAt,
		&client.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("client not found")
		}
		return nil, fmt.Errorf("failed to update client profile: %w", err)
	}
	assignClientLifecycleFields(&client, pausedAt, scheduledDeletionAt)

	return &client, nil
}

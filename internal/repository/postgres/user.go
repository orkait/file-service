package postgres

import (
	"context"
	"file-service/internal/domain/user"
	apperrors "file-service/pkg/errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, input user.CreateUserInput) (*user.User, error) {
	query := `
		INSERT INTO users (email, password_hash, client_id)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, client_id, created_at, updated_at
	`

	u := &user.User{}
	err := r.db.Pool.QueryRow(ctx, query, input.Email, input.Password, input.ClientID).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.ClientID,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.Conflict("user with this email already exists")
		}
		return nil, errFailedCreateUser(err)
	}

	return u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	query := `
		SELECT id, email, password_hash, client_id, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	u := &user.User{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.ClientID,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound(errUserNotFound)
		}
		return nil, errFailedGetUser(err)
	}

	return u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	query := `
		SELECT id, email, password_hash, client_id, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	u := &user.User{}
	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.ClientID,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound(errUserNotFound)
		}
		return nil, errFailedGetUser(err)
	}

	return u, nil
}

func (r *UserRepository) GetByClientID(ctx context.Context, clientID uuid.UUID) ([]*user.User, error) {
	query := `
		SELECT id, email, password_hash, client_id, created_at, updated_at
		FROM users
		WHERE client_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, clientID)
	if err != nil {
		return nil, errFailedListUsers(err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		u := &user.User{}
		if err := rows.Scan(
			&u.ID,
			&u.Email,
			&u.PasswordHash,
			&u.ClientID,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, errFailedScanUser(err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, errIterateUsers(err)
	}

	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, id uuid.UUID, input user.UpdateUserInput) error {
	query := "UPDATE users SET updated_at = NOW()"
	args := []interface{}{id}
	argCount := 1

	if input.Email != nil {
		argCount++
		query += fmt.Sprintf(", email = $%d", argCount)
		args = append(args, *input.Email)
	}

	if input.PasswordHash != nil {
		argCount++
		query += fmt.Sprintf(", password_hash = $%d", argCount)
		args = append(args, *input.PasswordHash)
	}

	query += " WHERE id = $1"

	result, err := r.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return apperrors.Conflict("email already exists")
		}
		return errFailedUpdateUser(err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound(errUserNotFound)
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM users WHERE id = $1"

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return errFailedDeleteUser(err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound(errUserNotFound)
	}

	return nil
}

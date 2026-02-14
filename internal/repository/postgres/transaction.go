package postgres

import (
	"context"
	"file-service/internal/domain/client"
	"file-service/internal/domain/project"
	"file-service/internal/domain/user"
	apperrors "file-service/pkg/errors"
	"fmt"

	"github.com/google/uuid"
)

func (db *DB) SignupTransaction(ctx context.Context, email, passwordHash string) (*user.User, *client.Client, *project.Project, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, nil, nil, errFailedStartTransaction(err)
	}
	defer tx.Rollback(ctx)

	userID := uuid.New()
	clientID := uuid.New()

	var createdUser user.User

	userQuery := `
		INSERT INTO users (id, email, password_hash, client_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, password_hash, client_id, created_at, updated_at
	`

	err = tx.QueryRow(ctx, userQuery, userID, email, passwordHash, clientID).Scan(
		&createdUser.ID,
		&createdUser.Email,
		&createdUser.PasswordHash,
		&createdUser.ClientID,
		&createdUser.CreatedAt,
		&createdUser.UpdatedAt,
	)

	if err != nil {
		if isUniqueViolation(err) {
			return nil, nil, nil, apperrors.ErrEmailExists
		}
		return nil, nil, nil, errFailedCreateUser(err)
	}

	var createdClient client.Client

	clientQuery := `
		INSERT INTO clients (id, owner_user_id)
		VALUES ($1, $2)
		RETURNING id, owner_user_id, created_at, updated_at
	`

	err = tx.QueryRow(ctx, clientQuery, clientID, userID).Scan(
		&createdClient.ID,
		&createdClient.OwnerUserID,
		&createdClient.CreatedAt,
		&createdClient.UpdatedAt,
	)

	if err != nil {
		return nil, nil, nil, errFailedCreateClient(err)
	}

	projectID := uuid.New()
	s3BucketName := fmt.Sprintf(
		"%s-%s",
		clientID.String()[:bucketNameIDSegmentLength],
		projectID.String()[:bucketNameIDSegmentLength],
	)

	var createdProject project.Project

	projectQuery := `
		INSERT INTO projects (id, client_id, name, s3_bucket_name, is_default)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, client_id, name, s3_bucket_name, is_default, created_at, updated_at
	`

	err = tx.QueryRow(ctx, projectQuery, projectID, clientID, defaultProjectName, s3BucketName, true).Scan(
		&createdProject.ID,
		&createdProject.ClientID,
		&createdProject.Name,
		&createdProject.S3BucketName,
		&createdProject.IsDefault,
		&createdProject.CreatedAt,
		&createdProject.UpdatedAt,
	)

	if err != nil {
		return nil, nil, nil, errFailedCreateDefaultProject(err)
	}

	memberQuery := `
		INSERT INTO project_members (project_id, user_id, role, invited_by)
		VALUES ($1, $2, $3, $4)
	`

	_, err = tx.Exec(ctx, memberQuery, createdProject.ID, userID, defaultProjectMemberRole, userID)
	if err != nil {
		return nil, nil, nil, errFailedAddUserAsProjectMember(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, nil, errFailedCommitTransaction(err)
	}

	return &createdUser, &createdClient, &createdProject, nil
}

func (db *DB) RollbackSignup(ctx context.Context, clientID uuid.UUID) error {
	query := `DELETE FROM clients WHERE id = $1`
	_, err := db.Pool.Exec(ctx, query, clientID)
	if err != nil {
		return fmt.Errorf("failed to rollback signup for client %s: %w", clientID, err)
	}
	return nil
}

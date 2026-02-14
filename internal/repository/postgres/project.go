package postgres

import (
	"context"
	"file-service/internal/domain/project"
	apperrors "file-service/pkg/errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ProjectRepository struct {
	db *DB
}

func NewProjectRepository(db *DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Create(ctx context.Context, input project.CreateProjectInput) (*project.Project, error) {
	projectID := uuid.New()
	s3BucketName := fmt.Sprintf(
		"%s-%s",
		input.ClientID.String()[:bucketNameIDSegmentLength],
		projectID.String()[:bucketNameIDSegmentLength],
	)

	query := `
		INSERT INTO projects (id, client_id, name, s3_bucket_name, is_default)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, client_id, name, s3_bucket_name, is_default, created_at, updated_at
	`

	p := &project.Project{}
	err := r.db.Pool.QueryRow(ctx, query, projectID, input.ClientID, input.Name, s3BucketName, input.IsDefault).Scan(
		&p.ID, &p.ClientID, &p.Name, &p.S3BucketName, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt,
	)

	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.Conflict("project with this name already exists")
		}
		return nil, errFailedCreateProject(err)
	}

	return p, nil
}

func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*project.Project, error) {
	query := `
		SELECT id, client_id, name, s3_bucket_name, is_default, created_at, updated_at
		FROM projects WHERE id = $1
	`

	p := &project.Project{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.ClientID, &p.Name, &p.S3BucketName, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound(errProjectNotFound)
		}
		return nil, errFailedGetProject(err)
	}

	return p, nil
}

func (r *ProjectRepository) GetByClientID(ctx context.Context, clientID uuid.UUID) ([]*project.Project, error) {
	query := `
		SELECT id, client_id, name, s3_bucket_name, is_default, created_at, updated_at
		FROM projects WHERE client_id = $1 ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, clientID)
	if err != nil {
		return nil, errFailedListProjects(err)
	}
	defer rows.Close()

	var projects []*project.Project
	for rows.Next() {
		p := &project.Project{}
		if err := rows.Scan(&p.ID, &p.ClientID, &p.Name, &p.S3BucketName, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, errFailedScanProject(err)
		}
		projects = append(projects, p)
	}

	return projects, rows.Err()
}

func (r *ProjectRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*project.Project, error) {
	query := `
		SELECT p.id, p.client_id, p.name, p.s3_bucket_name, p.is_default, p.created_at, p.updated_at
		FROM projects p
		INNER JOIN project_members pm ON pm.project_id = p.id
		WHERE pm.user_id = $1
		ORDER BY p.created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, errFailedListProjectsByUser(err)
	}
	defer rows.Close()

	var projects []*project.Project
	for rows.Next() {
		p := &project.Project{}
		if err := rows.Scan(&p.ID, &p.ClientID, &p.Name, &p.S3BucketName, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, errFailedScanProject(err)
		}
		projects = append(projects, p)
	}

	return projects, rows.Err()
}

func (r *ProjectRepository) GetDefaultByClientID(ctx context.Context, clientID uuid.UUID) (*project.Project, error) {
	query := `
		SELECT id, client_id, name, s3_bucket_name, is_default, created_at, updated_at
		FROM projects WHERE client_id = $1 AND is_default = TRUE
	`

	p := &project.Project{}
	err := r.db.Pool.QueryRow(ctx, query, clientID).Scan(
		&p.ID, &p.ClientID, &p.Name, &p.S3BucketName, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound("default project not found")
		}
		return nil, errFailedGetDefaultProject(err)
	}

	return p, nil
}

func (r *ProjectRepository) Update(ctx context.Context, id uuid.UUID, input project.UpdateProjectInput) error {
	query := "UPDATE projects SET updated_at = NOW()"
	args := []interface{}{id}
	argCount := 1

	if input.Name != nil {
		argCount++
		query += fmt.Sprintf(", name = $%d", argCount)
		args = append(args, *input.Name)
	}

	query += " WHERE id = $1"

	result, err := r.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return errFailedUpdateProject(err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound(errProjectNotFound)
	}

	return nil
}

func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM projects WHERE id = $1"
	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return errFailedDeleteProject(err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound(errProjectNotFound)
	}

	return nil
}

func (r *ProjectRepository) AddMember(ctx context.Context, input project.AddMemberInput) (*project.Member, error) {
	query := `
		INSERT INTO project_members (project_id, user_id, role, invited_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, project_id, user_id, role, invited_by, invited_at
	`

	m := &project.Member{}
	err := r.db.Pool.QueryRow(ctx, query, input.ProjectID, input.UserID, input.Role, input.InvitedBy).Scan(
		&m.ID, &m.ProjectID, &m.UserID, &m.Role, &m.InvitedBy, &m.InvitedAt,
	)

	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.Conflict("user is already a member of this project")
		}
		return nil, errFailedAddMember(err)
	}

	return m, nil
}

func (r *ProjectRepository) GetMember(ctx context.Context, projectID, userID uuid.UUID) (*project.Member, error) {
	query := `
		SELECT id, project_id, user_id, role, invited_by, invited_at
		FROM project_members WHERE project_id = $1 AND user_id = $2
	`

	m := &project.Member{}
	err := r.db.Pool.QueryRow(ctx, query, projectID, userID).Scan(
		&m.ID, &m.ProjectID, &m.UserID, &m.Role, &m.InvitedBy, &m.InvitedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound(errMemberNotFound)
		}
		return nil, errFailedGetMember(err)
	}

	return m, nil
}

func (r *ProjectRepository) GetMembers(ctx context.Context, projectID uuid.UUID) ([]*project.Member, error) {
	query := `
		SELECT id, project_id, user_id, role, invited_by, invited_at
		FROM project_members WHERE project_id = $1 ORDER BY invited_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, errFailedListMembers(err)
	}
	defer rows.Close()

	var members []*project.Member
	for rows.Next() {
		m := &project.Member{}
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.UserID, &m.Role, &m.InvitedBy, &m.InvitedAt); err != nil {
			return nil, errFailedScanMember(err)
		}
		members = append(members, m)
	}

	return members, rows.Err()
}

func (r *ProjectRepository) CountAdminsByProject(ctx context.Context, projectID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM project_members WHERE project_id = $1 AND role = 'admin'`

	var count int
	err := r.db.Pool.QueryRow(ctx, query, projectID).Scan(&count)
	if err != nil {
		return 0, errFailedCountAdmins(err)
	}

	return count, nil
}

func (r *ProjectRepository) UpdateMemberRole(ctx context.Context, input project.UpdateMemberRoleInput) error {
	query := "UPDATE project_members SET role = $1 WHERE project_id = $2 AND user_id = $3"
	result, err := r.db.Pool.Exec(ctx, query, input.Role, input.ProjectID, input.UserID)
	if err != nil {
		return errFailedUpdateMemberRole(err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound(errMemberNotFound)
	}

	return nil
}

func (r *ProjectRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	query := "DELETE FROM project_members WHERE project_id = $1 AND user_id = $2"
	result, err := r.db.Pool.Exec(ctx, query, projectID, userID)
	if err != nil {
		return errFailedRemoveMember(err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound(errMemberNotFound)
	}

	return nil
}

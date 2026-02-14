package postgres

import (
	"context"
	"file-service/internal/domain/file"
	apperrors "file-service/pkg/errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type FileRepository struct {
	db *DB
}

func NewFileRepository(db *DB) *FileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) Create(ctx context.Context, input file.CreateFileInput) (*file.File, error) {
	query := `
		INSERT INTO files (project_id, folder_id, name, s3_key, size_bytes, mime_type, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, project_id, folder_id, name, s3_key, size_bytes, mime_type, uploaded_by, created_at, updated_at
	`

	f := &file.File{}
	err := r.db.Pool.QueryRow(ctx, query, input.ProjectID, input.FolderID, input.Name, input.S3Key, input.SizeBytes, input.MimeType, input.UploadedBy).Scan(
		&f.ID, &f.ProjectID, &f.FolderID, &f.Name, &f.S3Key, &f.SizeBytes, &f.MimeType, &f.UploadedBy, &f.CreatedAt, &f.UpdatedAt,
	)

	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.Conflict("file already exists at this path")
		}
		return nil, errFailedCreateFile(err)
	}

	return f, nil
}

func (r *FileRepository) GetByID(ctx context.Context, id uuid.UUID) (*file.File, error) {
	query := `
		SELECT id, project_id, folder_id, name, s3_key, size_bytes, mime_type, uploaded_by, created_at, updated_at
		FROM files WHERE id = $1
	`

	f := &file.File{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&f.ID, &f.ProjectID, &f.FolderID, &f.Name, &f.S3Key, &f.SizeBytes, &f.MimeType, &f.UploadedBy, &f.CreatedAt, &f.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound(errFileNotFound)
		}
		return nil, errFailedGetFile(err)
	}

	return f, nil
}

func (r *FileRepository) GetByProjectAndS3Key(ctx context.Context, projectID uuid.UUID, s3Key string) (*file.File, error) {
	query := `
		SELECT id, project_id, folder_id, name, s3_key, size_bytes, mime_type, uploaded_by, created_at, updated_at
		FROM files WHERE project_id = $1 AND s3_key = $2
	`

	f := &file.File{}
	err := r.db.Pool.QueryRow(ctx, query, projectID, s3Key).Scan(
		&f.ID, &f.ProjectID, &f.FolderID, &f.Name, &f.S3Key, &f.SizeBytes, &f.MimeType, &f.UploadedBy, &f.CreatedAt, &f.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound(errFileNotFound)
		}
		return nil, errFailedGetFile(err)
	}

	return f, nil
}

func (r *FileRepository) List(ctx context.Context, filter file.ListFilesFilter) ([]*file.File, error) {
	query := `
		SELECT id, project_id, folder_id, name, s3_key, size_bytes, mime_type, uploaded_by, created_at, updated_at
		FROM files WHERE project_id = $1
	`
	args := []interface{}{filter.ProjectID}

	if filter.FolderID != nil {
		query += " AND folder_id = $2"
		args = append(args, *filter.FolderID)
	}

	query += " ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", len(args)+1) + " OFFSET $" + fmt.Sprintf("%d", len(args)+2)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, errFailedListFiles(err)
	}
	defer rows.Close()

	var files []*file.File
	for rows.Next() {
		f := &file.File{}
		if err := rows.Scan(&f.ID, &f.ProjectID, &f.FolderID, &f.Name, &f.S3Key, &f.SizeBytes, &f.MimeType, &f.UploadedBy, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, errFailedScanFile(err)
		}
		files = append(files, f)
	}

	return files, rows.Err()
}

func (r *FileRepository) DeleteByProjectAndPrefix(ctx context.Context, projectID uuid.UUID, prefix string) (int64, error) {
	basePrefix := strings.TrimRight(strings.TrimSpace(prefix), "/")
	if basePrefix == "" {
		return 0, fmt.Errorf(errPrefixEmpty)
	}

	query := `
		DELETE FROM files
		WHERE project_id = $1
		  AND (s3_key = $2 OR s3_key LIKE $3)
	`

	likePattern := escapeLikePattern(basePrefix) + "/%"
	result, err := r.db.Pool.Exec(ctx, query, projectID, basePrefix, likePattern)
	if err != nil {
		return 0, errFailedDeleteFilesByPrefix(err)
	}

	return result.RowsAffected(), nil
}

func (r *FileRepository) CountByProjectAndPrefix(ctx context.Context, projectID uuid.UUID, prefix string) (int64, error) {
	basePrefix := strings.TrimRight(strings.TrimSpace(prefix), "/")
	if basePrefix == "" {
		return 0, fmt.Errorf(errPrefixEmpty)
	}

	query := `
		SELECT COUNT(*)
		FROM files
		WHERE project_id = $1
		  AND (s3_key = $2 OR s3_key LIKE $3)
	`

	likePattern := escapeLikePattern(basePrefix) + "/%"
	var count int64
	if err := r.db.Pool.QueryRow(ctx, query, projectID, basePrefix, likePattern).Scan(&count); err != nil {
		return 0, errFailedCountFilesByPrefix(err)
	}

	return count, nil
}

func (r *FileRepository) Update(ctx context.Context, id uuid.UUID, input file.UpdateFileInput) error {
	query := "UPDATE files SET updated_at = NOW()"
	args := []interface{}{id}
	argCount := 1

	if input.Name != nil {
		argCount++
		query += fmt.Sprintf(", name = $%d", argCount)
		args = append(args, *input.Name)
	}

	if input.SizeBytes != nil {
		argCount++
		query += fmt.Sprintf(", size_bytes = $%d", argCount)
		args = append(args, *input.SizeBytes)
	}

	if input.MimeType != nil {
		argCount++
		query += fmt.Sprintf(", mime_type = $%d", argCount)
		args = append(args, *input.MimeType)
	}

	query += " WHERE id = $1"

	result, err := r.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return errFailedUpdateFile(err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound(errFileNotFound)
	}

	return nil
}

func (r *FileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM files WHERE id = $1"
	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return errFailedDeleteFile(err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound(errFileNotFound)
	}

	return nil
}

func (r *FileRepository) CreateFolder(ctx context.Context, input file.CreateFolderInput) (*file.Folder, error) {
	query := `
		INSERT INTO folders (project_id, parent_folder_id, name, s3_prefix, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, project_id, parent_folder_id, name, s3_prefix, created_by, created_at
	`

	folder := &file.Folder{}
	err := r.db.Pool.QueryRow(ctx, query, input.ProjectID, input.ParentFolderID, input.Name, input.S3Prefix, input.CreatedBy).Scan(
		&folder.ID, &folder.ProjectID, &folder.ParentFolderID, &folder.Name, &folder.S3Prefix, &folder.CreatedBy, &folder.CreatedAt,
	)

	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.Conflict("folder already exists at this path")
		}
		return nil, errFailedCreateFolder(err)
	}

	return folder, nil
}

func (r *FileRepository) GetFolder(ctx context.Context, id uuid.UUID) (*file.Folder, error) {
	query := `
		SELECT id, project_id, parent_folder_id, name, s3_prefix, created_by, created_at
		FROM folders WHERE id = $1
	`

	folder := &file.Folder{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&folder.ID, &folder.ProjectID, &folder.ParentFolderID, &folder.Name, &folder.S3Prefix, &folder.CreatedBy, &folder.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound(errFolderNotFound)
		}
		return nil, errFailedGetFolder(err)
	}

	return folder, nil
}

func (r *FileRepository) GetFolderByPath(ctx context.Context, projectID uuid.UUID, s3Prefix string) (*file.Folder, error) {
	query := `
		SELECT id, project_id, parent_folder_id, name, s3_prefix, created_by, created_at
		FROM folders WHERE project_id = $1 AND s3_prefix = $2
	`

	folder := &file.Folder{}
	err := r.db.Pool.QueryRow(ctx, query, projectID, s3Prefix).Scan(
		&folder.ID, &folder.ProjectID, &folder.ParentFolderID, &folder.Name, &folder.S3Prefix, &folder.CreatedBy, &folder.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound(errFolderNotFound)
		}
		return nil, errFailedGetFolder(err)
	}

	return folder, nil
}

func (r *FileRepository) ListFolders(ctx context.Context, projectID uuid.UUID, parentFolderID *uuid.UUID) ([]*file.Folder, error) {
	query := `
		SELECT id, project_id, parent_folder_id, name, s3_prefix, created_by, created_at
		FROM folders WHERE project_id = $1
	`
	args := []interface{}{projectID}

	if parentFolderID != nil {
		query += " AND parent_folder_id = $2"
		args = append(args, *parentFolderID)
	} else {
		query += " AND parent_folder_id IS NULL"
	}

	query += " ORDER BY name ASC"

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, errFailedListFolders(err)
	}
	defer rows.Close()

	var folders []*file.Folder
	for rows.Next() {
		folder := &file.Folder{}
		if err := rows.Scan(&folder.ID, &folder.ProjectID, &folder.ParentFolderID, &folder.Name, &folder.S3Prefix, &folder.CreatedBy, &folder.CreatedAt); err != nil {
			return nil, errFailedScanFolder(err)
		}
		folders = append(folders, folder)
	}

	return folders, rows.Err()
}

func (r *FileRepository) DeleteFolder(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM folders WHERE id = $1"
	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return errFailedDeleteFolder(err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.NotFound(errFolderNotFound)
	}

	return nil
}

package postgres

import (
	"context"
	"file-service/internal/domain/share"
	apperrors "file-service/pkg/errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ShareLinkRepository struct {
	db *DB
}

func NewShareLinkRepository(db *DB) *ShareLinkRepository {
	return &ShareLinkRepository{db: db}
}

func (r *ShareLinkRepository) Create(ctx context.Context, input share.CreateShareLinkInput) (*share.ShareLink, error) {
	query := `
		INSERT INTO share_links (file_id, token, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, file_id, token, created_by, created_at
	`

	link := &share.ShareLink{}
	err := r.db.Pool.QueryRow(ctx, query, input.FileID, input.Token, input.CreatedBy).Scan(
		&link.ID,
		&link.FileID,
		&link.Token,
		&link.CreatedBy,
		&link.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.Conflict("share link already exists")
		}
		return nil, errFailedCreateShareLink(err)
	}

	return link, nil
}

func (r *ShareLinkRepository) GetByID(ctx context.Context, id uuid.UUID) (*share.ShareLink, error) {
	query := `
		SELECT id, file_id, token, created_by, created_at
		FROM share_links WHERE id = $1
	`

	link := &share.ShareLink{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&link.ID,
		&link.FileID,
		&link.Token,
		&link.CreatedBy,
		&link.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound(errShareLinkNotFound)
		}
		return nil, errFailedGetShareLink(err)
	}

	return link, nil
}

func (r *ShareLinkRepository) GetByToken(ctx context.Context, token string) (*share.ShareLink, error) {
	query := `
		SELECT id, file_id, token, created_by, created_at
		FROM share_links WHERE token = $1
	`

	link := &share.ShareLink{}
	err := r.db.Pool.QueryRow(ctx, query, token).Scan(
		&link.ID,
		&link.FileID,
		&link.Token,
		&link.CreatedBy,
		&link.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NotFound(errShareLinkNotFound)
		}
		return nil, errFailedGetShareLinkByToken(err)
	}

	return link, nil
}

func (r *ShareLinkRepository) ListByFileID(ctx context.Context, fileID uuid.UUID) ([]*share.ShareLink, error) {
	query := `
		SELECT id, file_id, token, created_by, created_at
		FROM share_links WHERE file_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, fileID)
	if err != nil {
		return nil, errFailedListShareLinks(err)
	}
	defer rows.Close()

	links := make([]*share.ShareLink, 0)
	for rows.Next() {
		link := &share.ShareLink{}
		if err := rows.Scan(&link.ID, &link.FileID, &link.Token, &link.CreatedBy, &link.CreatedAt); err != nil {
			return nil, errFailedScanShareLink(err)
		}
		links = append(links, link)
	}

	return links, rows.Err()
}

func (r *ShareLinkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM share_links WHERE id = $1"
	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return errFailedDeleteShareLink(err)
	}
	if result.RowsAffected() == 0 {
		return apperrors.NotFound(errShareLinkNotFound)
	}

	return nil
}

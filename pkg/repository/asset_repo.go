package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type AssetRepository struct {
	db *sql.DB
}

func NewAssetRepository(db *sql.DB) *AssetRepository {
	return &AssetRepository{db: db}
}

type Asset struct {
	ID               string    `json:"id"`
	ClientID         string    `json:"client_id"`
	ProjectID        string    `json:"project_id"`
	FolderPath       string    `json:"folder_path"`
	Filename         string    `json:"filename"`
	OriginalFilename string    `json:"original_filename"`
	FileSize         int64     `json:"file_size"`
	MimeType         string    `json:"mime_type,omitempty"`
	S3Key            string    `json:"s3_key"`
	PresignedURL     string    `json:"presigned_url,omitempty"`
	Version          int       `json:"version"`
	IsLatest         bool      `json:"is_latest"`
	ParentAssetID    *string   `json:"parent_asset_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (r *AssetRepository) CreateAsset(clientID, projectID, folderPath, filename, originalFilename string, fileSize int64, mimeType, s3Key string) (*Asset, error) {
	assetID := uuid.New().String()

	if folderPath == "" {
		folderPath = "/"
	}

	query := `
		INSERT INTO assets (id, client_id, project_id, folder_path, filename, original_filename, file_size, mime_type, s3_key, version, is_latest)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 1, TRUE)
		RETURNING id, client_id, project_id, folder_path, filename, original_filename, file_size, mime_type, s3_key, version, is_latest, parent_asset_id, created_at, updated_at
	`

	var asset Asset
	var parentID sql.NullString

	err := r.db.QueryRow(query, assetID, clientID, projectID, folderPath, filename, originalFilename, fileSize, mimeType, s3Key).Scan(
		&asset.ID,
		&asset.ClientID,
		&asset.ProjectID,
		&asset.FolderPath,
		&asset.Filename,
		&asset.OriginalFilename,
		&asset.FileSize,
		&asset.MimeType,
		&asset.S3Key,
		&asset.Version,
		&asset.IsLatest,
		&parentID,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create asset: %w", err)
	}

	if parentID.Valid {
		asset.ParentAssetID = &parentID.String
	}

	return &asset, nil
}

func (r *AssetRepository) CreateAssetVersion(clientID, projectID, folderPath, filename, originalFilename string, fileSize int64, mimeType, s3Key, parentAssetID string) (*Asset, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var parentVersion int
	err = tx.QueryRow(`SELECT version FROM assets WHERE id = $1 AND client_id = $2`, parentAssetID, clientID).Scan(&parentVersion)
	if err != nil {
		return nil, fmt.Errorf("parent asset not found: %w", err)
	}

	_, err = tx.Exec(`UPDATE assets SET is_latest = FALSE WHERE (id = $1 OR parent_asset_id = $1) AND client_id = $2`, parentAssetID, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to update previous versions: %w", err)
	}

	assetID := uuid.New().String()
	nextVersion := parentVersion + 1

	query := `
		INSERT INTO assets (id, client_id, project_id, folder_path, filename, original_filename, file_size, mime_type, s3_key, version, is_latest, parent_asset_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, TRUE, $11)
		RETURNING id, client_id, project_id, folder_path, filename, original_filename, file_size, mime_type, s3_key, version, is_latest, parent_asset_id, created_at, updated_at
	`

	var asset Asset
	var parentID sql.NullString

	err = tx.QueryRow(query, assetID, clientID, projectID, folderPath, filename, originalFilename, fileSize, mimeType, s3Key, nextVersion, parentAssetID).Scan(
		&asset.ID,
		&asset.ClientID,
		&asset.ProjectID,
		&asset.FolderPath,
		&asset.Filename,
		&asset.OriginalFilename,
		&asset.FileSize,
		&asset.MimeType,
		&asset.S3Key,
		&asset.Version,
		&asset.IsLatest,
		&parentID,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create asset version: %w", err)
	}

	if parentID.Valid {
		asset.ParentAssetID = &parentID.String
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &asset, nil
}

func (r *AssetRepository) GetAssetsByProjectID(projectID, clientID string, folderPath *string) ([]Asset, error) {
	var query string
	var args []any

	if folderPath != nil && *folderPath != "" {
		query = `
			SELECT id, client_id, project_id, folder_path, filename, original_filename, file_size, mime_type, s3_key, version, is_latest, parent_asset_id, created_at, updated_at
			FROM assets
			WHERE project_id = $1 AND client_id = $2 AND folder_path = $3 AND is_latest = TRUE
			ORDER BY created_at DESC
		`
		args = []any{projectID, clientID, *folderPath}
	} else {
		query = `
			SELECT id, client_id, project_id, folder_path, filename, original_filename, file_size, mime_type, s3_key, version, is_latest, parent_asset_id, created_at, updated_at
			FROM assets
			WHERE project_id = $1 AND client_id = $2 AND is_latest = TRUE
			ORDER BY created_at DESC
		`
		args = []any{projectID, clientID}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get assets: %w", err)
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var asset Asset
		var parentID sql.NullString

		err := rows.Scan(
			&asset.ID,
			&asset.ClientID,
			&asset.ProjectID,
			&asset.FolderPath,
			&asset.Filename,
			&asset.OriginalFilename,
			&asset.FileSize,
			&asset.MimeType,
			&asset.S3Key,
			&asset.Version,
			&asset.IsLatest,
			&parentID,
			&asset.CreatedAt,
			&asset.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan asset: %w", err)
		}

		if parentID.Valid {
			asset.ParentAssetID = &parentID.String
		}

		assets = append(assets, asset)
	}

	return assets, nil
}

func (r *AssetRepository) GetAssetByID(assetID, clientID string) (*Asset, error) {
	query := `
		SELECT id, client_id, project_id, folder_path, filename, original_filename, file_size, mime_type, s3_key, version, is_latest, parent_asset_id, created_at, updated_at
		FROM assets
		WHERE id = $1 AND client_id = $2
	`

	var asset Asset
	var parentID sql.NullString

	err := r.db.QueryRow(query, assetID, clientID).Scan(
		&asset.ID,
		&asset.ClientID,
		&asset.ProjectID,
		&asset.FolderPath,
		&asset.Filename,
		&asset.OriginalFilename,
		&asset.FileSize,
		&asset.MimeType,
		&asset.S3Key,
		&asset.Version,
		&asset.IsLatest,
		&parentID,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("asset not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	if parentID.Valid {
		asset.ParentAssetID = &parentID.String
	}

	return &asset, nil
}

func (r *AssetRepository) DeleteAsset(assetID, clientID string) error {
	query := `DELETE FROM assets WHERE id = $1 AND client_id = $2`
	result, err := r.db.Exec(query, assetID, clientID)
	if err != nil {
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("asset not found")
	}

	return nil
}

func (r *AssetRepository) GetFoldersByProjectID(projectID, clientID string) ([]string, error) {
	query := `
		SELECT DISTINCT folder_path
		FROM assets
		WHERE project_id = $1 AND client_id = $2
		ORDER BY folder_path
	`

	rows, err := r.db.Query(query, projectID, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get folders: %w", err)
	}
	defer rows.Close()

	var folders []string
	for rows.Next() {
		var folder string
		if err := rows.Scan(&folder); err != nil {
			return nil, fmt.Errorf("failed to scan folder: %w", err)
		}
		folders = append(folders, folder)
	}

	return folders, nil
}

func (r *AssetRepository) GetAssetVersions(assetID, clientID string) ([]Asset, error) {
	asset, err := r.GetAssetByID(assetID, clientID)
	if err != nil {
		return nil, err
	}

	rootAssetID := assetID
	if asset.ParentAssetID != nil {
		rootAssetID = *asset.ParentAssetID
	}

	query := `
		SELECT id, client_id, project_id, folder_path, filename, original_filename, file_size, mime_type, s3_key, version, is_latest, parent_asset_id, created_at, updated_at
		FROM assets
		WHERE (id = $1 OR parent_asset_id = $1) AND client_id = $2
		ORDER BY version DESC
	`

	rows, err := r.db.Query(query, rootAssetID, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset versions: %w", err)
	}
	defer rows.Close()

	var versions []Asset
	for rows.Next() {
		var version Asset
		var parentID sql.NullString

		err := rows.Scan(
			&version.ID,
			&version.ClientID,
			&version.ProjectID,
			&version.FolderPath,
			&version.Filename,
			&version.OriginalFilename,
			&version.FileSize,
			&version.MimeType,
			&version.S3Key,
			&version.Version,
			&version.IsLatest,
			&parentID,
			&version.CreatedAt,
			&version.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}

		if parentID.Valid {
			version.ParentAssetID = &parentID.String
		}

		versions = append(versions, version)
	}

	return versions, nil
}

func (r *AssetRepository) GetAllS3KeysByClientID(clientID string) ([]string, error) {
	query := `
		SELECT DISTINCT s3_key
		FROM assets
		WHERE client_id = $1
	`

	rows, err := r.db.Query(query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get s3 keys by client: %w", err)
	}
	defer rows.Close()

	keys := make([]string, 0)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan s3 key: %w", err)
		}
		keys = append(keys, key)
	}

	return keys, nil
}

package repository

import (
	"database/sql"
	"file-service/pkg/models"
	"fmt"
)

type ProjectRepository struct {
	db *sql.DB
}

func NewProjectRepository(db *sql.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// CreateProject creates a new project
func (r *ProjectRepository) CreateProject(clientID, name, description string) (*models.Project, error) {
	query := `
		INSERT INTO projects (client_id, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, client_id, name, description, created_at, updated_at
	`

	var project models.Project
	err := r.db.QueryRow(query, clientID, name, description).Scan(
		&project.ID,
		&project.ClientID,
		&project.Name,
		&project.Description,
		&project.CreatedAt,
		&project.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return &project, nil
}

// GetProjectsByClientID retrieves all projects for a client
func (r *ProjectRepository) GetProjectsByClientID(clientID string) ([]models.Project, error) {
	query := `
		SELECT id, client_id, name, description, created_at, updated_at
		FROM projects
		WHERE client_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var project models.Project
		err := rows.Scan(
			&project.ID,
			&project.ClientID,
			&project.Name,
			&project.Description,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, project)
	}

	return projects, nil
}

// GetProjectByID retrieves a project by ID
func (r *ProjectRepository) GetProjectByID(projectID, clientID string) (*models.Project, error) {
	query := `
		SELECT id, client_id, name, description, created_at, updated_at
		FROM projects
		WHERE id = $1 AND client_id = $2
	`

	var project models.Project
	err := r.db.QueryRow(query, projectID, clientID).Scan(
		&project.ID,
		&project.ClientID,
		&project.Name,
		&project.Description,
		&project.CreatedAt,
		&project.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &project, nil
}

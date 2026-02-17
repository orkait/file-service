package repository

import (
	"database/sql"
	"file-service/pkg/models"
	"fmt"
)

type MemberRepository struct {
	db *sql.DB
}

func NewMemberRepository(db *sql.DB) *MemberRepository {
	return &MemberRepository{db: db}
}

// InviteMember adds a member to a project
func (r *MemberRepository) InviteMember(projectID, clientID, invitedBy, role string) (*models.ProjectMember, error) {
	query := `
		INSERT INTO project_members (project_id, client_id, role, invited_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, project_id, client_id, role, invited_by, created_at
	`

	var member models.ProjectMember
	var invitedByVal sql.NullString

	err := r.db.QueryRow(query, projectID, clientID, role, invitedBy).Scan(
		&member.ID,
		&member.ProjectID,
		&member.ClientID,
		&member.Role,
		&invitedByVal,
		&member.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to invite member: %w", err)
	}

	if invitedByVal.Valid {
		member.InvitedBy = &invitedByVal.String
	}

	return &member, nil
}

// GetProjectMembers retrieves all members of a project
func (r *MemberRepository) GetProjectMembers(projectID string) ([]models.ProjectMember, error) {
	query := `
		SELECT id, project_id, client_id, role, invited_by, created_at
		FROM project_members
		WHERE project_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}
	defer rows.Close()

	var members []models.ProjectMember
	for rows.Next() {
		var member models.ProjectMember
		var invitedByVal sql.NullString

		err := rows.Scan(
			&member.ID,
			&member.ProjectID,
			&member.ClientID,
			&member.Role,
			&invitedByVal,
			&member.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}

		if invitedByVal.Valid {
			member.InvitedBy = &invitedByVal.String
		}

		members = append(members, member)
	}

	return members, nil
}

// CheckMemberAccess checks if a client has access to a project
func (r *MemberRepository) CheckMemberAccess(projectID, clientID string) (bool, string, error) {
	query := `
		SELECT role FROM project_members
		WHERE project_id = $1 AND client_id = $2
	`

	var role string
	err := r.db.QueryRow(query, projectID, clientID).Scan(&role)

	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("failed to check access: %w", err)
	}

	return true, role, nil
}

// RemoveMember removes a member from a project
func (r *MemberRepository) RemoveMember(projectID, clientID string) error {
	query := `DELETE FROM project_members WHERE project_id = $1 AND client_id = $2 AND role != 'owner'`
	result, err := r.db.Exec(query, projectID, clientID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("member not found or cannot remove owner")
	}

	return nil
}

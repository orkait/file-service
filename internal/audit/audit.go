package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// ActorType represents the type of entity performing an action
type ActorType string

const (
	ActorTypeUser   ActorType = "user"
	ActorTypeAPIKey ActorType = "api_key"
	ActorTypeSystem ActorType = "system"
)

// ResourceType represents the type of resource being acted upon
type ResourceType string

const (
	ResourceTypeProject   ResourceType = "project"
	ResourceTypeFile      ResourceType = "file"
	ResourceTypeFolder    ResourceType = "folder"
	ResourceTypeAPIKey    ResourceType = "api_key"
	ResourceTypeShareLink ResourceType = "share_link"
	ResourceTypeUser      ResourceType = "user"
)

// Action represents the action being performed
type Action string

const (
	ActionCreate Action = "create"
	ActionRead   Action = "read"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionLogin  Action = "login"
	ActionLogout Action = "logout"
	ActionRevoke Action = "revoke"
	ActionShare  Action = "share"
)

// Status represents the outcome of an action
type Status string

const (
	StatusSuccess Status = "success"
	StatusFailure Status = "failure"
	StatusDenied  Status = "denied"
)

// Event represents an audit event
type Event struct {
	ID           uuid.UUID
	EventType    string
	ActorType    ActorType
	ActorID      *uuid.UUID
	ResourceType ResourceType
	ResourceID   *uuid.UUID
	Action       Action
	Status       Status
	IPAddress    string
	UserAgent    string
	RequestID    string
	Metadata     map[string]any
	ErrorMessage string
	CreatedAt    time.Time
}

// Logger handles audit logging
type Logger struct {
	pool *pgxpool.Pool
}

// NewLogger creates a new audit logger
func NewLogger(pool *pgxpool.Pool) *Logger {
	return &Logger{pool: pool}
}

// Log records an audit event
func (l *Logger) Log(ctx context.Context, event *Event) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	var metadataJSON []byte
	var err error
	if event.Metadata != nil {
		metadataJSON, err = json.Marshal(event.Metadata)
		if err != nil {
			return err
		}
	}

	query := `
		INSERT INTO audit_events (
			id, event_type, actor_type, actor_id, resource_type, resource_id,
			action, status, ip_address, user_agent, request_id, metadata, error_message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err = l.pool.Exec(ctx, query,
		event.ID,
		event.EventType,
		event.ActorType,
		event.ActorID,
		event.ResourceType,
		event.ResourceID,
		event.Action,
		event.Status,
		event.IPAddress,
		event.UserAgent,
		event.RequestID,
		metadataJSON,
		event.ErrorMessage,
		event.CreatedAt,
	)

	return err
}

// LogFromContext creates and logs an audit event from an Echo context asynchronously
func (l *Logger) LogFromContext(c echo.Context, resourceType ResourceType, resourceID *uuid.UUID, action Action, status Status, metadata map[string]any) error {
	event := &Event{
		EventType:    string(action) + "_" + string(resourceType),
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
		Status:       status,
		IPAddress:    c.RealIP(),
		UserAgent:    c.Request().UserAgent(),
		RequestID:    c.Response().Header().Get(echo.HeaderXRequestID),
		Metadata:     metadata,
	}

	// Extract actor information from context
	if userID := c.Get("user_id"); userID != nil {
		if uid, ok := userID.(uuid.UUID); ok {
			event.ActorType = ActorTypeUser
			event.ActorID = &uid
		}
	} else if apiKeyID := c.Get("api_key_id"); apiKeyID != nil {
		if kid, ok := apiKeyID.(uuid.UUID); ok {
			event.ActorType = ActorTypeAPIKey
			event.ActorID = &kid
		}
	} else {
		event.ActorType = ActorTypeSystem
	}

	// Log asynchronously with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	go func() {
		defer cancel()
		if err := l.Log(ctx, event); err != nil {
			// Log to stderr but don't block the request
			fmt.Fprintf(c.Logger().Output(), "audit log failed: %v\n", err)
		}
	}()

	return nil
}

// LogError logs a failed action with error details asynchronously
func (l *Logger) LogError(c echo.Context, resourceType ResourceType, resourceID *uuid.UUID, action Action, err error) error {
	metadata := map[string]any{
		"error": err.Error(),
	}

	event := &Event{
		EventType:    string(action) + "_" + string(resourceType),
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
		Status:       StatusFailure,
		IPAddress:    c.RealIP(),
		UserAgent:    c.Request().UserAgent(),
		RequestID:    c.Response().Header().Get(echo.HeaderXRequestID),
		Metadata:     metadata,
		ErrorMessage: err.Error(),
	}

	// Extract actor information
	if userID := c.Get("user_id"); userID != nil {
		if uid, ok := userID.(uuid.UUID); ok {
			event.ActorType = ActorTypeUser
			event.ActorID = &uid
		}
	} else if apiKeyID := c.Get("api_key_id"); apiKeyID != nil {
		if kid, ok := apiKeyID.(uuid.UUID); ok {
			event.ActorType = ActorTypeAPIKey
			event.ActorID = &kid
		}
	} else {
		event.ActorType = ActorTypeSystem
	}

	// Log asynchronously with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	go func() {
		defer cancel()
		if err := l.Log(ctx, event); err != nil {
			// Log to stderr but don't block the request
			fmt.Fprintf(c.Logger().Output(), "audit log failed: %v\n", err)
		}
	}()

	return nil
}

// Query retrieves audit events based on filters
type QueryFilter struct {
	ActorID      *uuid.UUID
	ResourceType *ResourceType
	ResourceID   *uuid.UUID
	Action       *Action
	Status       *Status
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	Offset       int
}

// Query retrieves audit events
func (l *Logger) Query(ctx context.Context, filter QueryFilter) ([]*Event, error) {
	query := `
		SELECT id, event_type, actor_type, actor_id, resource_type, resource_id,
		       action, status, ip_address, user_agent, request_id, metadata, error_message, created_at
		FROM audit_events
		WHERE 1=1
	`
	args := []any{}
	argCount := 1

	if filter.ActorID != nil {
		query += fmt.Sprintf(" AND actor_id = $%d", argCount)
		args = append(args, filter.ActorID)
		argCount++
	}

	if filter.ResourceType != nil {
		query += fmt.Sprintf(" AND resource_type = $%d", argCount)
		args = append(args, filter.ResourceType)
		argCount++
	}

	if filter.ResourceID != nil {
		query += fmt.Sprintf(" AND resource_id = $%d", argCount)
		args = append(args, filter.ResourceID)
		argCount++
	}

	if filter.Action != nil {
		query += fmt.Sprintf(" AND action = $%d", argCount)
		args = append(args, filter.Action)
		argCount++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, filter.StartTime)
		argCount++
	}

	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, filter.EndTime)
		argCount++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	} else {
		query += " LIMIT 100" // Default limit
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := l.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []*Event{}
	for rows.Next() {
		event := &Event{}
		var metadataJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.ActorType,
			&event.ActorID,
			&event.ResourceType,
			&event.ResourceID,
			&event.Action,
			&event.Status,
			&event.IPAddress,
			&event.UserAgent,
			&event.RequestID,
			&metadataJSON,
			&event.ErrorMessage,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
				return nil, err
			}
		}

		events = append(events, event)
	}

	return events, rows.Err()
}

package share

import (
	"time"

	"github.com/google/uuid"
)

type ShareLink struct {
	ID        uuid.UUID
	FileID    uuid.UUID
	Token     string
	CreatedBy uuid.UUID
	CreatedAt time.Time
}

type CreateShareLinkInput struct {
	FileID    uuid.UUID
	Token     string
	CreatedBy uuid.UUID
}

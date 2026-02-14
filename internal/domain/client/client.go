package client

import (
	"time"

	"github.com/google/uuid"
)

type Client struct {
	ID          uuid.UUID
	OwnerUserID uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateClientInput struct {
	OwnerUserID uuid.UUID
}

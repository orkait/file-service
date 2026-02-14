package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	ClientID     uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateUserInput struct {
	Email    string
	Password string
	ClientID uuid.UUID
}

type UpdateUserInput struct {
	Email        *string
	PasswordHash *string
}

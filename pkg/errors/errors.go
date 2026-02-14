package errors

import (
	"errors"
	"fmt"
)

// Domain errors - Sentinel errors for use with errors.Is()
var (
	ErrNotFound            = errors.New("resource not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrBadRequest          = errors.New("bad request")
	ErrConflict            = errors.New("resource already exists")
	ErrInternalServer      = errors.New("internal server error")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrExpired             = errors.New("resource expired")
	ErrRevoked             = errors.New("resource revoked")
	ErrEmailExists         = errors.New("email already exists")
	ErrValidation          = errors.New("validation error")
	ErrInvalidInput        = errors.New("invalid input")
	ErrAPIKeyRevoked       = errors.New("api key revoked")
	ErrAPIKeyExpired       = errors.New("api key expired")
	ErrInsufficientPerms   = errors.New("insufficient permissions")
	ErrPathTraversal       = errors.New("path traversal attempt detected")
	ErrFolderDepthExceeded = errors.New("folder depth limit exceeded")
)

// Custom error type with context
type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Constructors
func NotFound(msg string) *AppError {
	return &AppError{Code: "NOT_FOUND", Message: msg, Err: ErrNotFound}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Code: "UNAUTHORIZED", Message: msg, Err: ErrUnauthorized}
}

func Forbidden(msg string) *AppError {
	return &AppError{Code: "FORBIDDEN", Message: msg, Err: ErrForbidden}
}

func BadRequest(msg string) *AppError {
	return &AppError{Code: "BAD_REQUEST", Message: msg, Err: ErrBadRequest}
}

func Conflict(msg string) *AppError {
	return &AppError{Code: "CONFLICT", Message: msg, Err: ErrConflict}
}

func InternalServer(msg string, err error) *AppError {
	return &AppError{Code: "INTERNAL_SERVER_ERROR", Message: msg, Err: err}
}

func InvalidCredentials() *AppError {
	return &AppError{Code: "INVALID_CREDENTIALS", Message: "invalid email or password", Err: ErrInvalidCredentials}
}

func Expired(msg string) *AppError {
	return &AppError{Code: "EXPIRED", Message: msg, Err: ErrExpired}
}

func Revoked(msg string) *AppError {
	return &AppError{Code: "REVOKED", Message: msg, Err: ErrRevoked}
}

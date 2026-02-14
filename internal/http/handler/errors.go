package handler

import (
	"errors"
	apperrors "file-service/pkg/errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// MapToPublicError maps internal errors to public-facing HTTP status codes and messages
// This prevents information disclosure by providing consistent, generic error messages
func MapToPublicError(err error) (int, string) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		return http.StatusNotFound, "resource not found"
	case errors.Is(err, apperrors.ErrUnauthorized):
		return http.StatusUnauthorized, "authentication required"
	case errors.Is(err, apperrors.ErrForbidden):
		return http.StatusForbidden, "access denied"
	case errors.Is(err, apperrors.ErrConflict):
		return http.StatusConflict, "resource conflict"
	case errors.Is(err, apperrors.ErrValidation):
		return http.StatusBadRequest, "invalid input"
	case errors.Is(err, apperrors.ErrRevoked):
		return http.StatusUnauthorized, "credentials revoked"
	case errors.Is(err, apperrors.ErrExpired):
		return http.StatusUnauthorized, "credentials expired"
	default:
		// Never expose internal errors to clients
		return http.StatusInternalServerError, "internal server error"
	}
}

// RespondWithMappedError responds with a mapped error, preventing information disclosure
func RespondWithMappedError(c echo.Context, err error) error {
	status, msg := MapToPublicError(err)
	return respondError(c, status, msg)
}

// SafeErrorResponse provides a safe error response that doesn't leak information
// Use this when you want to return "not found" for both missing resources and authorization failures
func SafeErrorResponse(c echo.Context, err error, safeStatus int, safeMessage string) error {
	// Log the actual error for debugging
	c.Logger().Errorf("Error (masked as %d): %v", safeStatus, err)

	// Return safe error to client
	return respondError(c, safeStatus, safeMessage)
}

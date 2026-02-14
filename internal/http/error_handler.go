package http

import (
	"errors"
	"fmt"
	"net/http"

	apperrors "file-service/pkg/errors"

	"github.com/labstack/echo/v4"
)

// CustomHTTPErrorHandler handles all errors returned by handlers and middleware.
// It maps sentinel errors to appropriate HTTP status codes, sanitizes internal errors,
// and logs errors with request context.
func CustomHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	message := "Internal server error"

	// Check for Echo HTTP errors first
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		code = httpErr.Code
		message = fmt.Sprintf("%v", httpErr.Message)
	} else {
		// Map sentinel errors to HTTP status codes
		switch {
		case errors.Is(err, apperrors.ErrNotFound):
			code = http.StatusNotFound
			message = "Resource not found"
		case errors.Is(err, apperrors.ErrUnauthorized):
			code = http.StatusUnauthorized
			message = "Unauthorized"
		case errors.Is(err, apperrors.ErrInvalidCredentials):
			code = http.StatusUnauthorized
			message = "Invalid credentials"
		case errors.Is(err, apperrors.ErrForbidden):
			code = http.StatusForbidden
			message = "Forbidden"
		case errors.Is(err, apperrors.ErrInsufficientPerms):
			code = http.StatusForbidden
			message = "Insufficient permissions"
		case errors.Is(err, apperrors.ErrBadRequest):
			code = http.StatusBadRequest
			message = "Bad request"
		case errors.Is(err, apperrors.ErrInvalidInput):
			code = http.StatusBadRequest
			message = "Invalid input"
		case errors.Is(err, apperrors.ErrValidation):
			code = http.StatusBadRequest
			message = "Validation error"
		case errors.Is(err, apperrors.ErrPathTraversal):
			code = http.StatusBadRequest
			message = "Invalid path"
		case errors.Is(err, apperrors.ErrFolderDepthExceeded):
			code = http.StatusBadRequest
			message = "Folder depth limit exceeded"
		case errors.Is(err, apperrors.ErrConflict):
			code = http.StatusConflict
			message = "Resource already exists"
		case errors.Is(err, apperrors.ErrEmailExists):
			code = http.StatusConflict
			message = "Email already exists"
		case errors.Is(err, apperrors.ErrExpired):
			code = http.StatusGone
			message = "Resource expired"
		case errors.Is(err, apperrors.ErrRevoked):
			code = http.StatusForbidden
			message = "Resource revoked"
		case errors.Is(err, apperrors.ErrAPIKeyRevoked):
			code = http.StatusForbidden
			message = "API key revoked"
		case errors.Is(err, apperrors.ErrAPIKeyExpired):
			code = http.StatusForbidden
			message = "API key expired"
		}

		// Check for custom AppError type
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			// Use the message from AppError if it's a client error
			if code < 500 {
				message = appErr.Message
			}
		}
	}

	// Log error with request context
	requestID := c.Response().Header().Get(echo.HeaderXRequestID)
	if requestID == "" {
		requestID = "unknown"
	}

	// Log with appropriate level
	if code >= 500 {
		c.Logger().Error("internal_server_error",
			"request_id", requestID,
			"status", code,
			"error", err.Error())
		// Don't expose internal errors to clients
		message = "Internal server error"
	} else {
		c.Logger().Warn("client_error",
			"request_id", requestID,
			"status", code,
			"error", err.Error())
	}

	// Send JSON error response
	if err := c.JSON(code, map[string]interface{}{
		"error":      message,
		"request_id": requestID,
	}); err != nil {
		c.Logger().Error(err)
	}
}

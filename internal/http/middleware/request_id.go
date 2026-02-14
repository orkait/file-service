package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	// RequestIDHeader is the header name for request ID
	RequestIDHeader = "X-Request-ID"
	// RequestIDContextKey is the context key for request ID
	RequestIDContextKey = "request_id"
)

// RequestID returns a middleware that generates or extracts a request ID
// and adds it to the response headers and context
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Try to get request ID from header
			requestID := c.Request().Header.Get(RequestIDHeader)

			// If not provided, generate a new one
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Store in context for use in handlers and logging
			c.Set(RequestIDContextKey, requestID)

			// Add to response headers
			c.Response().Header().Set(RequestIDHeader, requestID)

			return next(c)
		}
	}
}

// GetRequestID extracts the request ID from the context
func GetRequestID(c echo.Context) string {
	if requestID, ok := c.Get(RequestIDContextKey).(string); ok {
		return requestID
	}
	return ""
}

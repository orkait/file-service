package middleware

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	csrfTokenLength = 32
	csrfTokenTTL    = 24 * time.Hour
	csrfHeaderName  = "X-CSRF-Token"
	csrfCookieName  = "csrf_token"
	cleanupInterval = 1 * time.Hour
)

// CSRFToken represents a CSRF token with expiry
type CSRFToken struct {
	Token     string
	ExpiresAt time.Time
}

// CSRFMiddleware implements CSRF protection
type CSRFMiddleware struct {
	tokens  sync.Map // userID -> CSRFToken
	ctx     context.Context
	cancel  context.CancelFunc
	stopped chan struct{}
}

// NewCSRFMiddleware creates a new CSRF middleware with background cleanup
func NewCSRFMiddleware(ctx context.Context) *CSRFMiddleware {
	cleanupCtx, cancel := context.WithCancel(ctx)
	m := &CSRFMiddleware{
		ctx:     cleanupCtx,
		cancel:  cancel,
		stopped: make(chan struct{}),
	}

	// Start background cleanup goroutine
	go m.cleanupLoop()

	return m
}

// Stop gracefully stops the cleanup goroutine
func (m *CSRFMiddleware) Stop() {
	m.cancel()
	<-m.stopped
}

// cleanupLoop periodically removes expired tokens
func (m *CSRFMiddleware) cleanupLoop() {
	defer close(m.stopped)

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.CleanupExpiredTokens()
		}
	}
}

// generateToken generates a cryptographically secure random token
func generateToken() (string, error) {
	bytes := make([]byte, csrfTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GetOrCreateToken gets or creates a CSRF token for a user
func (m *CSRFMiddleware) GetOrCreateToken(userID uuid.UUID) (string, error) {
	// Check if token exists and is valid
	if tokenRaw, exists := m.tokens.Load(userID.String()); exists {
		if csrfToken, ok := tokenRaw.(*CSRFToken); ok {
			if time.Now().Before(csrfToken.ExpiresAt) {
				return csrfToken.Token, nil
			}
		}
	}

	// Generate new token
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	csrfToken := &CSRFToken{
		Token:     token,
		ExpiresAt: time.Now().Add(csrfTokenTTL),
	}

	m.tokens.Store(userID.String(), csrfToken)
	return token, nil
}

// Middleware returns an Echo middleware function for CSRF protection
func (m *CSRFMiddleware) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip CSRF check for safe methods
			method := c.Request().Method
			if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
				return next(c)
			}

			// Skip CSRF check for API key authentication
			// API keys are not vulnerable to CSRF attacks
			if c.Get("api_key_id") != nil {
				return next(c)
			}

			// Get user ID from context (set by JWT middleware)
			userIDRaw := c.Get("user_id")
			if userIDRaw == nil {
				// No user ID, skip CSRF check (unauthenticated request)
				return next(c)
			}

			userID, ok := userIDRaw.(uuid.UUID)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid user context",
				})
			}

			// Get expected token
			tokenRaw, exists := m.tokens.Load(userID.String())
			if !exists {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "CSRF token not found",
				})
			}

			csrfToken, ok := tokenRaw.(*CSRFToken)
			if !ok || time.Now().After(csrfToken.ExpiresAt) {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "CSRF token expired",
				})
			}

			// Get provided token from header
			providedToken := c.Request().Header.Get(csrfHeaderName)
			if providedToken == "" {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "CSRF token required",
				})
			}

			// Constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(providedToken), []byte(csrfToken.Token)) != 1 {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "invalid CSRF token",
				})
			}

			return next(c)
		}
	}
}

// CleanupExpiredTokens removes expired tokens (called by background goroutine)
func (m *CSRFMiddleware) CleanupExpiredTokens() {
	now := time.Now()
	m.tokens.Range(func(key, value any) bool {
		if csrfToken, ok := value.(*CSRFToken); ok {
			if now.After(csrfToken.ExpiresAt) {
				m.tokens.Delete(key)
			}
		}
		return true
	})
}

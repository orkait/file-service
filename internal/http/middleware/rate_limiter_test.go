package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(2, 2) // 2 req/sec, burst of 2

	// First two requests should succeed
	assert.True(t, rl.Allow("test-key"))
	assert.True(t, rl.Allow("test-key"))

	// Third request should be rate limited
	assert.False(t, rl.Allow("test-key"))
}

func TestRateLimiter_Middleware(t *testing.T) {
	e := echo.New()
	rl := NewRateLimiter(2, 2)

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	middleware := rl.Middleware()

	// First request should succeed
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middleware(handler)(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Remaining"))

	// Second request should succeed
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	err = middleware(handler)(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Third request should be rate limited
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	err = middleware(handler)(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, "1", rec.Header().Get("Retry-After"))
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	rl := NewRateLimiter(1, 1)

	// Different keys should have independent rate limits
	assert.True(t, rl.Allow("key1"))
	assert.True(t, rl.Allow("key2"))

	// Both keys should now be rate limited
	assert.False(t, rl.Allow("key1"))
	assert.False(t, rl.Allow("key2"))
}

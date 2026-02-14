package middleware

import (
	"file-service/internal/auth"
	"file-service/internal/domain/apikey"
	"fmt"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// RateLimiter implements token bucket rate limiting per identity
type RateLimiter struct {
	limiters sync.Map // key -> *rate.Limiter
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
// requestsPerSecond: number of requests allowed per second
// burst: maximum burst size
func NewRateLimiter(requestsPerSecond int, burst int) *RateLimiter {
	return &RateLimiter{
		rate:  rate.Limit(requestsPerSecond),
		burst: burst,
	}
}

// getLimiter gets or creates a rate limiter for the given key
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	limiter, exists := rl.limiters.Load(key)
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters.Store(key, limiter)
	}
	return limiter.(*rate.Limiter)
}

// Allow checks if a request should be allowed for the given key
func (rl *RateLimiter) Allow(key string) bool {
	return rl.getLimiter(key).Allow()
}

// Middleware returns an Echo middleware function for rate limiting
func (rl *RateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var key string

			authType := auth.GetAuthType(c)
			switch authType {
			case auth.AuthTypeJWT:
				// Rate limit by user ID
				userID, err := auth.GetUserID(c)
				if err == nil {
					key = "user:" + userID.String()
				} else {
					// Fallback to IP if user ID not available
					key = "ip:" + c.RealIP()
				}
			case auth.AuthTypeAPIKey:
				// Rate limit by API key ID
				keyRaw := c.Get(auth.ContextKeyAPIKey)
				if apiKey, ok := keyRaw.(*apikey.APIKey); ok && apiKey != nil {
					key = "apikey:" + apiKey.ID.String()
				} else {
					// Fallback to IP
					key = "ip:" + c.RealIP()
				}
			default:
				// Rate limit unauthenticated requests by IP
				key = "ip:" + c.RealIP()
			}

			limiter := rl.getLimiter(key)

			// Check rate limit
			if !limiter.Allow() {
				// Add rate limit headers
				c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.burst))
				c.Response().Header().Set("X-RateLimit-Remaining", "0")
				c.Response().Header().Set("Retry-After", "1")

				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "rate limit exceeded",
				})
			}

			// Add rate limit headers for successful requests
			tokens := int(limiter.Tokens())
			c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.burst))
			c.Response().Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", tokens))

			return next(c)
		}
	}
}

// StrictRateLimiter is a more aggressive rate limiter for sensitive endpoints
type StrictRateLimiter struct {
	*RateLimiter
}

// NewStrictRateLimiter creates a strict rate limiter for sensitive operations
func NewStrictRateLimiter() *StrictRateLimiter {
	return &StrictRateLimiter{
		RateLimiter: NewRateLimiter(5, 10), // 5 req/sec, burst of 10
	}
}

// GlobalRateLimiter is a lenient rate limiter for general API usage
type GlobalRateLimiter struct {
	*RateLimiter
}

// NewGlobalRateLimiter creates a global rate limiter
func NewGlobalRateLimiter() *GlobalRateLimiter {
	return &GlobalRateLimiter{
		RateLimiter: NewRateLimiter(100, 200), // 100 req/sec, burst of 200
	}
}

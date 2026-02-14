package middleware

import (
	"github.com/labstack/echo/v4"
)

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Content Security Policy
			// Restrict resource loading to prevent XSS attacks
			c.Response().Header().Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self'; "+
					"style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data: https:; "+
					"font-src 'self'; "+
					"connect-src 'self'; "+
					"frame-ancestors 'none'; "+
					"base-uri 'self'; "+
					"form-action 'self'")

			// HTTP Strict Transport Security (HSTS)
			// Force HTTPS for 1 year, including subdomains
			c.Response().Header().Set("Strict-Transport-Security",
				"max-age=31536000; includeSubDomains; preload")

			// X-Content-Type-Options
			// Prevent MIME type sniffing
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")

			// X-Frame-Options
			// Prevent clickjacking attacks
			c.Response().Header().Set("X-Frame-Options", "DENY")

			// X-XSS-Protection
			// Enable browser XSS protection (legacy, but still useful)
			c.Response().Header().Set("X-XSS-Protection", "1; mode=block")

			// Referrer-Policy
			// Control referrer information
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Permissions-Policy
			// Disable unnecessary browser features
			c.Response().Header().Set("Permissions-Policy",
				"geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=()")

			// Remove server identification header
			c.Response().Header().Del("Server")
			c.Response().Header().Del("X-Powered-By")

			return next(c)
		}
	}
}

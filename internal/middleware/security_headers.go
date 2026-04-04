package middleware

import "github.com/labstack/echo/v4"

// SecurityHeaders returns middleware that sets OWASP-recommended security headers.
// Per WitFoo Way Section 3.6.
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-XSS-Protection", "0") // Modern browsers: CSP instead
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			h.Set("Cache-Control", "no-store")
			h.Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'unsafe-inline' https://static.cloudflareinsights.com; "+
					"style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data: blob:; "+
					"font-src 'self'; "+
					"connect-src 'self' https://cloudflareinsights.com; "+
					"frame-ancestors 'none'; "+
					"base-uri 'self'; "+
					"form-action 'self'")
			return next(c)
		}
	}
}

package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
)

// isImmutableAsset reports whether the path serves a content-hashed, immutable
// static asset that is safe to cache aggressively (SvelteKit emits these under
// /_app/ with fingerprinted filenames, plus self-hosted fonts).
func isImmutableAsset(path string) bool {
	return strings.HasPrefix(path, "/_app/") ||
		strings.HasPrefix(path, "/fonts/") ||
		strings.HasSuffix(path, ".woff2")
}

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
			// Fingerprinted assets are immutable and cacheable; everything else
			// (API responses, the HTML shell) must never be stored.
			if isImmutableAsset(c.Request().URL.Path) {
				h.Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				h.Set("Cache-Control", "no-store")
			}
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

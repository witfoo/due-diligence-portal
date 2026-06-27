package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/service"
	"github.com/witfoo/due-diligence-portal/pkg/response"
)

// Context keys for storing claims.
const (
	ContextKeyUserID = "user_id"
	ContextKeyEmail  = "email"
	ContextKeyName   = "name"
	ContextKeyRole   = "role"
	ContextKeyClaims = "claims"
)

// JWTAuth returns middleware that validates JWT tokens from the Authorization header.
func JWTAuth(authSvc *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return response.Unauthorized(c, "missing authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return response.Unauthorized(c, "invalid authorization header format")
			}

			claims, err := authSvc.ValidateToken(parts[1])
			if err != nil {
				return response.Unauthorized(c, "invalid or expired token")
			}

			// Only access tokens authorize API calls; a refresh token must not be
			// usable as a bearer token (token-type confusion).
			if claims.TokenType != service.TokenTypeAccess {
				return response.Unauthorized(c, "invalid or expired token")
			}

			// Inject claims into Echo context.
			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyEmail, claims.Email)
			c.Set(ContextKeyName, claims.Name)
			c.Set(ContextKeyRole, claims.Role)
			c.Set(ContextKeyClaims, claims)

			return next(c)
		}
	}
}

// RequireRole returns middleware that checks if the user has one of the required roles.
func RequireRole(roles ...string) echo.MiddlewareFunc {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, ok := c.Get(ContextKeyRole).(string)
			if !ok || !roleSet[role] {
				return response.Forbidden(c, "insufficient permissions")
			}
			return next(c)
		}
	}
}

// GetUserID extracts the user ID from the Echo context.
func GetUserID(c echo.Context) string {
	if v, ok := c.Get(ContextKeyUserID).(string); ok {
		return v
	}
	return ""
}

// GetUserEmail extracts the user email from the Echo context.
func GetUserEmail(c echo.Context) string {
	if v, ok := c.Get(ContextKeyEmail).(string); ok {
		return v
	}
	return ""
}

// GetUserName extracts the authenticated user's name from the Echo context.
func GetUserName(c echo.Context) string {
	if v, ok := c.Get(ContextKeyName).(string); ok {
		return v
	}
	return ""
}

// GetUserRole extracts the user role from the Echo context.
func GetUserRole(c echo.Context) string {
	if v, ok := c.Get(ContextKeyRole).(string); ok {
		return v
	}
	return ""
}

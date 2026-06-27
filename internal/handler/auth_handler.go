package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/service"
	"github.com/witfoo/due-diligence-portal/pkg/response"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authSvc *service.AuthService
	audit   *middleware.AuditLogger
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authSvc *service.AuthService, audit *middleware.AuditLogger) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, audit: audit}
}

// RegisterRoutes registers auth routes. loginThrottle is a stricter per-IP rate
// limiter applied to the unauthenticated credential endpoints to resist online
// brute-force / credential-stuffing (the global limiter is too permissive for login).
func (h *AuthHandler) RegisterRoutes(e *echo.Echo, authMiddleware, loginThrottle echo.MiddlewareFunc) {
	e.POST("/api/v1/auth/login", h.Login, loginThrottle)
	e.POST("/api/v1/auth/register", h.Register, loginThrottle)
	e.POST("/api/v1/auth/refresh", h.Refresh, loginThrottle)
	e.GET("/api/v1/auth/me", h.Me, authMiddleware)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return response.BadRequest(c, "email and password are required")
	}

	result, err := h.authSvc.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		switch err {
		case domain.ErrInvalidCredentials:
			return response.Unauthorized(c, "invalid email or password")
		case domain.ErrAccountDisabled:
			return response.Forbidden(c, "account is disabled")
		default:
			return response.InternalError(c)
		}
	}

	h.audit.LogFromContext(c, domain.AuditUserLogin, "user", result.User.ID, result.User.Email, "")

	return response.OK(c, "Login successful", map[string]any{
		"user":          result.User,
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
	})
}

type registerRequest struct {
	Token    string `json:"token"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.Token == "" || req.Name == "" || req.Password == "" {
		return response.BadRequest(c, "token, name, and password are required")
	}

	if len(req.Password) < 8 {
		return response.BadRequest(c, "password must be at least 8 characters")
	}

	result, err := h.authSvc.Register(c.Request().Context(), req.Token, req.Name, req.Password)
	if err != nil {
		switch err {
		case domain.ErrTokenNotFound:
			return response.NotFound(c, "invalid invite token")
		case domain.ErrTokenUsed:
			return response.Conflict(c, "invite token already used")
		case domain.ErrTokenExpired:
			return response.BadRequest(c, "invite token has expired")
		default:
			return response.InternalError(c)
		}
	}

	h.audit.LogFromContext(c, domain.AuditUserCreated, "user", result.User.ID, result.User.Email, "registered via invite")

	return response.Created(c, "Registration successful", map[string]any{
		"user":          result.User,
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *AuthHandler) Refresh(c echo.Context) error {
	var req refreshRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.RefreshToken == "" {
		return response.BadRequest(c, "refresh_token is required")
	}

	newToken, err := h.authSvc.RefreshToken(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return response.Unauthorized(c, "invalid or expired refresh token")
	}

	return response.OK(c, "Token refreshed", map[string]any{
		"access_token": newToken,
	})
}

// Me handles GET /api/v1/auth/me.
func (h *AuthHandler) Me(c echo.Context) error {
	claims, ok := c.Get(middleware.ContextKeyClaims).(*service.JWTClaims)
	if !ok {
		return response.Unauthorized(c, "invalid session")
	}

	return response.OK(c, "Current user", map[string]any{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"name":    claims.Name,
		"role":    claims.Role,
	})
}

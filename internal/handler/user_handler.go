package handler

import (
	"log"

	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/internal/service"
	"github.com/witfoo/due-diligence-portal/pkg/response"
	"github.com/witfoo/due-diligence-portal/pkg/sanitize"
)

// UserHandler handles user management endpoints.
type UserHandler struct {
	userRepo repository.UserRepository
	authSvc  *service.AuthService
	emailSvc *service.EmailService
	audit    *middleware.AuditLogger
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userRepo repository.UserRepository, authSvc *service.AuthService, emailSvc *service.EmailService, audit *middleware.AuditLogger) *UserHandler {
	return &UserHandler{userRepo: userRepo, authSvc: authSvc, emailSvc: emailSvc, audit: audit}
}

// RegisterRoutes registers user management routes.
func (h *UserHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/users", h.List)
	g.GET("/users/:id", h.Get)
	g.PUT("/users/:id", h.Update)
	g.DELETE("/users/:id", h.Deactivate)
	g.POST("/users/invite", h.Invite)
}

// List handles GET /api/v1/users.
func (h *UserHandler) List(c echo.Context) error {
	limit := 50
	offset := 0

	users, total, err := h.userRepo.List(c.Request().Context(), limit, offset)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OKWithMeta(c, "Users retrieved", users, &response.Meta{
		Count:    len(users),
		Total:    total,
		Page:     1,
		PageSize: limit,
		HasMore:  offset+limit < total,
	})
}

// Get handles GET /api/v1/users/:id.
func (h *UserHandler) Get(c echo.Context) error {
	id := c.Param("id")
	user, err := h.userRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return response.NotFound(c, "user not found")
		}
		return response.InternalError(c)
	}
	return response.OK(c, "User retrieved", user)
}

type updateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsActive *bool  `json:"is_active"`
}

// Update handles PUT /api/v1/users/:id.
func (h *UserHandler) Update(c echo.Context) error {
	id := c.Param("id")
	var req updateUserRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	user, err := h.userRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return response.NotFound(c, "user not found")
		}
		return response.InternalError(c)
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		if err := domain.ValidateEmail(req.Email); err != nil {
			return response.BadRequest(c, err.Error())
		}
		user.Email = req.Email
	}
	if req.Role != "" {
		if err := domain.ValidateRole(req.Role); err != nil {
			return response.BadRequest(c, err.Error())
		}
		user.Role = req.Role
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := h.userRepo.Update(c.Request().Context(), user); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditUserUpdated, "user", user.ID, user.Email, "")

	return response.OK(c, "User updated", user)
}

// Deactivate handles DELETE /api/v1/users/:id.
func (h *UserHandler) Deactivate(c echo.Context) error {
	id := c.Param("id")

	// Prevent self-deactivation.
	if middleware.GetUserID(c) == id {
		return response.BadRequest(c, "cannot deactivate your own account")
	}

	if err := h.userRepo.Deactivate(c.Request().Context(), id); err != nil {
		if err == domain.ErrUserNotFound {
			return response.NotFound(c, "user not found")
		}
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditUserDeactivated, "user", id, "", "")

	return response.OK(c, "User deactivated", nil)
}

type inviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Invite handles POST /api/v1/users/invite.
func (h *UserHandler) Invite(c echo.Context) error {
	var req inviteRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	inviterID := middleware.GetUserID(c)
	invite, err := h.authSvc.CreateInvite(c.Request().Context(), req.Email, req.Role, inviterID)
	if err != nil {
		switch err {
		case domain.ErrInvalidEmail:
			return response.BadRequest(c, "invalid email format")
		case domain.ErrInvalidRole:
			return response.BadRequest(c, "invalid role")
		default:
			return response.InternalError(c)
		}
	}

	h.audit.LogFromContext(c, domain.AuditUserInvited, "invite", invite.ID, req.Email, "role="+req.Role)

	// Send invite email if SMTP is configured.
	if h.emailSvc != nil {
		portalURL := "https://" + c.Request().Host
		if err := h.emailSvc.SendInvite(req.Email, invite.Token, portalURL); err != nil {
			// Non-fatal: log but don't fail the invite creation.
			log.Printf("[WARN] Failed to send invite email to %s: %v",
				sanitize.LogValue(req.Email), err)
		}
	}

	return response.Created(c, "Invitation created", invite)
}

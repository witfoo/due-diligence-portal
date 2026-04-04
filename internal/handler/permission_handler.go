package handler

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/pkg/response"
)

// PermissionHandler handles access grant management endpoints.
type PermissionHandler struct {
	permRepo repository.PermissionRepository
	audit    *middleware.AuditLogger
}

// NewPermissionHandler creates a new PermissionHandler.
func NewPermissionHandler(permRepo repository.PermissionRepository, audit *middleware.AuditLogger) *PermissionHandler {
	return &PermissionHandler{permRepo: permRepo, audit: audit}
}

// RegisterRoutes registers permission routes on a pre-authenticated admin group.
func (h *PermissionHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/permissions/document/:id", h.ListByDocument)
	g.GET("/permissions/category/:id", h.ListByCategory)
	g.POST("/permissions", h.Grant)
	g.PUT("/permissions/:id", h.Update)
	g.DELETE("/permissions/:id", h.Revoke)
}

// ListByDocument handles GET /permissions/document/:id.
func (h *PermissionHandler) ListByDocument(c echo.Context) error {
	id := c.Param("id")
	grants, err := h.permRepo.ListByResource(c.Request().Context(), domain.ResourceDocument, id)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OK(c, "Grants retrieved", grants)
}

// ListByCategory handles GET /permissions/category/:id.
func (h *PermissionHandler) ListByCategory(c echo.Context) error {
	id := c.Param("id")
	grants, err := h.permRepo.ListByResource(c.Request().Context(), domain.ResourceCategory, id)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OK(c, "Grants retrieved", grants)
}

type grantRequest struct {
	UserID       string  `json:"user_id"`
	ResourceType string  `json:"resource_type"`
	ResourceID   string  `json:"resource_id"`
	AccessLevel  string  `json:"access_level"`
	ExpiresAt    *string `json:"expires_at"`
}

// Grant handles POST /permissions.
func (h *PermissionHandler) Grant(c echo.Context) error {
	var req grantRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.UserID == "" || req.ResourceType == "" || req.ResourceID == "" || req.AccessLevel == "" {
		return response.BadRequest(c, "user_id, resource_type, resource_id, and access_level are required")
	}

	if req.ResourceType != domain.ResourceDocument && req.ResourceType != domain.ResourceCategory {
		return response.BadRequest(c, "resource_type must be 'document' or 'category'")
	}

	id, err := generateHandlerID()
	if err != nil {
		return response.InternalError(c)
	}

	grant := &domain.AccessGrant{
		ID:           id,
		UserID:       req.UserID,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		AccessLevel:  req.AccessLevel,
		GrantedBy:    middleware.GetUserID(c),
	}

	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return response.BadRequest(c, "expires_at must be in RFC3339 format")
		}
		grant.ExpiresAt = &t
	}

	if err := h.permRepo.Grant(c.Request().Context(), grant); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditPermissionGranted, req.ResourceType, req.ResourceID,
		"", "user="+req.UserID+" level="+req.AccessLevel)

	return response.Created(c, "Access granted", grant)
}

type updateGrantRequest struct {
	AccessLevel string  `json:"access_level"`
	ExpiresAt   *string `json:"expires_at"`
}

// Update handles PUT /permissions/:id.
func (h *PermissionHandler) Update(c echo.Context) error {
	id := c.Param("id")
	var req updateGrantRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.AccessLevel == "" {
		return response.BadRequest(c, "access_level is required")
	}

	grant, err := h.permRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == domain.ErrGrantNotFound {
			return response.NotFound(c, "access grant not found")
		}
		return response.InternalError(c)
	}

	grant.AccessLevel = req.AccessLevel
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return response.BadRequest(c, "expires_at must be in RFC3339 format")
		}
		grant.ExpiresAt = &t
	}

	if err := h.permRepo.Update(c.Request().Context(), grant); err != nil {
		if err == domain.ErrGrantNotFound {
			return response.NotFound(c, "access grant not found")
		}
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditPermissionUpdated, grant.ResourceType, grant.ResourceID,
		"", "grant="+id+" level="+req.AccessLevel)

	return response.OK(c, "Access grant updated", grant)
}

// Revoke handles DELETE /permissions/:id.
func (h *PermissionHandler) Revoke(c echo.Context) error {
	id := c.Param("id")

	grant, err := h.permRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == domain.ErrGrantNotFound {
			return response.NotFound(c, "access grant not found")
		}
		return response.InternalError(c)
	}

	if err := h.permRepo.Revoke(c.Request().Context(), id); err != nil {
		if err == domain.ErrGrantNotFound {
			return response.NotFound(c, "access grant not found")
		}
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditPermissionRevoked, grant.ResourceType, grant.ResourceID,
		"", "user="+grant.UserID)

	return response.OK(c, "Access revoked", nil)
}

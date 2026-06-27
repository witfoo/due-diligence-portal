package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/pkg/response"
)

// AuditHandler handles audit log endpoints.
type AuditHandler struct {
	auditRepo repository.AuditRepository
	audit     *middleware.AuditLogger
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(auditRepo repository.AuditRepository, audit *middleware.AuditLogger) *AuditHandler {
	return &AuditHandler{auditRepo: auditRepo, audit: audit}
}

// RegisterRoutes registers audit routes (admin-only group).
func (h *AuditHandler) RegisterRoutes(g *echo.Group) {
	admin := g.Group("", middleware.RequireRole(domain.RoleAdmin))
	admin.GET("/audit", h.List)
	admin.GET("/audit/export", h.Export)
	admin.GET("/audit/document/:id", h.GetByDocument)
	admin.GET("/audit/user/:id", h.GetByUser)
}

// Export handles GET /audit/export, streaming the (optionally filtered) audit log
// as CSV for offline evidence/review.
func (h *AuditHandler) Export(c echo.Context) error {
	filter := repository.AuditFilter{
		Action:       c.QueryParam("action"),
		UserID:       c.QueryParam("user_id"),
		ResourceType: c.QueryParam("resource_type"),
	}

	const exportLimit = 10000
	entries, _, err := h.auditRepo.List(c.Request().Context(), filter, exportLimit, 0)
	if err != nil {
		return response.InternalError(c)
	}

	c.Response().Header().Set(echo.HeaderContentType, "text/csv; charset=utf-8")
	c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="audit-log.csv"`)
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	defer w.Flush()
	_ = w.Write([]string{"created_at", "user_email", "action", "resource_type", "resource_id", "resource_name", "details", "ip_address"})
	for _, e := range entries {
		_ = w.Write([]string{
			e.CreatedAt.Format(time.RFC3339), e.UserEmail, e.Action, e.ResourceType,
			e.ResourceID, e.ResourceName, e.Details, e.IPAddress,
		})
	}
	return nil
}

// List handles GET /audit.
func (h *AuditHandler) List(c echo.Context) error {
	limit := 50
	offset := 0

	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	filter := repository.AuditFilter{
		Action:       c.QueryParam("action"),
		UserID:       c.QueryParam("user_id"),
		ResourceType: c.QueryParam("resource_type"),
	}

	entries, total, err := h.auditRepo.List(c.Request().Context(), filter, limit, offset)
	if err != nil {
		return response.InternalError(c)
	}

	page := 1
	if limit > 0 {
		page = (offset / limit) + 1
	}

	return response.OKWithMeta(c, "Audit log retrieved", entries, &response.Meta{
		Count:    len(entries),
		Total:    total,
		Page:     page,
		PageSize: limit,
		HasMore:  offset+limit < total,
	})
}

// GetByDocument handles GET /audit/document/:id.
func (h *AuditHandler) GetByDocument(c echo.Context) error {
	documentID := c.Param("id")
	limit := 50
	offset := 0

	entries, total, err := h.auditRepo.GetByDocument(c.Request().Context(), documentID, limit, offset)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OKWithMeta(c, "Document audit trail retrieved", entries, &response.Meta{
		Count:    len(entries),
		Total:    total,
		Page:     1,
		PageSize: limit,
		HasMore:  offset+limit < total,
	})
}

// GetByUser handles GET /audit/user/:id.
func (h *AuditHandler) GetByUser(c echo.Context) error {
	userID := c.Param("id")
	limit := 50
	offset := 0

	entries, total, err := h.auditRepo.GetByUser(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OKWithMeta(c, "User activity retrieved", entries, &response.Meta{
		Count:    len(entries),
		Total:    total,
		Page:     1,
		PageSize: limit,
		HasMore:  offset+limit < total,
	})
}

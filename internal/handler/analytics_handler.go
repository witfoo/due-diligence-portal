package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/pkg/response"
)

// AnalyticsHandler handles analytics endpoints.
type AnalyticsHandler struct {
	analyticsRepo repository.AnalyticsRepository
	audit         *middleware.AuditLogger
}

// NewAnalyticsHandler creates a new AnalyticsHandler.
func NewAnalyticsHandler(analyticsRepo repository.AnalyticsRepository, audit *middleware.AuditLogger) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsRepo: analyticsRepo, audit: audit}
}

// RegisterRoutes registers analytics routes on the given group.
func (h *AnalyticsHandler) RegisterRoutes(g *echo.Group) {
	adminCompany := middleware.RequireRole(domain.RoleAdmin, domain.RoleCompanyMember)
	g.GET("/analytics/dashboard", h.Dashboard, adminCompany)
	g.GET("/analytics/documents/:id", h.DocumentAnalytics, adminCompany)
	g.GET("/analytics/users/:id", h.UserAnalytics, adminCompany)
	g.POST("/analytics/view-event", h.RecordViewEvent)
}

type viewEventRequest struct {
	DocumentID string `json:"document_id"`
	DurationMs *int64 `json:"duration_ms,omitempty"`
	PageCount  *int   `json:"page_count,omitempty"`
}

// Dashboard handles GET /analytics/dashboard.
func (h *AnalyticsHandler) Dashboard(c echo.Context) error {
	summary, err := h.analyticsRepo.GetEngagementSummary(c.Request().Context())
	if err != nil {
		return response.InternalError(c)
	}

	return response.OK(c, "Dashboard retrieved", summary)
}

// DocumentAnalytics handles GET /analytics/documents/:id.
func (h *AnalyticsHandler) DocumentAnalytics(c echo.Context) error {
	documentID := c.Param("id")

	analytics, err := h.analyticsRepo.GetDocumentAnalytics(c.Request().Context(), documentID)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OK(c, "Document analytics retrieved", analytics)
}

// UserAnalytics handles GET /analytics/users/:id.
func (h *AnalyticsHandler) UserAnalytics(c echo.Context) error {
	userID := c.Param("id")

	analytics, err := h.analyticsRepo.GetUserAnalytics(c.Request().Context(), userID)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OK(c, "User analytics retrieved", analytics)
}

// RecordViewEvent handles POST /analytics/view-event.
func (h *AnalyticsHandler) RecordViewEvent(c echo.Context) error {
	var req viewEventRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.DocumentID == "" {
		return response.BadRequest(c, "document_id is required")
	}

	id, err := generateHandlerID()
	if err != nil {
		return response.InternalError(c)
	}

	event := &domain.ViewEvent{
		ID:         id,
		UserID:     middleware.GetUserID(c),
		DocumentID: req.DocumentID,
		DurationMs: req.DurationMs,
		PageCount:  req.PageCount,
	}

	if err := h.analyticsRepo.RecordView(c.Request().Context(), event); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditDocumentViewed, "document", req.DocumentID, "", "")

	return response.Created(c, "View event recorded", event)
}

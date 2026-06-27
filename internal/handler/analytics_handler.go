package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/pkg/response"
)

// maxViewDurationMs caps a single recorded view duration at 24 hours so a client
// cannot skew average-duration analytics with absurd values.
const maxViewDurationMs int64 = 24 * 60 * 60 * 1000

// maxViewPageCount caps a single recorded page count.
const maxViewPageCount = 100000

// AnalyticsHandler handles analytics endpoints.
type AnalyticsHandler struct {
	analyticsRepo repository.AnalyticsRepository
	docRepo       repository.DocumentRepository
	permRepo      repository.PermissionRepository
	userRepo      repository.UserRepository
	audit         *middleware.AuditLogger
}

// NewAnalyticsHandler creates a new AnalyticsHandler.
func NewAnalyticsHandler(analyticsRepo repository.AnalyticsRepository, docRepo repository.DocumentRepository, permRepo repository.PermissionRepository, userRepo repository.UserRepository, audit *middleware.AuditLogger) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsRepo: analyticsRepo, docRepo: docRepo, permRepo: permRepo, userRepo: userRepo, audit: audit}
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

	// Return 404 for a document that does not exist rather than zero-value stats.
	if _, err := h.docRepo.GetByID(c.Request().Context(), documentID); err != nil {
		if err == domain.ErrDocumentNotFound {
			return response.NotFound(c, "document not found")
		}
		return response.InternalError(c)
	}

	analytics, err := h.analyticsRepo.GetDocumentAnalytics(c.Request().Context(), documentID)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OK(c, "Document analytics retrieved", analytics)
}

// UserAnalytics handles GET /analytics/users/:id.
func (h *AnalyticsHandler) UserAnalytics(c echo.Context) error {
	userID := c.Param("id")

	// Return 404 for a user that does not exist rather than zero-value stats.
	if _, err := h.userRepo.GetByID(c.Request().Context(), userID); err != nil {
		if err == domain.ErrUserNotFound {
			return response.NotFound(c, "user not found")
		}
		return response.InternalError(c)
	}

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

	// Only record a view for a document the caller can actually access; otherwise
	// any authenticated user could fabricate views for documents they were never
	// granted, poisoning engagement analytics.
	doc, err := h.docRepo.GetByID(c.Request().Context(), req.DocumentID)
	if err != nil {
		if err == domain.ErrDocumentNotFound {
			return response.NotFound(c, "document not found")
		}
		return response.InternalError(c)
	}
	if ok, err := documentAccessAllowed(c.Request().Context(), h.permRepo,
		middleware.GetUserID(c), middleware.GetUserRole(c), doc, domain.AccessView); err != nil {
		return response.InternalError(c)
	} else if !ok {
		return response.NotFound(c, "document not found")
	}

	// Clamp client-supplied metrics to sane, non-negative bounds.
	duration := req.DurationMs
	if duration != nil {
		v := *duration
		if v < 0 {
			v = 0
		} else if v > maxViewDurationMs {
			v = maxViewDurationMs
		}
		duration = &v
	}
	pageCount := req.PageCount
	if pageCount != nil {
		v := *pageCount
		if v < 0 {
			v = 0
		} else if v > maxViewPageCount {
			v = maxViewPageCount
		}
		pageCount = &v
	}

	id, err := generateHandlerID()
	if err != nil {
		return response.InternalError(c)
	}

	event := &domain.ViewEvent{
		ID:         id,
		UserID:     middleware.GetUserID(c),
		DocumentID: req.DocumentID,
		DurationMs: duration,
		PageCount:  pageCount,
	}

	if err := h.analyticsRepo.RecordView(c.Request().Context(), event); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditDocumentViewed, "document", req.DocumentID, "", "")

	return response.Created(c, "View event recorded", event)
}

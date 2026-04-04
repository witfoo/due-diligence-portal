package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/pkg/response"
)

// QAHandler handles Q&A endpoints.
type QAHandler struct {
	qaRepo repository.QARepository
	audit  *middleware.AuditLogger
}

// NewQAHandler creates a new QAHandler.
func NewQAHandler(qaRepo repository.QARepository, audit *middleware.AuditLogger) *QAHandler {
	return &QAHandler{qaRepo: qaRepo, audit: audit}
}

// RegisterRoutes registers Q&A routes on the given group.
func (h *QAHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/qa", h.ListThreads)
	g.POST("/qa", h.CreateThread)
	g.GET("/qa/:id", h.GetThread)
	g.POST("/qa/:id/messages", h.PostMessage)
	g.PATCH("/qa/:id/status", h.ChangeStatus, middleware.RequireRole(domain.RoleAdmin, domain.RoleCompanyMember))
}

type createThreadRequest struct {
	Subject    string  `json:"subject"`
	DocumentID *string `json:"document_id,omitempty"`
	CategoryID *string `json:"category_id,omitempty"`
}

// ListThreads handles GET /qa.
func (h *QAHandler) ListThreads(c echo.Context) error {
	status := c.QueryParam("status")
	limit := 50
	offset := 0

	threads, total, err := h.qaRepo.ListThreads(c.Request().Context(), status, limit, offset)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OKWithMeta(c, "Threads retrieved", threads, &response.Meta{
		Count:    len(threads),
		Total:    total,
		Page:     1,
		PageSize: limit,
		HasMore:  offset+limit < total,
	})
}

// CreateThread handles POST /qa.
func (h *QAHandler) CreateThread(c echo.Context) error {
	var req createThreadRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.Subject == "" {
		return response.BadRequest(c, "subject is required")
	}

	id, err := generateHandlerID()
	if err != nil {
		return response.InternalError(c)
	}

	thread := &domain.QAThread{
		ID:         id,
		Subject:    req.Subject,
		DocumentID: req.DocumentID,
		CategoryID: req.CategoryID,
		Status:     domain.QAStatusOpen,
		AskedBy:    middleware.GetUserID(c),
	}

	if err := h.qaRepo.CreateThread(c.Request().Context(), thread); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditQACreated, "qa_thread", thread.ID, thread.Subject, "")

	return response.Created(c, "Thread created", thread)
}

// GetThread handles GET /qa/:id.
func (h *QAHandler) GetThread(c echo.Context) error {
	id := c.Param("id")

	thread, err := h.qaRepo.GetThread(c.Request().Context(), id)
	if err != nil {
		if err == domain.ErrThreadNotFound {
			return response.NotFound(c, "thread not found")
		}
		return response.InternalError(c)
	}

	messages, err := h.qaRepo.ListMessages(c.Request().Context(), id)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OK(c, "Thread retrieved", map[string]any{
		"thread":   thread,
		"messages": messages,
	})
}

type postMessageRequest struct {
	Body       string `json:"body"`
	IsInternal *bool  `json:"is_internal,omitempty"`
}

// PostMessage handles POST /qa/:id/messages.
func (h *QAHandler) PostMessage(c echo.Context) error {
	threadID := c.Param("id")

	var req postMessageRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.Body == "" {
		return response.BadRequest(c, "body is required")
	}

	// Verify thread exists.
	if _, err := h.qaRepo.GetThread(c.Request().Context(), threadID); err != nil {
		if err == domain.ErrThreadNotFound {
			return response.NotFound(c, "thread not found")
		}
		return response.InternalError(c)
	}

	isInternal := false
	if req.IsInternal != nil {
		role := middleware.GetUserRole(c)
		if role == domain.RoleAdmin || role == domain.RoleCompanyMember {
			isInternal = *req.IsInternal
		}
	}

	id, err := generateHandlerID()
	if err != nil {
		return response.InternalError(c)
	}

	msg := &domain.QAMessage{
		ID:         id,
		ThreadID:   threadID,
		AuthorID:   middleware.GetUserID(c),
		Body:       req.Body,
		IsInternal: isInternal,
	}

	if err := h.qaRepo.CreateMessage(c.Request().Context(), msg); err != nil {
		return response.InternalError(c)
	}

	return response.Created(c, "Message posted", msg)
}

type changeStatusRequest struct {
	Status string `json:"status"`
}

// ChangeStatus handles PATCH /qa/:id/status.
func (h *QAHandler) ChangeStatus(c echo.Context) error {
	threadID := c.Param("id")

	var req changeStatusRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.Status == "" {
		return response.BadRequest(c, "status is required")
	}

	if req.Status != domain.QAStatusOpen && req.Status != domain.QAStatusAnswered && req.Status != domain.QAStatusClosed {
		return response.BadRequest(c, "invalid status")
	}

	if err := h.qaRepo.UpdateThreadStatus(c.Request().Context(), threadID, req.Status); err != nil {
		if err == domain.ErrThreadNotFound {
			return response.NotFound(c, "thread not found")
		}
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditQAClosed, "qa_thread", threadID, "", "status="+req.Status)

	return response.OK(c, "Status updated", map[string]any{
		"status": req.Status,
	})
}

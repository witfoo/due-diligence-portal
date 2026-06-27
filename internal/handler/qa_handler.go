package handler

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/internal/service"
	"github.com/witfoo/due-diligence-portal/pkg/response"
	"github.com/witfoo/due-diligence-portal/pkg/sanitize"
)

// QAHandler handles Q&A endpoints.
type QAHandler struct {
	qaRepo   repository.QARepository
	permRepo repository.PermissionRepository
	userRepo repository.UserRepository
	emailSvc *service.EmailService
	audit    *middleware.AuditLogger
}

// NewQAHandler creates a new QAHandler.
func NewQAHandler(qaRepo repository.QARepository, permRepo repository.PermissionRepository, userRepo repository.UserRepository, emailSvc *service.EmailService, audit *middleware.AuditLogger) *QAHandler {
	return &QAHandler{qaRepo: qaRepo, permRepo: permRepo, userRepo: userRepo, emailSvc: emailSvc, audit: audit}
}

// isPrivileged reports whether the role is staff (admin or company member), which
// may see all threads/messages; other roles (investors) are scoped to their own.
func isPrivileged(role string) bool {
	return role == domain.RoleAdmin || role == domain.RoleCompanyMember
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

	// Investors only see their own threads; staff see all.
	askedBy := ""
	if !isPrivileged(middleware.GetUserRole(c)) {
		askedBy = middleware.GetUserID(c)
	}

	threads, total, err := h.qaRepo.ListThreads(c.Request().Context(), status, askedBy, limit, offset)
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

	// Investors may only attach a question to a document/category they can access,
	// preventing them from referencing (and revealing) resources they lack a grant for.
	if !isPrivileged(middleware.GetUserRole(c)) {
		userID := middleware.GetUserID(c)
		ctx := c.Request().Context()
		if req.DocumentID != nil && *req.DocumentID != "" {
			ok, err := h.permRepo.HasAccess(ctx, userID, domain.ResourceDocument, *req.DocumentID, domain.AccessView)
			if err != nil {
				return response.InternalError(c)
			}
			if !ok {
				return response.Forbidden(c, "no access to the referenced document")
			}
		}
		if req.CategoryID != nil && *req.CategoryID != "" {
			ok, err := h.permRepo.HasAccess(ctx, userID, domain.ResourceCategory, *req.CategoryID, domain.AccessView)
			if err != nil {
				return response.InternalError(c)
			}
			if !ok {
				return response.Forbidden(c, "no access to the referenced category")
			}
		}
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

	// Investors may only view their own thread, and never company-internal messages.
	privileged := isPrivileged(middleware.GetUserRole(c))
	if !privileged && thread.AskedBy != middleware.GetUserID(c) {
		return response.NotFound(c, "thread not found")
	}

	messages, err := h.qaRepo.ListMessages(c.Request().Context(), id, privileged)
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

	// Verify thread exists and the caller may post to it (investors only on their own).
	thread, err := h.qaRepo.GetThread(c.Request().Context(), threadID)
	if err != nil {
		if err == domain.ErrThreadNotFound {
			return response.NotFound(c, "thread not found")
		}
		return response.InternalError(c)
	}
	privileged := isPrivileged(middleware.GetUserRole(c))
	if !privileged && thread.AskedBy != middleware.GetUserID(c) {
		return response.NotFound(c, "thread not found")
	}

	isInternal := false
	if req.IsInternal != nil && privileged {
		isInternal = *req.IsInternal
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

	h.audit.LogFromContext(c, domain.AuditQAMessagePosted, "qa_thread", threadID, thread.Subject,
		fmt.Sprintf("internal=%t", isInternal))

	// Best-effort: notify the asker when staff post a non-internal reply on their
	// thread (no-op when SMTP is disabled).
	if h.emailSvc != nil && privileged && !isInternal && thread.AskedBy != middleware.GetUserID(c) {
		if asker, err := h.userRepo.GetByID(c.Request().Context(), thread.AskedBy); err == nil && asker.Email != "" {
			authorName := middleware.GetUserName(c)
			if err := h.emailSvc.SendQANotification(asker.Email, thread.Subject, thread.Subject, authorName, req.Body); err != nil {
				log.Printf("[WARN] Failed to send Q&A notification: %v", sanitize.LogValue(err.Error()))
			}
		}
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

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

// NDAHandler handles NDA endpoints.
type NDAHandler struct {
	ndaRepo    repository.NDARepository
	emailSvc   *service.EmailService
	adminEmail string
	audit      *middleware.AuditLogger
}

// NewNDAHandler creates a new NDAHandler. adminEmail receives NDA-signed
// notifications (best-effort, only when SMTP is enabled).
func NewNDAHandler(ndaRepo repository.NDARepository, emailSvc *service.EmailService, adminEmail string, audit *middleware.AuditLogger) *NDAHandler {
	return &NDAHandler{ndaRepo: ndaRepo, emailSvc: emailSvc, adminEmail: adminEmail, audit: audit}
}

// RegisterRoutes registers NDA routes on the given group.
func (h *NDAHandler) RegisterRoutes(g *echo.Group) {
	adminOnly := middleware.RequireRole(domain.RoleAdmin)
	g.GET("/nda/templates", h.ListTemplates, adminOnly)
	g.POST("/nda/templates", h.CreateTemplate, adminOnly)
	g.PUT("/nda/templates/:id", h.UpdateTemplate, adminOnly)
	g.GET("/nda/status", h.CheckStatus)
	g.GET("/nda/active", h.ActiveTemplate)
	g.POST("/nda/sign/:templateId", h.Sign)
	g.GET("/nda/signatures", h.ListSignatures, adminOnly)
}

// ActiveTemplate handles GET /nda/active, returning the current active NDA template
// (id, name, content) to any authenticated user so they can read and sign it. Unlike
// ListTemplates this is not admin-only, since investors must see the NDA to sign it.
func (h *NDAHandler) ActiveTemplate(c echo.Context) error {
	templates, err := h.ndaRepo.ListTemplates(c.Request().Context())
	if err != nil {
		return response.InternalError(c)
	}
	for _, tmpl := range templates {
		if tmpl.IsActive {
			return response.OK(c, "Active NDA template", tmpl)
		}
	}
	return response.NotFound(c, "no active NDA template")
}

type createTemplateRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type updateTemplateRequest struct {
	Name     string `json:"name,omitempty"`
	Content  string `json:"content,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
}

type signRequest struct {
	SignerName    string `json:"signer_name"`
	SignerCompany string `json:"signer_company,omitempty"`
}

// ListTemplates handles GET /nda/templates.
func (h *NDAHandler) ListTemplates(c echo.Context) error {
	templates, err := h.ndaRepo.ListTemplates(c.Request().Context())
	if err != nil {
		return response.InternalError(c)
	}
	return response.OK(c, "Templates retrieved", templates)
}

// CreateTemplate handles POST /nda/templates.
func (h *NDAHandler) CreateTemplate(c echo.Context) error {
	var req createTemplateRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.Name == "" || req.Content == "" {
		return response.BadRequest(c, "name and content are required")
	}

	id, err := generateHandlerID()
	if err != nil {
		return response.InternalError(c)
	}

	tmpl := &domain.NDATemplate{
		ID:        id,
		Name:      req.Name,
		Content:   req.Content,
		IsActive:  true,
		Version:   1,
		CreatedBy: middleware.GetUserID(c),
	}

	if err := h.ndaRepo.CreateTemplate(c.Request().Context(), tmpl); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditNDACreated, "nda_template", tmpl.ID, tmpl.Name, "")

	return response.Created(c, "Template created", tmpl)
}

// UpdateTemplate handles PUT /nda/templates/:id.
func (h *NDAHandler) UpdateTemplate(c echo.Context) error {
	id := c.Param("id")

	var req updateTemplateRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	tmpl, err := h.ndaRepo.GetTemplate(c.Request().Context(), id)
	if err != nil {
		if err == domain.ErrTemplateNotFound {
			return response.NotFound(c, "template not found")
		}
		return response.InternalError(c)
	}

	if req.Name != "" {
		tmpl.Name = req.Name
	}
	if req.Content != "" {
		tmpl.Content = req.Content
	}
	if req.IsActive != nil {
		tmpl.IsActive = *req.IsActive
	}

	if err := h.ndaRepo.UpdateTemplate(c.Request().Context(), tmpl); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditNDAUpdated, "nda_template", tmpl.ID, tmpl.Name, "")

	return response.OK(c, "Template updated", tmpl)
}

// CheckStatus handles GET /nda/status.
func (h *NDAHandler) CheckStatus(c echo.Context) error {
	userID := middleware.GetUserID(c)

	// Find active templates.
	templates, err := h.ndaRepo.ListTemplates(c.Request().Context())
	if err != nil {
		return response.InternalError(c)
	}

	// Check if user has signed any active template.
	for _, tmpl := range templates {
		if !tmpl.IsActive {
			continue
		}
		signed, err := h.ndaRepo.HasSigned(c.Request().Context(), userID, tmpl.ID)
		if err != nil {
			return response.InternalError(c)
		}
		if signed {
			return response.OK(c, "NDA status", map[string]any{
				"signed":      true,
				"template_id": tmpl.ID,
			})
		}
	}

	return response.OK(c, "NDA status", map[string]any{
		"signed": false,
	})
}

// Sign handles POST /nda/sign/:templateId.
func (h *NDAHandler) Sign(c echo.Context) error {
	templateID := c.Param("templateId")

	var req signRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.SignerName == "" {
		return response.BadRequest(c, "signer_name is required")
	}

	// Verify template exists and is active (signing a retired template is meaningless).
	tmpl, err := h.ndaRepo.GetTemplate(c.Request().Context(), templateID)
	if err != nil {
		if err == domain.ErrTemplateNotFound {
			return response.NotFound(c, "template not found")
		}
		return response.InternalError(c)
	}
	if !tmpl.IsActive {
		return response.BadRequest(c, "template is not active")
	}

	userID := middleware.GetUserID(c)

	// Check if already signed.
	signed, err := h.ndaRepo.HasSigned(c.Request().Context(), userID, templateID)
	if err != nil {
		return response.InternalError(c)
	}
	if signed {
		return response.Conflict(c, "NDA already signed")
	}

	id, err := generateHandlerID()
	if err != nil {
		return response.InternalError(c)
	}

	// Bind the signer identity to the authenticated account rather than trusting a
	// client-supplied name; fall back to the submitted name only if the account has
	// no name on record. user_id and email are always the authenticated values.
	signerName := middleware.GetUserName(c)
	if signerName == "" {
		signerName = req.SignerName
	}

	sig := &domain.NDASignature{
		ID:            id,
		TemplateID:    templateID,
		UserID:        userID,
		SignerName:    signerName,
		SignerEmail:   middleware.GetUserEmail(c),
		SignerCompany: req.SignerCompany,
		IPAddress:     c.RealIP(),
	}

	if err := h.ndaRepo.CreateSignature(c.Request().Context(), sig); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditNDASigned, "nda_template", tmpl.ID, tmpl.Name, "signer="+signerName)

	// Best-effort notify the admin that an NDA was signed (no-op when SMTP disabled).
	if h.emailSvc != nil && h.adminEmail != "" {
		if err := h.emailSvc.SendNDASignedNotification(h.adminEmail, signerName, sig.SignerEmail, tmpl.Name); err != nil {
			log.Printf("[WARN] Failed to send NDA-signed notification: %v", sanitize.LogValue(err.Error()))
		}
	}

	return response.Created(c, "NDA signed", sig)
}

// ListSignatures handles GET /nda/signatures.
func (h *NDAHandler) ListSignatures(c echo.Context) error {
	// List all templates first, then collect all signatures.
	templates, err := h.ndaRepo.ListTemplates(c.Request().Context())
	if err != nil {
		return response.InternalError(c)
	}

	var allSigs []*domain.NDASignature
	for _, tmpl := range templates {
		sigs, err := h.ndaRepo.ListSignatures(c.Request().Context(), tmpl.ID)
		if err != nil {
			return response.InternalError(c)
		}
		allSigs = append(allSigs, sigs...)
	}

	return response.OK(c, "Signatures retrieved", allSigs)
}

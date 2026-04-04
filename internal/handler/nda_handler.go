package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/pkg/response"
)

// NDAHandler handles NDA endpoints.
type NDAHandler struct {
	ndaRepo repository.NDARepository
	audit   *middleware.AuditLogger
}

// NewNDAHandler creates a new NDAHandler.
func NewNDAHandler(ndaRepo repository.NDARepository, audit *middleware.AuditLogger) *NDAHandler {
	return &NDAHandler{ndaRepo: ndaRepo, audit: audit}
}

// RegisterRoutes registers NDA routes on the given group.
func (h *NDAHandler) RegisterRoutes(g *echo.Group) {
	adminOnly := middleware.RequireRole(domain.RoleAdmin)
	g.GET("/nda/templates", h.ListTemplates, adminOnly)
	g.POST("/nda/templates", h.CreateTemplate, adminOnly)
	g.PUT("/nda/templates/:id", h.UpdateTemplate, adminOnly)
	g.GET("/nda/status", h.CheckStatus)
	g.POST("/nda/sign/:templateId", h.Sign)
	g.GET("/nda/signatures", h.ListSignatures, adminOnly)
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

	// Verify template exists.
	tmpl, err := h.ndaRepo.GetTemplate(c.Request().Context(), templateID)
	if err != nil {
		if err == domain.ErrTemplateNotFound {
			return response.NotFound(c, "template not found")
		}
		return response.InternalError(c)
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

	sig := &domain.NDASignature{
		ID:            id,
		TemplateID:    templateID,
		UserID:        userID,
		SignerName:    req.SignerName,
		SignerEmail:   middleware.GetUserEmail(c),
		SignerCompany: req.SignerCompany,
		IPAddress:     c.RealIP(),
	}

	if err := h.ndaRepo.CreateSignature(c.Request().Context(), sig); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditNDASigned, "nda_template", tmpl.ID, tmpl.Name, "signer="+req.SignerName)

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

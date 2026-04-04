package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/pkg/response"
)

// CategoryHandler handles category management endpoints.
type CategoryHandler struct {
	catRepo repository.CategoryRepository
	docRepo repository.DocumentRepository
	audit   *middleware.AuditLogger
}

// NewCategoryHandler creates a new CategoryHandler.
func NewCategoryHandler(catRepo repository.CategoryRepository, docRepo repository.DocumentRepository, audit *middleware.AuditLogger) *CategoryHandler {
	return &CategoryHandler{catRepo: catRepo, docRepo: docRepo, audit: audit}
}

// RegisterRoutes registers category routes on a pre-authenticated group.
// List is available to all authenticated users; create/update/delete require admin.
func (h *CategoryHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/categories", h.List)
	g.POST("/categories", h.Create, middleware.RequireRole(domain.RoleAdmin))
	g.PUT("/categories/:id", h.Update, middleware.RequireRole(domain.RoleAdmin))
	g.DELETE("/categories/:id", h.Delete, middleware.RequireRole(domain.RoleAdmin))
}

// List handles GET /categories (tree structure).
func (h *CategoryHandler) List(c echo.Context) error {
	cats, err := h.catRepo.ListAsTree(c.Request().Context())
	if err != nil {
		return response.InternalError(c)
	}

	return response.OK(c, "Categories retrieved", cats)
}

type createCategoryRequest struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	ParentID    *string `json:"parent_id"`
	SortOrder   int     `json:"sort_order"`
	Icon        string  `json:"icon"`
}

// Create handles POST /categories.
func (h *CategoryHandler) Create(c echo.Context) error {
	var req createCategoryRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.Name == "" {
		return response.BadRequest(c, "name is required")
	}
	if req.Slug == "" {
		return response.BadRequest(c, "slug is required")
	}

	id, err := generateHandlerID()
	if err != nil {
		return response.InternalError(c)
	}

	cat := &domain.Category{
		ID:          id,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		ParentID:    req.ParentID,
		SortOrder:   req.SortOrder,
		Icon:        req.Icon,
	}

	if err := h.catRepo.Create(c.Request().Context(), cat); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditCategoryCreated, "category", cat.ID, cat.Name, "")

	return response.Created(c, "Category created", cat)
}

type updateCategoryRequest struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	ParentID    *string `json:"parent_id"`
	SortOrder   *int    `json:"sort_order"`
	Icon        string  `json:"icon"`
}

// Update handles PUT /categories/:id.
func (h *CategoryHandler) Update(c echo.Context) error {
	id := c.Param("id")
	var req updateCategoryRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	cat, err := h.catRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == domain.ErrCategoryNotFound {
			return response.NotFound(c, "category not found")
		}
		return response.InternalError(c)
	}

	if req.Name != "" {
		cat.Name = req.Name
	}
	if req.Slug != "" {
		cat.Slug = req.Slug
	}
	if req.Description != "" {
		cat.Description = req.Description
	}
	if req.ParentID != nil {
		cat.ParentID = req.ParentID
	}
	if req.SortOrder != nil {
		cat.SortOrder = *req.SortOrder
	}
	if req.Icon != "" {
		cat.Icon = req.Icon
	}

	if err := h.catRepo.Update(c.Request().Context(), cat); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditCategoryUpdated, "category", cat.ID, cat.Name, "")

	return response.OK(c, "Category updated", cat)
}

// Delete handles DELETE /categories/:id.
// Fails if the category has documents.
func (h *CategoryHandler) Delete(c echo.Context) error {
	id := c.Param("id")

	// Check that the category exists.
	cat, err := h.catRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == domain.ErrCategoryNotFound {
			return response.NotFound(c, "category not found")
		}
		return response.InternalError(c)
	}

	// Check if category has documents.
	docs, total, err := h.docRepo.List(c.Request().Context(), id, 1, 0)
	if err != nil {
		return response.InternalError(c)
	}
	_ = docs
	if total > 0 {
		return response.Conflict(c, "category has documents and cannot be deleted")
	}

	if err := h.catRepo.Delete(c.Request().Context(), id); err != nil {
		if err == domain.ErrCategoryNotFound {
			return response.NotFound(c, "category not found")
		}
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditCategoryDeleted, "category", id, cat.Name, "")

	return response.OK(c, "Category deleted", nil)
}

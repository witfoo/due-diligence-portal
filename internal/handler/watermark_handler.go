package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/pkg/response"
)

// WatermarkHandler handles watermark configuration endpoints.
type WatermarkHandler struct {
	wmRepo repository.WatermarkRepository
	audit  *middleware.AuditLogger
}

// NewWatermarkHandler creates a new WatermarkHandler.
func NewWatermarkHandler(wmRepo repository.WatermarkRepository, audit *middleware.AuditLogger) *WatermarkHandler {
	return &WatermarkHandler{wmRepo: wmRepo, audit: audit}
}

// RegisterRoutes registers watermark routes on the given group.
func (h *WatermarkHandler) RegisterRoutes(g *echo.Group) {
	adminOnly := middleware.RequireRole(domain.RoleAdmin)
	g.GET("/watermark", h.GetConfig, adminOnly)
	g.PUT("/watermark", h.UpdateConfig, adminOnly)
	g.DELETE("/watermark", h.ResetConfig, adminOnly)
}

// GetConfig handles GET /api/v1/watermark.
func (h *WatermarkHandler) GetConfig(c echo.Context) error {
	config, err := h.wmRepo.GetConfig(c.Request().Context())
	if err != nil {
		return response.InternalError(c)
	}
	return response.OK(c, "Watermark config retrieved", config)
}

type updateWatermarkRequest struct {
	Enabled      *bool    `json:"enabled"`
	TextTemplate string   `json:"text_template"`
	Position     string   `json:"position"`
	Opacity      *float64 `json:"opacity"`
	FontSize     *int     `json:"font_size"`
	Color        string   `json:"color"`
}

var validPositions = map[string]bool{
	"diagonal": true,
	"top":      true,
	"bottom":   true,
	"center":   true,
}

// UpdateConfig handles PUT /api/v1/watermark.
func (h *WatermarkHandler) UpdateConfig(c echo.Context) error {
	var req updateWatermarkRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	config, err := h.wmRepo.GetConfig(c.Request().Context())
	if err != nil {
		return response.InternalError(c)
	}

	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.TextTemplate != "" {
		config.TextTemplate = req.TextTemplate
	}
	if req.Position != "" {
		if !validPositions[req.Position] {
			return response.BadRequest(c, "invalid position (must be diagonal, top, bottom, or center)")
		}
		config.Position = req.Position
	}
	if req.Opacity != nil {
		if *req.Opacity < 0 || *req.Opacity > 1 {
			return response.BadRequest(c, "opacity must be between 0 and 1")
		}
		config.Opacity = *req.Opacity
	}
	if req.FontSize != nil {
		if *req.FontSize < 6 || *req.FontSize > 72 {
			return response.BadRequest(c, "font_size must be between 6 and 72")
		}
		config.FontSize = *req.FontSize
	}
	if req.Color != "" {
		config.Color = req.Color
	}

	if err := h.wmRepo.UpsertConfig(c.Request().Context(), config); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, "watermark.updated", "watermark", "default", "", "")
	return response.OK(c, "Watermark config updated", config)
}

// ResetConfig handles DELETE /api/v1/watermark.
func (h *WatermarkHandler) ResetConfig(c echo.Context) error {
	if err := h.wmRepo.ResetConfig(c.Request().Context()); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, "watermark.reset", "watermark", "default", "", "")

	config, err := h.wmRepo.GetConfig(c.Request().Context())
	if err != nil {
		return response.InternalError(c)
	}
	return response.OK(c, "Watermark config reset to defaults", config)
}

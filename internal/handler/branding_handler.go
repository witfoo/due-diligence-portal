package handler

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/pkg/response"
	"github.com/witfoo/due-diligence-portal/pkg/sanitize"
)

const maxAssetSize = 2 * 1024 * 1024 // 2MB

// BrandingHandler handles branding endpoints.
type BrandingHandler struct {
	brandingRepo repository.BrandingRepository
	audit        *middleware.AuditLogger
}

// NewBrandingHandler creates a new BrandingHandler.
func NewBrandingHandler(brandingRepo repository.BrandingRepository, audit *middleware.AuditLogger) *BrandingHandler {
	return &BrandingHandler{brandingRepo: brandingRepo, audit: audit}
}

// RegisterRoutes registers branding routes on the given group.
func (h *BrandingHandler) RegisterRoutes(g *echo.Group) {
	adminOnly := middleware.RequireRole(domain.RoleAdmin)
	g.GET("/branding", h.GetConfig)
	g.PUT("/branding", h.UpdateConfig, adminOnly)
	g.DELETE("/branding", h.ResetConfig, adminOnly)
	g.GET("/branding/assets/:key", h.GetAsset)
	g.POST("/branding/assets/:key", h.UploadAsset, adminOnly)
	g.DELETE("/branding/assets/:key", h.DeleteAsset, adminOnly)
}

// GetConfig handles GET /branding.
func (h *BrandingHandler) GetConfig(c echo.Context) error {
	config, err := h.brandingRepo.GetConfig(c.Request().Context())
	if err != nil {
		return response.InternalError(c)
	}
	return response.OK(c, "Branding config retrieved", config)
}

// UpdateConfig handles PUT /branding.
func (h *BrandingHandler) UpdateConfig(c echo.Context) error {
	var config domain.BrandingConfig
	if err := c.Bind(&config); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	// Sanitize custom CSS.
	if config.CustomCSS != "" {
		sanitized, _ := sanitize.CSS(config.CustomCSS)
		config.CustomCSS = sanitized
	}

	if err := h.brandingRepo.UpsertConfig(c.Request().Context(), &config); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditBrandingUpdated, "branding", "default", "", "")

	return response.OK(c, "Branding config updated", config)
}

// ResetConfig handles DELETE /branding.
func (h *BrandingHandler) ResetConfig(c echo.Context) error {
	if err := h.brandingRepo.ResetConfig(c.Request().Context()); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditBrandingReset, "branding", "default", "", "")

	return response.OK(c, "Branding config reset to defaults", nil)
}

// GetAsset handles GET /branding/assets/:key.
func (h *BrandingHandler) GetAsset(c echo.Context) error {
	key := c.Param("key")

	asset, err := h.brandingRepo.GetAsset(c.Request().Context(), key)
	if err != nil {
		if err == domain.ErrDocumentNotFound {
			return response.NotFound(c, "asset not found")
		}
		return response.InternalError(c)
	}

	return c.Blob(http.StatusOK, asset.MimeType, asset.FileData)
}

// UploadAsset handles POST /branding/assets/:key.
func (h *BrandingHandler) UploadAsset(c echo.Context) error {
	key := c.Param("key")

	if !domain.ValidAssetKeys[key] {
		return response.BadRequest(c, "invalid asset key")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "file is required")
	}

	if file.Size > maxAssetSize {
		return response.TooLarge(c, "file exceeds 2MB limit")
	}

	src, err := file.Open()
	if err != nil {
		return response.InternalError(c)
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return response.InternalError(c)
	}

	// Detect content type.
	mimeType := http.DetectContentType(data)

	// Compute checksum.
	checksum := fmt.Sprintf("%x", sha256.Sum256(data))

	asset := &domain.BrandingAsset{
		Key:            key,
		FileData:       data,
		MimeType:       mimeType,
		FileSize:       int64(len(data)),
		ChecksumSHA256: checksum,
		UploadedBy:     middleware.GetUserID(c),
	}

	if err := h.brandingRepo.UpsertAsset(c.Request().Context(), asset); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditAssetUploaded, "branding_asset", key, key, "")

	return response.Created(c, "Asset uploaded", map[string]any{
		"key":       asset.Key,
		"mime_type": asset.MimeType,
		"file_size": asset.FileSize,
		"checksum":  asset.ChecksumSHA256,
	})
}

// DeleteAsset handles DELETE /branding/assets/:key.
func (h *BrandingHandler) DeleteAsset(c echo.Context) error {
	key := c.Param("key")

	if err := h.brandingRepo.DeleteAsset(c.Request().Context(), key); err != nil {
		if err == domain.ErrDocumentNotFound {
			return response.NotFound(c, "asset not found")
		}
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditAssetDeleted, "branding_asset", key, key, "")

	return response.OK(c, "Asset deleted", nil)
}

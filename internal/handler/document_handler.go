package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/pkg/envconfig"
	"github.com/witfoo/due-diligence-portal/pkg/response"
	"github.com/witfoo/due-diligence-portal/pkg/sanitize"
)

const defaultMaxUploadSize int64 = 100 * 1024 * 1024 // 100MB

// DocumentHandler handles document management endpoints.
type DocumentHandler struct {
	docRepo       repository.DocumentRepository
	permRepo      repository.PermissionRepository
	audit         *middleware.AuditLogger
	maxUploadSize int64
}

// NewDocumentHandler creates a new DocumentHandler.
func NewDocumentHandler(docRepo repository.DocumentRepository, permRepo repository.PermissionRepository, audit *middleware.AuditLogger) *DocumentHandler {
	maxSize := envconfig.GetEnvInt64("DD_MAX_UPLOAD_SIZE", defaultMaxUploadSize)
	return &DocumentHandler{docRepo: docRepo, permRepo: permRepo, audit: audit, maxUploadSize: maxSize}
}

// RegisterRoutes registers document routes on a pre-authenticated group.
func (h *DocumentHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/documents", h.List)
	g.GET("/documents/:id", h.Get)
	g.POST("/documents", h.Upload, middleware.RequireRole(domain.RoleAdmin, domain.RoleCompanyMember))
	g.PUT("/documents/:id", h.Update, middleware.RequireRole(domain.RoleAdmin, domain.RoleCompanyMember))
	g.DELETE("/documents/:id", h.Archive, middleware.RequireRole(domain.RoleAdmin, domain.RoleCompanyMember))
	g.POST("/documents/:id/versions", h.UploadVersion, middleware.RequireRole(domain.RoleAdmin, domain.RoleCompanyMember))
	g.GET("/documents/:id/versions/:version", h.DownloadVersion)
	g.GET("/documents/:id/download", h.Download)
	g.POST("/documents/search", h.Search)
}

// List handles GET /documents.
func (h *DocumentHandler) List(c echo.Context) error {
	categoryID := c.QueryParam("category_id")
	limit := 50
	offset := 0
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	docs, total, err := h.docRepo.List(c.Request().Context(), categoryID, limit, offset)
	if err != nil {
		return response.InternalError(c)
	}

	// Filter for investors: only show documents they have access grants for.
	role := middleware.GetUserRole(c)
	if role == domain.RoleInvestor {
		userID := middleware.GetUserID(c)
		var filtered []*domain.Document
		for _, doc := range docs {
			hasAccess, err := h.permRepo.HasAccess(c.Request().Context(), userID, domain.ResourceDocument, doc.ID, domain.AccessView)
			if err != nil {
				continue
			}
			if hasAccess {
				filtered = append(filtered, doc)
			}
		}
		docs = filtered
		total = len(filtered)
	}

	return response.OKWithMeta(c, "Documents retrieved", docs, &response.Meta{
		Count:    len(docs),
		Total:    total,
		Page:     (offset / limit) + 1,
		PageSize: limit,
		HasMore:  offset+limit < total,
	})
}

// Get handles GET /documents/:id.
func (h *DocumentHandler) Get(c echo.Context) error {
	id := c.Param("id")
	doc, err := h.docRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == domain.ErrDocumentNotFound {
			return response.NotFound(c, "document not found")
		}
		return response.InternalError(c)
	}

	versions, err := h.docRepo.ListVersions(c.Request().Context(), id)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OK(c, "Document retrieved", map[string]any{
		"document": doc,
		"versions": versions,
	})
}

// Upload handles POST /documents (multipart/form-data).
func (h *DocumentHandler) Upload(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "file is required")
	}

	if file.Size > h.maxUploadSize {
		return response.TooLarge(c, fmt.Sprintf("file exceeds maximum size of %d bytes", h.maxUploadSize))
	}

	name := c.FormValue("name")
	if name == "" {
		return response.BadRequest(c, "name is required")
	}
	description := c.FormValue("description")
	categoryID := c.FormValue("category_id")
	if categoryID == "" {
		return response.BadRequest(c, "category_id is required")
	}
	tags := c.FormValue("tags")

	src, err := file.Open()
	if err != nil {
		return response.InternalError(c)
	}
	defer src.Close()

	fileData, err := io.ReadAll(src)
	if err != nil {
		return response.InternalError(c)
	}

	checksum := sha256.Sum256(fileData)
	checksumHex := hex.EncodeToString(checksum[:])
	safeFilename := sanitize.FileName(file.Filename)
	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(fileData)
	}

	docID, err := generateHandlerID()
	if err != nil {
		return response.InternalError(c)
	}
	versionID, err := generateHandlerID()
	if err != nil {
		return response.InternalError(c)
	}

	userID := middleware.GetUserID(c)

	doc := &domain.Document{
		ID:             docID,
		Name:           name,
		Description:    description,
		CategoryID:     categoryID,
		UploadedBy:     userID,
		CurrentVersion: 1,
		MimeType:       mimeType,
		FileSize:       int64(len(fileData)),
		Tags:           tags,
	}

	if err := h.docRepo.Create(c.Request().Context(), doc); err != nil {
		return response.InternalError(c)
	}

	version := &domain.DocumentVersion{
		ID:             versionID,
		DocumentID:     docID,
		VersionNumber:  1,
		FileData:       fileData,
		FileSize:       int64(len(fileData)),
		MimeType:       mimeType,
		ChecksumSHA256: checksumHex,
		ChangeNote:     "Initial upload",
		UploadedBy:     userID,
	}

	if err := h.docRepo.CreateVersion(c.Request().Context(), version); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditDocumentUploaded, "document", docID, safeFilename, "")

	return response.Created(c, "Document uploaded", doc)
}

type updateDocumentRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CategoryID  string `json:"category_id"`
	Tags        string `json:"tags"`
}

// Update handles PUT /documents/:id.
func (h *DocumentHandler) Update(c echo.Context) error {
	id := c.Param("id")
	var req updateDocumentRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	doc, err := h.docRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == domain.ErrDocumentNotFound {
			return response.NotFound(c, "document not found")
		}
		return response.InternalError(c)
	}

	if req.Name != "" {
		doc.Name = req.Name
	}
	if req.Description != "" {
		doc.Description = req.Description
	}
	if req.CategoryID != "" {
		doc.CategoryID = req.CategoryID
	}
	if req.Tags != "" {
		doc.Tags = req.Tags
	}

	if err := h.docRepo.Update(c.Request().Context(), doc); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditDocumentUpdated, "document", id, doc.Name, "")

	return response.OK(c, "Document updated", doc)
}

// Archive handles DELETE /documents/:id (soft delete).
func (h *DocumentHandler) Archive(c echo.Context) error {
	id := c.Param("id")

	if err := h.docRepo.Archive(c.Request().Context(), id); err != nil {
		if err == domain.ErrDocumentNotFound {
			return response.NotFound(c, "document not found")
		}
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditDocumentArchived, "document", id, "", "")

	return response.OK(c, "Document archived", nil)
}

// UploadVersion handles POST /documents/:id/versions.
func (h *DocumentHandler) UploadVersion(c echo.Context) error {
	docID := c.Param("id")

	doc, err := h.docRepo.GetByID(c.Request().Context(), docID)
	if err != nil {
		if err == domain.ErrDocumentNotFound {
			return response.NotFound(c, "document not found")
		}
		return response.InternalError(c)
	}

	file, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "file is required")
	}

	if file.Size > h.maxUploadSize {
		return response.TooLarge(c, fmt.Sprintf("file exceeds maximum size of %d bytes", h.maxUploadSize))
	}

	changeNote := c.FormValue("change_note")

	src, err := file.Open()
	if err != nil {
		return response.InternalError(c)
	}
	defer src.Close()

	fileData, err := io.ReadAll(src)
	if err != nil {
		return response.InternalError(c)
	}

	checksum := sha256.Sum256(fileData)
	checksumHex := hex.EncodeToString(checksum[:])
	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(fileData)
	}

	versionID, err := generateHandlerID()
	if err != nil {
		return response.InternalError(c)
	}

	userID := middleware.GetUserID(c)
	newVersionNum := doc.CurrentVersion + 1

	version := &domain.DocumentVersion{
		ID:             versionID,
		DocumentID:     docID,
		VersionNumber:  newVersionNum,
		FileData:       fileData,
		FileSize:       int64(len(fileData)),
		MimeType:       mimeType,
		ChecksumSHA256: checksumHex,
		ChangeNote:     changeNote,
		UploadedBy:     userID,
	}

	if err := h.docRepo.CreateVersion(c.Request().Context(), version); err != nil {
		return response.InternalError(c)
	}

	// Update document metadata to reflect new version.
	doc.CurrentVersion = newVersionNum
	doc.MimeType = mimeType
	doc.FileSize = int64(len(fileData))
	if err := h.docRepo.Update(c.Request().Context(), doc); err != nil {
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditDocumentNewVersion, "document", docID, doc.Name,
		fmt.Sprintf("version=%d", newVersionNum))

	return response.Created(c, "New version uploaded", version)
}

// DownloadVersion handles GET /documents/:id/versions/:version.
func (h *DocumentHandler) DownloadVersion(c echo.Context) error {
	docID := c.Param("id")
	versionStr := c.Param("version")
	versionNum, err := strconv.Atoi(versionStr)
	if err != nil {
		return response.BadRequest(c, "invalid version number")
	}

	doc, err := h.docRepo.GetByID(c.Request().Context(), docID)
	if err != nil {
		if err == domain.ErrDocumentNotFound {
			return response.NotFound(c, "document not found")
		}
		return response.InternalError(c)
	}

	version, err := h.docRepo.GetVersion(c.Request().Context(), docID, versionNum)
	if err != nil {
		if err == domain.ErrVersionNotFound {
			return response.NotFound(c, "version not found")
		}
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditDocumentDownloaded, "document", docID, doc.Name,
		fmt.Sprintf("version=%d", versionNum))

	c.Response().Header().Set("Content-Type", version.MimeType)
	c.Response().Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%q", sanitize.FileName(doc.Name)))
	return c.Blob(http.StatusOK, version.MimeType, version.FileData)
}

// Download handles GET /documents/:id/download (current version).
func (h *DocumentHandler) Download(c echo.Context) error {
	docID := c.Param("id")

	doc, err := h.docRepo.GetByID(c.Request().Context(), docID)
	if err != nil {
		if err == domain.ErrDocumentNotFound {
			return response.NotFound(c, "document not found")
		}
		return response.InternalError(c)
	}

	version, err := h.docRepo.GetVersion(c.Request().Context(), docID, doc.CurrentVersion)
	if err != nil {
		if err == domain.ErrVersionNotFound {
			return response.NotFound(c, "version not found")
		}
		return response.InternalError(c)
	}

	h.audit.LogFromContext(c, domain.AuditDocumentDownloaded, "document", docID, doc.Name, "")

	c.Response().Header().Set("Content-Type", version.MimeType)
	c.Response().Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%q", sanitize.FileName(doc.Name)))
	return c.Blob(http.StatusOK, version.MimeType, version.FileData)
}

type searchRequest struct {
	Query string `json:"query"`
}

// Search handles POST /documents/search.
func (h *DocumentHandler) Search(c echo.Context) error {
	var req searchRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if req.Query == "" {
		return response.BadRequest(c, "query is required")
	}

	docs, total, err := h.docRepo.Search(c.Request().Context(), req.Query, 50, 0)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OKWithMeta(c, "Search results", docs, &response.Meta{
		Count:    len(docs),
		Total:    total,
		Page:     1,
		PageSize: 50,
		HasMore:  total > 50,
	})
}



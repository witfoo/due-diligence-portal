package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/internal/service"
	"github.com/witfoo/due-diligence-portal/pkg/response"
)

func setupBrandingHandlerTest(t *testing.T) (*echo.Echo, *BrandingHandler, string) {
	t.Helper()
	db, err := repository.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, testJWTSecret)
	audit := middleware.NewAuditLogger(db)
	brandingRepo := repository.NewBrandingRepository(db)
	brandingHandler := NewBrandingHandler(brandingRepo, audit)

	e := echo.New()
	authMW := middleware.JWTAuth(authSvc)
	g := e.Group("/api/v1", authMW)
	brandingHandler.RegisterRoutes(g)

	// Create test admin user and get token.
	require.NoError(t, authSvc.EnsureAdminExists(context.Background(), "admin@test.com", "password123"))
	result, err := authSvc.Login(context.Background(), "admin@test.com", "password123")
	require.NoError(t, err)

	return e, brandingHandler, result.AccessToken
}

func TestBrandingHandler_GetConfig(t *testing.T) {
	e, _, token := setupBrandingHandlerTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/branding", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "Branding config retrieved", resp.Message)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, data["company_name"])
	assert.NotEmpty(t, data["primary_color"])
}

func TestBrandingHandler_UpdateConfig(t *testing.T) {
	e, _, token := setupBrandingHandlerTest(t)

	body := `{"company_name":"Acme Corp","primary_color":"#ff0000","secondary_color":"#00ff00","accent_color":"#0000ff","error_color":"#da1e28","warning_color":"#f1c21b","success_color":"#24a148","info_color":"#4589ff","background_color":"#161616","surface_color":"#262626","text_color":"#f4f4f4","text_secondary_color":"#c6c6c6","border_color":"#393939","hover_color":"#353535","active_color":"#525252","header_color":"#161616","sidebar_color":"#1c1c1c","custom_css":"body { margin: 0; }"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/branding", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	data := resp.Data.(map[string]any)
	assert.Equal(t, "Acme Corp", data["company_name"])
	assert.Equal(t, "#ff0000", data["primary_color"])
}

func TestBrandingHandler_UploadAsset(t *testing.T) {
	e, _, token := setupBrandingHandlerTest(t)

	// Create multipart form with a small PNG file.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "logo.png")
	require.NoError(t, err)
	// Write a minimal valid content.
	_, err = part.Write([]byte("fake-png-data-for-testing"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/branding/assets/logo", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	data := resp.Data.(map[string]any)
	assert.Equal(t, "logo", data["key"])
	assert.NotEmpty(t, data["checksum"])
}

func TestBrandingHandler_UploadAsset_InvalidKey(t *testing.T) {
	e, _, token := setupBrandingHandlerTest(t)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("data"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/branding/assets/invalid_key", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBrandingHandler_GetAsset(t *testing.T) {
	e, _, token := setupBrandingHandlerTest(t)

	// Upload first.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "favicon.ico")
	require.NoError(t, err)
	_, err = part.Write([]byte("favicon-data"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/branding/assets/favicon", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Get asset.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/branding/assets/favicon", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "favicon-data", rec.Body.String())
}

package handler

import (
	"context"
	"encoding/json"
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

func setupWatermarkHandlerTest(t *testing.T) (*echo.Echo, string) {
	t.Helper()
	db, err := repository.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, testJWTSecret)
	audit := middleware.NewAuditLogger(db)
	wmRepo := repository.NewWatermarkRepository(db)
	wmHandler := NewWatermarkHandler(wmRepo, audit)

	e := echo.New()
	authMW := middleware.JWTAuth(authSvc)
	g := e.Group("/api/v1", authMW)
	wmHandler.RegisterRoutes(g)

	_, adminErr := authSvc.EnsureAdminExists(context.Background(), "admin@test.com", "password123")
	require.NoError(t, adminErr)
	result, err := authSvc.Login(context.Background(), "admin@test.com", "password123")
	require.NoError(t, err)

	return e, result.AccessToken
}

func TestWatermarkHandler_GetConfig(t *testing.T) {
	e, token := setupWatermarkHandlerTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/watermark", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, data["enabled"])
	assert.Equal(t, "diagonal", data["position"])
}

func TestWatermarkHandler_UpdateConfig(t *testing.T) {
	e, token := setupWatermarkHandlerTest(t)

	body := `{"enabled":true,"text_template":"CONFIDENTIAL - {{user_email}}","position":"bottom","opacity":0.3,"font_size":16,"color":"#ff0000"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/watermark", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, data["enabled"])
	assert.Equal(t, "bottom", data["position"])
	assert.Equal(t, "CONFIDENTIAL - {{user_email}}", data["text_template"])
}

func TestWatermarkHandler_UpdateConfig_InvalidPosition(t *testing.T) {
	e, token := setupWatermarkHandlerTest(t)

	body := `{"position":"left"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/watermark", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWatermarkHandler_UpdateConfig_InvalidOpacity(t *testing.T) {
	e, token := setupWatermarkHandlerTest(t)

	body := `{"opacity":1.5}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/watermark", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWatermarkHandler_ResetConfig(t *testing.T) {
	e, token := setupWatermarkHandlerTest(t)

	// First update.
	body := `{"enabled":true,"position":"top"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/watermark", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Reset.
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/watermark", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, data["enabled"])
	assert.Equal(t, "diagonal", data["position"])
}

func TestWatermarkHandler_Unauthenticated(t *testing.T) {
	e, _ := setupWatermarkHandlerTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/watermark", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

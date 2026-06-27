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

func setupAnalyticsHandlerTest(t *testing.T) (*echo.Echo, *AnalyticsHandler, string) {
	t.Helper()
	db, err := repository.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, testJWTSecret)
	audit := middleware.NewAuditLogger(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	docRepo := repository.NewDocumentRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	analyticsHandler := NewAnalyticsHandler(analyticsRepo, docRepo, permRepo, userRepo, audit)

	e := echo.New()
	authMW := middleware.JWTAuth(authSvc)
	g := e.Group("/api/v1", authMW)
	analyticsHandler.RegisterRoutes(g)

	// Create test admin user and get token.
	_, adminErr := authSvc.EnsureAdminExists(context.Background(), "admin@test.com", "password123")
	require.NoError(t, adminErr)
	result, err := authSvc.Login(context.Background(), "admin@test.com", "password123")
	require.NoError(t, err)

	// Seed a test document for FK constraints.
	uid := result.User.ID
	_, err = db.ExecContext(context.Background(),
		`INSERT INTO documents (id, name, category_id, uploaded_by, current_version, mime_type, file_size) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"doc-123", "Test Doc", "cat-corporate", uid, 1, "text/plain", 100)
	require.NoError(t, err)

	return e, analyticsHandler, result.AccessToken
}

func TestAnalyticsHandler_RecordViewEvent(t *testing.T) {
	e, _, token := setupAnalyticsHandlerTest(t)

	body := `{"document_id":"doc-123","duration_ms":5000,"page_count":3}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/view-event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "View event recorded", resp.Message)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "doc-123", data["document_id"])
}

func TestAnalyticsHandler_RecordViewEvent_MissingDocID(t *testing.T) {
	e, _, token := setupAnalyticsHandlerTest(t)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/view-event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAnalyticsHandler_Dashboard(t *testing.T) {
	e, _, token := setupAnalyticsHandlerTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "Dashboard retrieved", resp.Message)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	// Should have numeric fields.
	assert.NotNil(t, data["total_documents"])
	assert.NotNil(t, data["total_views"])
}

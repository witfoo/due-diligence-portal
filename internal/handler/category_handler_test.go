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

func setupCategoryTest(t *testing.T) (*echo.Echo, string) {
	t.Helper()
	db, err := repository.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, testJWTSecret)
	audit := middleware.NewAuditLogger(db)

	catRepo := repository.NewCategoryRepository(db)
	docRepo := repository.NewDocumentRepository(db)

	require.NoError(t, authSvc.EnsureAdminExists(context.Background(), "admin@test.com", "password123"))

	e := echo.New()
	authMW := middleware.JWTAuth(authSvc)
	g := e.Group("/api/v1", authMW)

	catHandler := NewCategoryHandler(catRepo, docRepo, audit)
	catHandler.RegisterRoutes(g)

	result, err := authSvc.Login(context.Background(), "admin@test.com", "password123")
	require.NoError(t, err)

	return e, result.AccessToken
}

func createCategory(t *testing.T, e *echo.Echo, token, name, slug string) string {
	t.Helper()
	body := `{"name":"` + name + `","slug":"` + slug + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	return data["id"].(string)
}

func TestCategoryHandler_Create(t *testing.T) {
	e, token := setupCategoryTest(t)

	body := `{"name":"Custom Reports","slug":"custom-reports","description":"Custom report documents"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "Category created", resp.Message)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "Custom Reports", data["name"])
	assert.Equal(t, "custom-reports", data["slug"])
}

func TestCategoryHandler_Create_MissingName(t *testing.T) {
	e, token := setupCategoryTest(t)

	body := `{"slug":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCategoryHandler_List(t *testing.T) {
	e, token := setupCategoryTest(t)
	createCategory(t, e, token, "Custom A", "custom-a")
	createCategory(t, e, token, "Custom B", "custom-b")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)

	// Seeded categories (10) + 2 custom = 12 total at root level.
	data, ok := resp.Data.([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(data), 12)
}

func TestCategoryHandler_Update(t *testing.T) {
	e, token := setupCategoryTest(t)
	catID := createCategory(t, e, token, "Custom Legal", "custom-legal")

	body := `{"name":"Legal Docs","slug":"legal-docs"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/categories/"+catID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Legal Docs", data["name"])
}

func TestCategoryHandler_Update_NotFound(t *testing.T) {
	e, token := setupCategoryTest(t)

	body := `{"name":"Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/categories/nonexistent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCategoryHandler_Delete(t *testing.T) {
	e, token := setupCategoryTest(t)
	catID := createCategory(t, e, token, "To Delete", "to-delete")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/categories/"+catID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Category deleted", resp.Message)
}

func TestCategoryHandler_Delete_NotFound(t *testing.T) {
	e, token := setupCategoryTest(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/categories/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCategoryHandler_Unauthenticated(t *testing.T) {
	e, _ := setupCategoryTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

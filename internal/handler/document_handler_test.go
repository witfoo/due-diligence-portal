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

func setupDocumentTest(t *testing.T) (*echo.Echo, *service.AuthService, string) {
	t.Helper()
	db, err := repository.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, testJWTSecret)
	audit := middleware.NewAuditLogger(db)

	docRepo := repository.NewDocumentRepository(db)
	permRepo := repository.NewPermissionRepository(db)

	// Use the seeded "Financials" category (cat-financials) from migration 003.

	// Create admin user.
	require.NoError(t, authSvc.EnsureAdminExists(context.Background(), "admin@test.com", "password123"))

	e := echo.New()
	authMW := middleware.JWTAuth(authSvc)

	// Register document routes on auth group.
	g := e.Group("/api/v1", authMW)
	docHandler := NewDocumentHandler(docRepo, permRepo, audit)
	docHandler.RegisterRoutes(g)

	// Get admin token.
	result, err := authSvc.Login(context.Background(), "admin@test.com", "password123")
	require.NoError(t, err)

	return e, authSvc, result.AccessToken
}

func uploadDocument(t *testing.T, e *echo.Echo, token, catID string) string {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake pdf content"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("name", "Test Document"))
	require.NoError(t, writer.WriteField("category_id", catID))
	require.NoError(t, writer.WriteField("description", "A test document"))
	require.NoError(t, writer.WriteField("tags", "test,finance"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
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

func TestDocumentHandler_Upload(t *testing.T) {
	e, _, token := setupDocumentTest(t)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake pdf content"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("name", "Test Document"))
	require.NoError(t, writer.WriteField("category_id", "cat-financials"))
	require.NoError(t, writer.WriteField("description", "A test document"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "Document uploaded", resp.Message)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "Test Document", data["name"])
	assert.Equal(t, float64(1), data["current_version"])
}

func TestDocumentHandler_Upload_MissingFile(t *testing.T) {
	e, _, token := setupDocumentTest(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents", strings.NewReader(""))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDocumentHandler_GetByID(t *testing.T) {
	e, _, token := setupDocumentTest(t)
	docID := uploadDocument(t, e, token, "cat-financials")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+docID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)

	doc, ok := data["document"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, docID, doc["id"])

	versions, ok := data["versions"].([]any)
	require.True(t, ok)
	assert.Len(t, versions, 1)
}

func TestDocumentHandler_GetByID_NotFound(t *testing.T) {
	e, _, token := setupDocumentTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDocumentHandler_List(t *testing.T) {
	e, _, token := setupDocumentTest(t)
	uploadDocument(t, e, token, "cat-financials")
	uploadDocument(t, e, token, "cat-financials")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Meta)
	assert.Equal(t, 2, resp.Meta.Count)
}

func TestDocumentHandler_List_FilterByCategory(t *testing.T) {
	e, _, token := setupDocumentTest(t)
	uploadDocument(t, e, token, "cat-financials")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents?category_id=cat-financials", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, 1, resp.Meta.Count)
}

func TestDocumentHandler_Search(t *testing.T) {
	e, _, token := setupDocumentTest(t)
	uploadDocument(t, e, token, "cat-financials")

	body := `{"query":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
}

func TestDocumentHandler_Search_EmptyQuery(t *testing.T) {
	e, _, token := setupDocumentTest(t)

	body := `{"query":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDocumentHandler_Download(t *testing.T) {
	e, _, token := setupDocumentTest(t)
	docID := uploadDocument(t, e, token, "cat-financials")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+docID+"/download", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment")
	assert.Equal(t, "fake pdf content", rec.Body.String())
}

func TestDocumentHandler_DownloadVersion(t *testing.T) {
	e, _, token := setupDocumentTest(t)
	docID := uploadDocument(t, e, token, "cat-financials")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+docID+"/versions/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment")
	assert.Equal(t, "fake pdf content", rec.Body.String())
}

func TestDocumentHandler_Archive(t *testing.T) {
	e, _, token := setupDocumentTest(t)
	docID := uploadDocument(t, e, token, "cat-financials")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/documents/"+docID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Document archived", resp.Message)
}

func TestDocumentHandler_Archive_NotFound(t *testing.T) {
	e, _, token := setupDocumentTest(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/documents/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDocumentHandler_Update(t *testing.T) {
	e, _, token := setupDocumentTest(t)
	docID := uploadDocument(t, e, token, "cat-financials")

	body := `{"name":"Updated Name","description":"Updated desc"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/documents/"+docID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Updated Name", data["name"])
}

func TestDocumentHandler_Unauthenticated(t *testing.T) {
	e, _, _ := setupDocumentTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

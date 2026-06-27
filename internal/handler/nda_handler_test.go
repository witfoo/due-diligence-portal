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

func setupNDAHandlerTest(t *testing.T) (*echo.Echo, *NDAHandler, string) {
	t.Helper()
	db, err := repository.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, testJWTSecret)
	audit := middleware.NewAuditLogger(db)
	ndaRepo := repository.NewNDARepository(db)
	ndaHandler := NewNDAHandler(ndaRepo, service.NewEmailService(), "admin@test.com", audit)

	e := echo.New()
	authMW := middleware.JWTAuth(authSvc)
	g := e.Group("/api/v1", authMW)
	ndaHandler.RegisterRoutes(g)

	// Create test admin user and get token.
	_, adminErr := authSvc.EnsureAdminExists(context.Background(), "admin@test.com", "password123")
	require.NoError(t, adminErr)
	result, err := authSvc.Login(context.Background(), "admin@test.com", "password123")
	require.NoError(t, err)

	return e, ndaHandler, result.AccessToken
}

func TestNDAHandler_CreateTemplate(t *testing.T) {
	e, _, token := setupNDAHandlerTest(t)

	body := `{"name":"Standard NDA","content":"This is the NDA content..."}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nda/templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "Template created", resp.Message)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Standard NDA", data["name"])
	assert.Equal(t, true, data["is_active"])
}

func TestNDAHandler_CreateTemplate_MissingFields(t *testing.T) {
	e, _, token := setupNDAHandlerTest(t)

	body := `{"name":"Incomplete"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nda/templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestNDAHandler_SignNDA(t *testing.T) {
	e, _, token := setupNDAHandlerTest(t)

	// Create template first.
	body := `{"name":"Test NDA","content":"NDA content here"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nda/templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var createResp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	templateID := createResp.Data.(map[string]any)["id"].(string)

	// Sign the NDA.
	body = `{"signer_name":"John Doe","signer_company":"Acme Inc"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/nda/sign/"+templateID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "NDA signed", resp.Message)

	data := resp.Data.(map[string]any)
	// signer_name is bound to the authenticated account (the seeded admin), not the
	// client-submitted "John Doe", so the signature cannot be attributed to a third party.
	assert.Equal(t, "Administrator", data["signer_name"])
	assert.Equal(t, templateID, data["template_id"])
}

func TestNDAHandler_SignNDA_AlreadySigned(t *testing.T) {
	e, _, token := setupNDAHandlerTest(t)

	// Create template.
	body := `{"name":"Test NDA","content":"NDA content"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nda/templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var createResp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	templateID := createResp.Data.(map[string]any)["id"].(string)

	// Sign once.
	body = `{"signer_name":"John Doe"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/nda/sign/"+templateID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Try to sign again.
	body = `{"signer_name":"John Doe"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/nda/sign/"+templateID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestNDAHandler_CheckStatus(t *testing.T) {
	e, _, token := setupNDAHandlerTest(t)

	// Initially should not be signed.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nda/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data := resp.Data.(map[string]any)
	assert.Equal(t, false, data["signed"])

	// Create template and sign it.
	body := `{"name":"Active NDA","content":"Content"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/nda/templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var createResp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	templateID := createResp.Data.(map[string]any)["id"].(string)

	body = `{"signer_name":"Admin User"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/nda/sign/"+templateID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Now check status - should be signed.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/nda/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data = resp.Data.(map[string]any)
	assert.Equal(t, true, data["signed"])
}

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

func setupNDAHandlerTest(t *testing.T) (*echo.Echo, *NDAHandler, string, string) {
	t.Helper()
	db, err := repository.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, testJWTSecret)
	audit := middleware.NewAuditLogger(db)
	ndaRepo := repository.NewNDARepository(db)
	ndaHandler := NewNDAHandler(ndaRepo, userRepo, service.NewEmailService(), "admin@test.com", audit)

	e := echo.New()
	authMW := middleware.JWTAuth(authSvc)
	g := e.Group("/api/v1", authMW)
	ndaHandler.RegisterRoutes(g)

	// Create test admin user and get token.
	_, adminErr := authSvc.EnsureAdminExists(context.Background(), "admin@test.com", "password123")
	require.NoError(t, adminErr)
	result, err := authSvc.Login(context.Background(), "admin@test.com", "password123")
	require.NoError(t, err)

	return e, ndaHandler, result.AccessToken, result.User.ID
}

func TestNDAHandler_CreateTemplate(t *testing.T) {
	e, _, token, _ := setupNDAHandlerTest(t)

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
	e, _, token, _ := setupNDAHandlerTest(t)

	body := `{"name":"Incomplete"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nda/templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestNDAHandler_SignNDA(t *testing.T) {
	e, _, token, _ := setupNDAHandlerTest(t)

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
	e, _, token, _ := setupNDAHandlerTest(t)

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
	e, _, token, _ := setupNDAHandlerTest(t)

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

func grantExemptionRequest(t *testing.T, userID, reason string, fileContent []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if userID != "" {
		require.NoError(t, mw.WriteField("user_id", userID))
	}
	if reason != "" {
		require.NoError(t, mw.WriteField("reason", reason))
	}
	if fileContent != nil {
		fw, err := mw.CreateFormFile("file", "signed-nda.pdf")
		require.NoError(t, err)
		_, err = fw.Write(fileContent)
		require.NoError(t, err)
	}
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/nda/exemptions", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestNDAHandler_GrantExemption_WithDocument(t *testing.T) {
	e, _, token, adminID := setupNDAHandlerTest(t)

	fileContent := []byte("%PDF-1.4 externally executed nda")
	req := grantExemptionRequest(t, adminID, "Executed externally", fileContent)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	// NDA status is now satisfied by the exemption.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/nda/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data := resp.Data.(map[string]any)
	assert.Equal(t, true, data["signed"])
	assert.Equal(t, true, data["exempt"])

	// The exemption is listed with its document flag but without file bytes.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/nda/exemptions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	list := resp.Data.([]any)
	require.Len(t, list, 1)
	ex := list[0].(map[string]any)
	assert.Equal(t, adminID, ex["user_id"])
	assert.Equal(t, "Executed externally", ex["reason"])
	assert.Equal(t, true, ex["has_document"])
	assert.Equal(t, "signed-nda.pdf", ex["file_name"])

	// The uploaded document downloads back byte-for-byte.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/nda/exemptions/"+adminID+"/document", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, fileContent, rec.Body.Bytes())
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "signed-nda.pdf")
}

func TestNDAHandler_GrantExemption_MissingUserID(t *testing.T) {
	e, _, token, _ := setupNDAHandlerTest(t)

	req := grantExemptionRequest(t, "", "reason", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestNDAHandler_GrantExemption_UserNotFound(t *testing.T) {
	e, _, token, _ := setupNDAHandlerTest(t)

	req := grantExemptionRequest(t, "no-such-user", "reason", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestNDAHandler_RevokeExemption(t *testing.T) {
	e, _, token, adminID := setupNDAHandlerTest(t)

	req := grantExemptionRequest(t, adminID, "waived", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Revoke it.
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/nda/exemptions/"+adminID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Status no longer reports exempt (no active template, so unsigned).
	req = httptest.NewRequest(http.MethodGet, "/api/v1/nda/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data := resp.Data.(map[string]any)
	assert.Equal(t, false, data["signed"])

	// Revoking again reports not found.
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/nda/exemptions/"+adminID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestNDAHandler_ExemptionDocument_NoneAttached(t *testing.T) {
	e, _, token, adminID := setupNDAHandlerTest(t)

	req := grantExemptionRequest(t, adminID, "waived without document", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/nda/exemptions/"+adminID+"/document", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

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

func setupQAHandlerTest(t *testing.T) (*echo.Echo, *QAHandler, string) {
	t.Helper()
	db, err := repository.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, testJWTSecret)
	audit := middleware.NewAuditLogger(db)
	qaRepo := repository.NewQARepository(db)
	qaHandler := NewQAHandler(qaRepo, audit)

	e := echo.New()
	authMW := middleware.JWTAuth(authSvc)
	g := e.Group("/api/v1", authMW)
	qaHandler.RegisterRoutes(g)

	// Create test admin user and get token.
	require.NoError(t, authSvc.EnsureAdminExists(context.Background(), "admin@test.com", "password123"))
	result, err := authSvc.Login(context.Background(), "admin@test.com", "password123")
	require.NoError(t, err)

	return e, qaHandler, result.AccessToken
}

func TestQAHandler_CreateThread(t *testing.T) {
	e, _, token := setupQAHandlerTest(t)

	body := `{"subject":"How does revenue recognition work?"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/qa", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "Thread created", resp.Message)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "How does revenue recognition work?", data["subject"])
	assert.Equal(t, "open", data["status"])
}

func TestQAHandler_CreateThread_MissingSubject(t *testing.T) {
	e, _, token := setupQAHandlerTest(t)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/qa", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQAHandler_GetThread(t *testing.T) {
	e, _, token := setupQAHandlerTest(t)

	// Create a thread first.
	body := `{"subject":"Test thread"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/qa", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var createResp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	data := createResp.Data.(map[string]any)
	threadID := data["id"].(string)

	// Get thread.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/qa/"+threadID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	respData := resp.Data.(map[string]any)
	thread := respData["thread"].(map[string]any)
	assert.Equal(t, "Test thread", thread["subject"])
}

func TestQAHandler_PostMessage(t *testing.T) {
	e, _, token := setupQAHandlerTest(t)

	// Create a thread.
	body := `{"subject":"Message test thread"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/qa", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var createResp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	threadID := createResp.Data.(map[string]any)["id"].(string)

	// Post message.
	body = `{"body":"This is a test message"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/qa/"+threadID+"/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	msgData := resp.Data.(map[string]any)
	assert.Equal(t, "This is a test message", msgData["body"])
}

func TestQAHandler_ChangeStatus(t *testing.T) {
	e, _, token := setupQAHandlerTest(t)

	// Create a thread.
	body := `{"subject":"Status change thread"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/qa", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var createResp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	threadID := createResp.Data.(map[string]any)["id"].(string)

	// Change status.
	body = `{"status":"answered"}`
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/qa/"+threadID+"/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	respData := resp.Data.(map[string]any)
	assert.Equal(t, "answered", respData["status"])
}

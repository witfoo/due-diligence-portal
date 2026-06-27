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

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/internal/service"
	"github.com/witfoo/due-diligence-portal/pkg/response"
)

func setupPermissionTest(t *testing.T) (*echo.Echo, string, string) {
	t.Helper()
	db, err := repository.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, testJWTSecret)
	audit := middleware.NewAuditLogger(db)

	permRepo := repository.NewPermissionRepository(db)

	_, adminErr := authSvc.EnsureAdminExists(context.Background(), "admin@test.com", "password123")
	require.NoError(t, adminErr)

	// Create an investor user via invite to use as target.
	result, err := authSvc.Login(context.Background(), "admin@test.com", "password123")
	require.NoError(t, err)
	adminClaims, err := authSvc.ValidateToken(result.AccessToken)
	require.NoError(t, err)

	invite, err := authSvc.CreateInvite(context.Background(), "investor@test.com", domain.RoleInvestor, adminClaims.UserID)
	require.NoError(t, err)
	investorResult, err := authSvc.Register(context.Background(), invite.Token, "Investor User", "password123")
	require.NoError(t, err)
	investorID := investorResult.User.ID

	e := echo.New()
	authMW := middleware.JWTAuth(authSvc)
	g := e.Group("/api/v1", authMW, middleware.RequireRole(domain.RoleAdmin))

	permHandler := NewPermissionHandler(permRepo, audit)
	permHandler.RegisterRoutes(g)

	return e, result.AccessToken, investorID
}

func grantPermission(t *testing.T, e *echo.Echo, token, userID, resourceType, resourceID, accessLevel string) string {
	t.Helper()
	body := `{"user_id":"` + userID + `","resource_type":"` + resourceType + `","resource_id":"` + resourceID + `","access_level":"` + accessLevel + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/permissions", strings.NewReader(body))
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

func TestPermissionHandler_Grant(t *testing.T) {
	e, token, investorID := setupPermissionTest(t)

	body := `{"user_id":"` + investorID + `","resource_type":"document","resource_id":"doc-001","access_level":"download"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/permissions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "Access granted", resp.Message)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, investorID, data["user_id"])
	assert.Equal(t, "download", data["access_level"])
}

func TestPermissionHandler_Grant_MissingFields(t *testing.T) {
	e, token, _ := setupPermissionTest(t)

	body := `{"user_id":"some-user"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/permissions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPermissionHandler_Grant_InvalidResourceType(t *testing.T) {
	e, token, investorID := setupPermissionTest(t)

	body := `{"user_id":"` + investorID + `","resource_type":"invalid","resource_id":"x","access_level":"view"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/permissions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPermissionHandler_ListByDocument(t *testing.T) {
	e, token, investorID := setupPermissionTest(t)
	grantPermission(t, e, token, investorID, "document", "doc-001", "download")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/permissions/document/doc-001", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)

	data, ok := resp.Data.([]any)
	require.True(t, ok)
	assert.Len(t, data, 1)
}

func TestPermissionHandler_ListByCategory(t *testing.T) {
	e, token, investorID := setupPermissionTest(t)
	grantPermission(t, e, token, investorID, "category", "cat-001", "view")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/permissions/category/cat-001", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.([]any)
	require.True(t, ok)
	assert.Len(t, data, 1)
}

func TestPermissionHandler_Update(t *testing.T) {
	e, token, investorID := setupPermissionTest(t)
	grantID := grantPermission(t, e, token, investorID, "document", "doc-001", "view")

	body := `{"access_level":"manage"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/permissions/"+grantID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Access grant updated", resp.Message)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "manage", data["access_level"])
}

func TestPermissionHandler_Update_NotFound(t *testing.T) {
	e, token, _ := setupPermissionTest(t)

	body := `{"access_level":"manage"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/permissions/nonexistent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPermissionHandler_Revoke(t *testing.T) {
	e, token, investorID := setupPermissionTest(t)
	grantID := grantPermission(t, e, token, investorID, "document", "doc-001", "download")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/permissions/"+grantID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Access revoked", resp.Message)
}

func TestPermissionHandler_Revoke_NotFound(t *testing.T) {
	e, token, _ := setupPermissionTest(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/permissions/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPermissionHandler_Unauthenticated(t *testing.T) {
	e, _, _ := setupPermissionTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/permissions/document/doc-001", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

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

const testJWTSecret = "test-jwt-secret-must-be-32-chars!"

func setupHandlerTest(t *testing.T) (*echo.Echo, *AuthHandler, *service.AuthService) {
	t.Helper()
	db, err := repository.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, testJWTSecret)
	audit := middleware.NewAuditLogger(db)
	authHandler := NewAuthHandler(authSvc, audit)

	e := echo.New()
	authMW := middleware.JWTAuth(authSvc)
	authHandler.RegisterRoutes(e, authMW)

	// Create test admin user.
	require.NoError(t, authSvc.EnsureAdminExists(context.Background(), "admin@test.com", "password123"))

	return e, authHandler, authSvc
}

func TestAuthHandler_Login_Success(t *testing.T) {
	e, _, _ := setupHandlerTest(t)

	body := `{"email":"admin@test.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "Login successful", resp.Message)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["refresh_token"])
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	e, _, _ := setupHandlerTest(t)

	body := `{"email":"admin@test.com","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthHandler_Login_MissingFields(t *testing.T) {
	e, _, _ := setupHandlerTest(t)

	body := `{"email":"admin@test.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Me_Authenticated(t *testing.T) {
	e, _, authSvc := setupHandlerTest(t)

	// Login first.
	result, err := authSvc.Login(context.Background(), "admin@test.com", "password123")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+result.AccessToken)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "admin@test.com", data["email"])
	assert.Equal(t, domain.RoleAdmin, data["role"])
}

func TestAuthHandler_Me_Unauthenticated(t *testing.T) {
	e, _, _ := setupHandlerTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthHandler_Register_Success(t *testing.T) {
	e, _, authSvc := setupHandlerTest(t)

	// Login as admin to get an invite.
	result, err := authSvc.Login(context.Background(), "admin@test.com", "password123")
	require.NoError(t, err)

	claims, err := authSvc.ValidateToken(result.AccessToken)
	require.NoError(t, err)

	invite, err := authSvc.CreateInvite(context.Background(), "new@test.com", domain.RoleInvestor, claims.UserID)
	require.NoError(t, err)

	body := `{"token":"` + invite.Token + `","name":"New User","password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestAuthHandler_Register_ShortPassword(t *testing.T) {
	e, _, _ := setupHandlerTest(t)

	body := `{"token":"abc","name":"User","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Refresh(t *testing.T) {
	e, _, authSvc := setupHandlerTest(t)

	result, err := authSvc.Login(context.Background(), "admin@test.com", "password123")
	require.NoError(t, err)

	body := `{"refresh_token":"` + result.RefreshToken + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/internal/service"
)

const testSecret = "test-jwt-secret-must-be-32-chars!"

func setupAuthMiddlewareTest(t *testing.T) (*service.AuthService, string) {
	t.Helper()
	db, err := repository.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, testSecret)

	hash, _ := service.HashPassword("password")
	user := &domain.User{
		ID: "u1", Email: "test@test.com", Name: "Test", PasswordHash: hash,
		Role: domain.RoleAdmin, IsActive: true,
	}
	require.NoError(t, userRepo.Create(context.Background(), user))

	result, err := authSvc.Login(context.Background(), "test@test.com", "password")
	require.NoError(t, err)
	return authSvc, result.AccessToken
}

func TestJWTAuth_ValidToken(t *testing.T) {
	authSvc, token := setupAuthMiddlewareTest(t)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := JWTAuth(authSvc)(func(c echo.Context) error {
		assert.Equal(t, "u1", GetUserID(c))
		assert.Equal(t, "test@test.com", GetUserEmail(c))
		assert.Equal(t, domain.RoleAdmin, GetUserRole(c))
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	authSvc, _ := setupAuthMiddlewareTest(t)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := JWTAuth(authSvc)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	authSvc, _ := setupAuthMiddlewareTest(t)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := JWTAuth(authSvc)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_BadFormat(t *testing.T) {
	authSvc, _ := setupAuthMiddlewareTest(t)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "NotBearer token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := JWTAuth(authSvc)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireRole_Allowed(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextKeyRole, domain.RoleAdmin)

	handler := RequireRole(domain.RoleAdmin, domain.RoleCompanyMember)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireRole_Forbidden(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextKeyRole, domain.RoleInvestor)

	handler := RequireRole(domain.RoleAdmin)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

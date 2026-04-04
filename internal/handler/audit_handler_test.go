package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func setupAuditHandlerTest(t *testing.T) (*echo.Echo, *AuditHandler, string, *middleware.AuditLogger, string) {
	t.Helper()
	db, err := repository.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, testJWTSecret)
	audit := middleware.NewAuditLogger(db)
	auditRepo := repository.NewAuditRepository(db)
	auditHandler := NewAuditHandler(auditRepo, audit)

	e := echo.New()
	authMW := middleware.JWTAuth(authSvc)
	g := e.Group("/api/v1", authMW)
	auditHandler.RegisterRoutes(g)

	// Create test admin user and get token.
	require.NoError(t, authSvc.EnsureAdminExists(context.Background(), "admin@test.com", "password123"))
	result, err := authSvc.Login(context.Background(), "admin@test.com", "password123")
	require.NoError(t, err)

	return e, auditHandler, result.AccessToken, audit, result.User.ID
}

func TestAuditHandler_List(t *testing.T) {
	e, _, token, audit, adminID := setupAuditHandlerTest(t)

	// Create some audit entries using real user ID (FK constraint).
	ctx := context.Background()
	require.NoError(t, audit.Log(ctx, &domain.AuditEntry{
		UserID:       adminID,
		UserEmail:    "admin@test.com",
		Action:       domain.AuditDocumentViewed,
		ResourceType: "document",
		ResourceID:   "doc1",
		ResourceName: "test.pdf",
	}))
	require.NoError(t, audit.Log(ctx, &domain.AuditEntry{
		UserID:       adminID,
		UserEmail:    "admin@test.com",
		Action:       domain.AuditDocumentDownloaded,
		ResourceType: "document",
		ResourceID:   "doc2",
		ResourceName: "report.pdf",
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "Audit log retrieved", resp.Message)

	// Should have the login entry + 2 we created.
	data, ok := resp.Data.([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(data), 2)
}

func TestAuditHandler_FilterByDocument(t *testing.T) {
	e, _, token, audit, adminID := setupAuditHandlerTest(t)

	ctx := context.Background()
	require.NoError(t, audit.Log(ctx, &domain.AuditEntry{
		UserID:       adminID,
		UserEmail:    "admin@test.com",
		Action:       domain.AuditDocumentViewed,
		ResourceType: "document",
		ResourceID:   "doc-abc",
		ResourceName: "special.pdf",
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/document/doc-abc", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "Document audit trail retrieved", resp.Message)

	data, ok := resp.Data.([]any)
	require.True(t, ok)
	assert.Equal(t, 1, len(data))
}

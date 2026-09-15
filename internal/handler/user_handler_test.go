package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/middleware"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/internal/service"
)

const (
	userTestAdminEmail       = "admin@test.com"
	userTestAdminPassword    = "password123"
	userTestInvestorEmail    = "investor@test.com"
	userTestInvestorPassword = "investorpass1"
)

type userHandlerTestEnv struct {
	e             *echo.Echo
	db            *repository.DB
	userRepo      repository.UserRepository
	authSvc       *service.AuthService
	adminID       string
	adminToken    string
	adminRefresh  string
	investorID    string
	investorToken string
}

// passThrough is a no-op middleware standing in for the password throttle in tests
// that are not about rate limiting.
func passThrough(next echo.HandlerFunc) echo.HandlerFunc { return next }

// setupUserHandlerTest mirrors the production wiring in cmd/main.go: UserHandler
// routes are mounted on a group guarded by JWTAuth + RequireRole(admin).
func setupUserHandlerTest(t *testing.T) *userHandlerTestEnv {
	t.Helper()
	return setupUserHandlerTestWithThrottle(t, passThrough)
}

func setupUserHandlerTestWithThrottle(t *testing.T, passwordThrottle echo.MiddlewareFunc) *userHandlerTestEnv {
	t.Helper()
	ctx := context.Background()

	db, err := repository.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { _ = db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, testJWTSecret)
	audit := middleware.NewAuditLogger(db)

	e := echo.New()
	adminGroup := e.Group("/api/v1", middleware.JWTAuth(authSvc), middleware.RequireRole(domain.RoleAdmin))
	NewUserHandler(userRepo, authSvc, service.NewEmailService(), audit).RegisterRoutes(adminGroup, passwordThrottle)

	_, err = authSvc.EnsureAdminExists(ctx, userTestAdminEmail, userTestAdminPassword)
	require.NoError(t, err)
	admin, err := authSvc.Login(ctx, userTestAdminEmail, userTestAdminPassword)
	require.NoError(t, err)

	hash, err := service.HashPassword(userTestInvestorPassword)
	require.NoError(t, err)
	require.NoError(t, userRepo.Create(ctx, &domain.User{
		ID:           "investor-1",
		Email:        userTestInvestorEmail,
		Name:         "Test Investor",
		PasswordHash: hash,
		Role:         domain.RoleInvestor,
		IsActive:     true,
	}))
	investor, err := authSvc.Login(ctx, userTestInvestorEmail, userTestInvestorPassword)
	require.NoError(t, err)

	return &userHandlerTestEnv{
		e:             e,
		db:            db,
		userRepo:      userRepo,
		authSvc:       authSvc,
		adminID:       admin.User.ID,
		adminToken:    admin.AccessToken,
		adminRefresh:  admin.RefreshToken,
		investorID:    investor.User.ID,
		investorToken: investor.AccessToken,
	}
}

func (env *userHandlerTestEnv) putPassword(t *testing.T, token, userID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+userID+"/password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	env.e.ServeHTTP(rec, req)
	return rec
}

func decodeUserHandlerBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
	return body
}

// assertNoSecrets checks a raw response body for passwords and hash material.
func assertNoSecrets(t *testing.T, raw string, passwords ...string) {
	t.Helper()
	for _, pw := range passwords {
		assert.NotContains(t, raw, pw)
	}
	assert.NotContains(t, raw, "password_hash")
	assert.NotContains(t, raw, "$2a$")
}

func TestUserHandler_SetPassword_OtherUser(t *testing.T) {
	env := setupUserHandlerTest(t)
	ctx := context.Background()
	const newPassword = "N3wInvestorPass!"

	rec := env.putPassword(t, env.adminToken, env.investorID,
		`{"password":"`+newPassword+`","current_password":"ignored-for-other-users"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assertNoSecrets(t, rec.Body.String(), newPassword, "ignored-for-other-users")

	body := decodeUserHandlerBody(t, rec)
	assert.Equal(t, true, body["success"])
	assert.Equal(t, "Password updated", body["message"])

	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, data, "access_token", "tokens are only returned for a self change")
	assert.NotContains(t, data, "refresh_token")

	user, ok := data["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, env.investorID, user["id"])
	assert.Equal(t, userTestInvestorEmail, user["email"])
	assert.Equal(t, domain.RoleInvestor, user["role"])
	assert.Equal(t, true, user["is_active"])

	_, err := env.authSvc.Login(ctx, userTestInvestorEmail, newPassword)
	require.NoError(t, err)
	_, err = env.authSvc.Login(ctx, userTestInvestorEmail, userTestInvestorPassword)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestUserHandler_SetPassword_DisabledUser(t *testing.T) {
	env := setupUserHandlerTest(t)
	ctx := context.Background()
	const newPassword = "DisabledUserPass1"
	require.NoError(t, env.userRepo.Deactivate(ctx, env.investorID))

	rec := env.putPassword(t, env.adminToken, env.investorID, `{"password":"`+newPassword+`"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	data, ok := decodeUserHandlerBody(t, rec)["data"].(map[string]any)
	require.True(t, ok)
	user, ok := data["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, user["is_active"])

	stored, err := env.userRepo.GetByID(ctx, env.investorID)
	require.NoError(t, err)
	assert.False(t, stored.IsActive)
	assert.Equal(t, domain.RoleInvestor, stored.Role)

	// Disabled status is only revealed after a correct password.
	_, err = env.authSvc.Login(ctx, userTestInvestorEmail, newPassword)
	assert.ErrorIs(t, err, domain.ErrAccountDisabled)
}

func TestUserHandler_SetPassword_BadRequest(t *testing.T) {
	env := setupUserHandlerTest(t)

	tests := []struct {
		name      string
		userID    string
		body      string
		wantError string
	}{
		{name: "empty password", userID: env.investorID, body: `{"password":""}`, wantError: "password is required"},
		{name: "missing password", userID: env.investorID, body: `{}`, wantError: "password is required"},
		{name: "too short", userID: env.investorID, body: `{"password":"short12"}`, wantError: "password must be at least 8 characters"},
		{name: "too long", userID: env.investorID, body: `{"password":"` + strings.Repeat("a", 73) + `"}`, wantError: "password must be at most 72 bytes"},
		{name: "malformed JSON", userID: env.investorID, body: `{"password":`, wantError: "invalid request body"},
		{name: "validation runs before user lookup", userID: "no-such-user", body: `{"password":"short"}`, wantError: "password must be at least 8 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := env.putPassword(t, env.adminToken, tt.userID, tt.body)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

			body := decodeUserHandlerBody(t, rec)
			assert.Equal(t, false, body["success"])
			assert.Equal(t, tt.wantError, body["error"])
		})
	}

	// None of the rejected requests changed the password.
	_, err := env.authSvc.Login(context.Background(), userTestInvestorEmail, userTestInvestorPassword)
	assert.NoError(t, err)
}

func TestUserHandler_SetPassword_UnknownUser(t *testing.T) {
	env := setupUserHandlerTest(t)

	rec := env.putPassword(t, env.adminToken, "no-such-user", `{"password":"validpassword1"}`)
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	assert.Equal(t, "user not found", decodeUserHandlerBody(t, rec)["error"])
}

func TestUserHandler_SetPassword_SelfReauthFailures(t *testing.T) {
	env := setupUserHandlerTest(t)
	ctx := context.Background()

	tests := []struct {
		name      string
		body      string
		wantError string
	}{
		{name: "missing current password", body: `{"password":"newadminpass1"}`, wantError: "current password is required"},
		{name: "empty current password", body: `{"password":"newadminpass1","current_password":""}`, wantError: "current password is required"},
		{name: "wrong current password", body: `{"password":"newadminpass1","current_password":"wrongpassword"}`, wantError: "current password is incorrect"},
		{name: "invalid new password", body: `{"password":"short","current_password":"` + userTestAdminPassword + `"}`, wantError: "password must be at least 8 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := env.putPassword(t, env.adminToken, env.adminID, tt.body)
			// Never 401: the UI treats any 401 as session expiry and logs out.
			assert.NotEqual(t, http.StatusUnauthorized, rec.Code)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			assert.Equal(t, tt.wantError, decodeUserHandlerBody(t, rec)["error"])
			assertNoSecrets(t, rec.Body.String(), "newadminpass1", "wrongpassword", userTestAdminPassword)
		})
	}

	// The password and the existing session are untouched.
	_, err := env.authSvc.Login(ctx, userTestAdminEmail, userTestAdminPassword)
	require.NoError(t, err)
	_, err = env.authSvc.RefreshToken(ctx, env.adminRefresh)
	assert.NoError(t, err)
}

func TestUserHandler_SetPassword_SelfSuccess(t *testing.T) {
	env := setupUserHandlerTest(t)
	ctx := context.Background()
	const newPassword = "N3wAdminPass!"

	rec := env.putPassword(t, env.adminToken, env.adminID,
		`{"password":"`+newPassword+`","current_password":"`+userTestAdminPassword+`"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assertNoSecrets(t, rec.Body.String(), newPassword, userTestAdminPassword)

	body := decodeUserHandlerBody(t, rec)
	assert.Equal(t, "Password updated", body["message"])
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	user, ok := data["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, env.adminID, user["id"])
	assert.Equal(t, domain.RoleAdmin, user["role"])

	accessToken, _ := data["access_token"].(string)
	refreshToken, _ := data["refresh_token"].(string)
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)

	claims, err := env.authSvc.ValidateToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, env.adminID, claims.UserID)
	assert.Equal(t, service.TokenTypeAccess, claims.TokenType)

	// The returned refresh token works; the pre-change one is revoked.
	_, err = env.authSvc.RefreshToken(ctx, refreshToken)
	require.NoError(t, err)
	_, err = env.authSvc.RefreshToken(ctx, env.adminRefresh)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)

	// The returned access token authorizes admin endpoints.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	listRec := httptest.NewRecorder()
	env.e.ServeHTTP(listRec, req)
	assert.Equal(t, http.StatusOK, listRec.Code)

	_, err = env.authSvc.Login(ctx, userTestAdminEmail, newPassword)
	require.NoError(t, err)
	_, err = env.authSvc.Login(ctx, userTestAdminEmail, userTestAdminPassword)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestUserHandler_SetPassword_SelfDisabledAccount(t *testing.T) {
	env := setupUserHandlerTest(t)
	ctx := context.Background()
	const newPassword = "DisabledAdminPass1"
	// Another admin deactivated this account; its access token has not expired yet.
	require.NoError(t, env.userRepo.Deactivate(ctx, env.adminID))

	rec := env.putPassword(t, env.adminToken, env.adminID,
		`{"password":"`+newPassword+`","current_password":"`+userTestAdminPassword+`"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	data, ok := decodeUserHandlerBody(t, rec)["data"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, data["access_token"])
	user, ok := data["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, user["is_active"])

	stored, err := env.userRepo.GetByID(ctx, env.adminID)
	require.NoError(t, err)
	assert.False(t, stored.IsActive, "setting a password never re-enables an account")
}

func TestUserHandler_SetPassword_Throttled(t *testing.T) {
	const limit = 2
	env := setupUserHandlerTestWithThrottle(t, middleware.NewRateLimiter(limit, time.Minute).Middleware())
	wrongGuess := `{"password":"newadminpass1","current_password":"wrong-guess"}`

	// Callers rejected by the auth middleware do not use up the budget.
	for i := 0; i < limit+1; i++ {
		rec := env.putPassword(t, env.investorToken, env.adminID, wrongGuess)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	}

	for i := 0; i < limit; i++ {
		rec := env.putPassword(t, env.adminToken, env.adminID, wrongGuess)
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	// Further current_password guesses are refused before they are checked.
	rec := env.putPassword(t, env.adminToken, env.adminID,
		`{"password":"newadminpass1","current_password":"`+userTestAdminPassword+`"}`)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, rec.Body.String())

	_, err := env.authSvc.Login(context.Background(), userTestAdminEmail, userTestAdminPassword)
	assert.NoError(t, err, "the throttled request did not change the password")
}

func TestUserHandler_SetPassword_RequiresAdmin(t *testing.T) {
	env := setupUserHandlerTest(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		token      string
		userID     string
		wantStatus int
	}{
		{name: "investor targeting an admin", token: env.investorToken, userID: env.adminID, wantStatus: http.StatusForbidden},
		{name: "investor targeting self", token: env.investorToken, userID: env.investorID, wantStatus: http.StatusForbidden},
		{name: "unauthenticated", token: "", userID: env.investorID, wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := env.putPassword(t, tt.token, tt.userID,
				`{"password":"HijackAttempt123","current_password":"`+userTestInvestorPassword+`"}`)
			assert.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
		})
	}

	_, err := env.authSvc.Login(ctx, userTestAdminEmail, userTestAdminPassword)
	require.NoError(t, err)
	_, err = env.authSvc.Login(ctx, userTestInvestorEmail, userTestInvestorPassword)
	assert.NoError(t, err)
}

func TestUserHandler_SetPassword_AuditLog(t *testing.T) {
	env := setupUserHandlerTest(t)
	ctx := context.Background()
	const (
		otherPassword = "Audit0therPass!"
		selfPassword  = "AuditSelfPass1!"
	)

	rec := env.putPassword(t, env.adminToken, env.investorID, `{"password":"`+otherPassword+`"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = env.putPassword(t, env.adminToken, env.adminID,
		`{"password":"`+selfPassword+`","current_password":"`+userTestAdminPassword+`"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	type auditRow struct {
		userID, resourceType, resourceName, details string
	}
	rows, err := env.db.QueryContext(ctx,
		`SELECT user_id, resource_type, resource_id, resource_name, details
		 FROM audit_log WHERE action = ?`, domain.AuditUserPasswordSet)
	require.NoError(t, err)
	got := map[string]auditRow{}
	for rows.Next() {
		var r auditRow
		var resourceID string
		require.NoError(t, rows.Scan(&r.userID, &r.resourceType, &resourceID, &r.resourceName, &r.details))
		got[resourceID] = r
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())

	require.Len(t, got, 2)
	assert.Equal(t, auditRow{userID: env.adminID, resourceType: "user", resourceName: userTestInvestorEmail, details: "set by administrator"}, got[env.investorID])
	assert.Equal(t, auditRow{userID: env.adminID, resourceType: "user", resourceName: userTestAdminEmail, details: "changed own password"}, got[env.adminID])

	// No column of any audit row contains a password (new or current).
	allRows, err := env.db.QueryContext(ctx,
		`SELECT COALESCE(user_email, '') || '|' || action || '|' || COALESCE(resource_type, '') || '|' ||
		        COALESCE(resource_id, '') || '|' || COALESCE(resource_name, '') || '|' ||
		        COALESCE(details, '') || '|' || COALESCE(user_agent, '')
		 FROM audit_log`)
	require.NoError(t, err)
	defer func() { _ = allRows.Close() }()
	count := 0
	for allRows.Next() {
		var line string
		require.NoError(t, allRows.Scan(&line))
		for _, pw := range []string{otherPassword, selfPassword, userTestAdminPassword} {
			assert.NotContains(t, line, pw)
		}
		count++
	}
	require.NoError(t, allRows.Err())
	assert.Positive(t, count)
}

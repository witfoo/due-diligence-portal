package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/repository"
)

const testJWTSecret = "test-secret-at-least-32-characters-long"

func setupAuthTest(t *testing.T) (*AuthService, repository.UserRepository) {
	t.Helper()
	db, err := repository.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	authSvc := NewAuthService(userRepo, testJWTSecret)
	return authSvc, userRepo
}

func createTestUser(t *testing.T, repo repository.UserRepository, email, password, role string) *domain.User {
	t.Helper()
	hash, err := HashPassword(password)
	require.NoError(t, err)
	id, err := generateID()
	require.NoError(t, err)
	user := &domain.User{
		ID:           id,
		Email:        email,
		Name:         "Test User",
		PasswordHash: hash,
		Role:         role,
		IsActive:     true,
	}
	require.NoError(t, repo.Create(context.Background(), user))
	return user
}

func TestAuthService_Login_Success(t *testing.T) {
	svc, repo := setupAuthTest(t)
	createTestUser(t, repo, "admin@test.com", "password123", domain.RoleAdmin)

	result, err := svc.Login(context.Background(), "admin@test.com", "password123")
	require.NoError(t, err)
	assert.Equal(t, "admin@test.com", result.User.Email)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	svc, repo := setupAuthTest(t)
	createTestUser(t, repo, "user@test.com", "correct", domain.RoleInvestor)

	_, err := svc.Login(context.Background(), "user@test.com", "wrong")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	svc, _ := setupAuthTest(t)

	_, err := svc.Login(context.Background(), "nobody@test.com", "password")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_Login_DisabledAccount(t *testing.T) {
	svc, repo := setupAuthTest(t)
	user := createTestUser(t, repo, "disabled@test.com", "password", domain.RoleInvestor)
	require.NoError(t, repo.Deactivate(context.Background(), user.ID))

	_, err := svc.Login(context.Background(), "disabled@test.com", "password")
	assert.ErrorIs(t, err, domain.ErrAccountDisabled)
}

func TestAuthService_ValidateToken(t *testing.T) {
	svc, repo := setupAuthTest(t)
	createTestUser(t, repo, "jwt@test.com", "pass", domain.RoleAdmin)

	result, err := svc.Login(context.Background(), "jwt@test.com", "pass")
	require.NoError(t, err)

	claims, err := svc.ValidateToken(result.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "jwt@test.com", claims.Email)
	assert.Equal(t, domain.RoleAdmin, claims.Role)
}

func TestAuthService_ValidateToken_Invalid(t *testing.T) {
	svc, _ := setupAuthTest(t)

	_, err := svc.ValidateToken("invalid-token")
	assert.Error(t, err)
}

func TestAuthService_ValidateToken_WrongSecret(t *testing.T) {
	svc1, repo := setupAuthTest(t)
	createTestUser(t, repo, "cross@test.com", "pass", domain.RoleAdmin)

	result, err := svc1.Login(context.Background(), "cross@test.com", "pass")
	require.NoError(t, err)

	// Validate with different secret.
	svc2 := NewAuthService(repo, "different-secret-32-chars-minimum!")
	_, err = svc2.ValidateToken(result.AccessToken)
	assert.Error(t, err)
}

func TestAuthService_RefreshToken(t *testing.T) {
	svc, repo := setupAuthTest(t)
	createTestUser(t, repo, "refresh@test.com", "pass", domain.RoleCompanyMember)

	result, err := svc.Login(context.Background(), "refresh@test.com", "pass")
	require.NoError(t, err)

	newToken, err := svc.RefreshToken(context.Background(), result.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newToken)

	// New token should be valid.
	claims, err := svc.ValidateToken(newToken)
	require.NoError(t, err)
	assert.Equal(t, "refresh@test.com", claims.Email)
}

func TestAuthService_Register(t *testing.T) {
	svc, repo := setupAuthTest(t)
	admin := createTestUser(t, repo, "admin@test.com", "pass", domain.RoleAdmin)

	// Create invite.
	invite, err := svc.CreateInvite(context.Background(), "new@test.com", domain.RoleInvestor, admin.ID)
	require.NoError(t, err)

	// Register with the invite.
	result, err := svc.Register(context.Background(), invite.Token, "New User", "newpassword")
	require.NoError(t, err)
	assert.Equal(t, "new@test.com", result.User.Email)
	assert.Equal(t, domain.RoleInvestor, result.User.Role)
	assert.NotEmpty(t, result.AccessToken)
}

func TestAuthService_Register_UsedToken(t *testing.T) {
	svc, repo := setupAuthTest(t)
	admin := createTestUser(t, repo, "admin@test.com", "pass", domain.RoleAdmin)

	invite, err := svc.CreateInvite(context.Background(), "dup@test.com", domain.RoleInvestor, admin.ID)
	require.NoError(t, err)

	_, err = svc.Register(context.Background(), invite.Token, "User1", "pass1234")
	require.NoError(t, err)

	_, err = svc.Register(context.Background(), invite.Token, "User2", "pass1234")
	assert.ErrorIs(t, err, domain.ErrTokenUsed)
}

func TestAuthService_Register_ExpiredToken(t *testing.T) {
	svc, repo := setupAuthTest(t)
	admin := createTestUser(t, repo, "admin@test.com", "pass", domain.RoleAdmin)

	id, _ := generateID()
	tok, _ := generateToken()
	expired := &domain.InviteToken{
		ID:        id,
		Token:     tok,
		Email:     "expired@test.com",
		Role:      domain.RoleInvestor,
		InvitedBy: admin.ID,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired.
	}
	require.NoError(t, repo.CreateInviteToken(context.Background(), expired))

	_, err := svc.Register(context.Background(), tok, "Expired", "pass1234")
	assert.ErrorIs(t, err, domain.ErrTokenExpired)
}

func TestAuthService_CreateInvite_InvalidEmail(t *testing.T) {
	svc, _ := setupAuthTest(t)
	_, err := svc.CreateInvite(context.Background(), "not-an-email", domain.RoleInvestor, "admin1")
	assert.ErrorIs(t, err, domain.ErrInvalidEmail)
}

func TestAuthService_CreateInvite_InvalidRole(t *testing.T) {
	svc, _ := setupAuthTest(t)
	_, err := svc.CreateInvite(context.Background(), "valid@test.com", "superadmin", "admin1")
	assert.ErrorIs(t, err, domain.ErrInvalidRole)
}

func TestAuthService_EnsureAdminExists(t *testing.T) {
	svc, repo := setupAuthTest(t)
	ctx := context.Background()

	_, err := svc.EnsureAdminExists(ctx, "first@test.com", "adminpass")
	require.NoError(t, err)

	count, _ := repo.Count(ctx)
	assert.Equal(t, 1, count)

	// Second call should be a no-op.
	_, err = svc.EnsureAdminExists(ctx, "second@test.com", "pass2")
	require.NoError(t, err)

	count, _ = repo.Count(ctx)
	assert.Equal(t, 1, count) // Still 1.
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("testpassword")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "testpassword", hash)
}

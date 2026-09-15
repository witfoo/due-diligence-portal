package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

func TestAuthService_SetPassword(t *testing.T) {
	svc, repo := setupAuthTest(t)
	ctx := context.Background()
	user := createTestUser(t, repo, "target@test.com", "oldpassword", domain.RoleInvestor)

	updated, err := svc.SetPassword(ctx, user.ID, "newpassword1")
	require.NoError(t, err)
	assert.Equal(t, user.ID, updated.ID)
	assert.NotEqual(t, user.PasswordHash, updated.PasswordHash)
	assert.Equal(t, domain.RoleInvestor, updated.Role)
	assert.True(t, updated.IsActive)

	stored, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, updated.PasswordHash, stored.PasswordHash, "returned user carries the stored hash")

	_, err = svc.Login(ctx, "target@test.com", "newpassword1")
	require.NoError(t, err)

	_, err = svc.Login(ctx, "target@test.com", "oldpassword")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_SetPassword_DisabledUserStaysDisabled(t *testing.T) {
	svc, repo := setupAuthTest(t)
	ctx := context.Background()
	user := createTestUser(t, repo, "disabled@test.com", "oldpassword", domain.RoleCompanyMember)
	require.NoError(t, repo.Deactivate(ctx, user.ID))

	_, err := svc.SetPassword(ctx, user.ID, "newpassword1")
	require.NoError(t, err)

	stored, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.False(t, stored.IsActive)
	assert.Equal(t, domain.RoleCompanyMember, stored.Role)

	// ErrAccountDisabled is only returned after a correct password, which proves
	// the new hash was stored while the account stayed disabled.
	_, err = svc.Login(ctx, "disabled@test.com", "newpassword1")
	assert.ErrorIs(t, err, domain.ErrAccountDisabled)
}

func TestAuthService_SetPassword_Errors(t *testing.T) {
	svc, repo := setupAuthTest(t)
	ctx := context.Background()
	user := createTestUser(t, repo, "errors@test.com", "oldpassword", domain.RoleInvestor)

	tests := []struct {
		name     string
		userID   string
		password string
		wantErr  error
	}{
		{name: "empty password", userID: user.ID, password: "", wantErr: domain.ErrPasswordRequired},
		{name: "too short", userID: user.ID, password: "short12", wantErr: domain.ErrPasswordTooShort},
		{name: "too long", userID: user.ID, password: strings.Repeat("a", 73), wantErr: domain.ErrPasswordTooLong},
		{name: "validation runs before lookup", userID: "no-such-user", password: "short", wantErr: domain.ErrPasswordTooShort},
		{name: "unknown user", userID: "no-such-user", password: "newpassword1", wantErr: domain.ErrUserNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.SetPassword(ctx, tt.userID, tt.password)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, got)
		})
	}

	// None of the failures replaced the original password.
	_, err := svc.Login(ctx, "errors@test.com", "oldpassword")
	assert.NoError(t, err)
}

func TestAuthService_ChangeOwnPassword(t *testing.T) {
	svc, repo := setupAuthTest(t)
	ctx := context.Background()
	user := createTestUser(t, repo, "self@test.com", "oldpassword", domain.RoleAdmin)

	before, err := svc.Login(ctx, "self@test.com", "oldpassword")
	require.NoError(t, err)

	result, err := svc.ChangeOwnPassword(ctx, user.ID, "oldpassword", "newpassword1")
	require.NoError(t, err)
	assert.Equal(t, user.ID, result.User.ID)

	access, err := svc.ValidateToken(result.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, user.ID, access.UserID)
	assert.Equal(t, TokenTypeAccess, access.TokenType)

	refresh, err := svc.ValidateToken(result.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, TokenTypeRefresh, refresh.TokenType)

	// The new refresh token works...
	newAccess, err := svc.RefreshToken(ctx, result.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newAccess)

	// ...and the one issued before the change is revoked.
	_, err = svc.RefreshToken(ctx, before.RefreshToken)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)

	_, err = svc.Login(ctx, "self@test.com", "newpassword1")
	require.NoError(t, err)
	_, err = svc.Login(ctx, "self@test.com", "oldpassword")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_ChangeOwnPassword_Errors(t *testing.T) {
	svc, repo := setupAuthTest(t)
	ctx := context.Background()
	user := createTestUser(t, repo, "selferr@test.com", "oldpassword", domain.RoleAdmin)

	tests := []struct {
		name        string
		userID      string
		current     string
		newPassword string
		wantErr     error
	}{
		{name: "new password validated first", userID: user.ID, current: "", newPassword: "short", wantErr: domain.ErrPasswordTooShort},
		{name: "new password too long", userID: user.ID, current: "oldpassword", newPassword: strings.Repeat("a", 73), wantErr: domain.ErrPasswordTooLong},
		{name: "missing current password", userID: user.ID, current: "", newPassword: "newpassword1", wantErr: domain.ErrCurrentPasswordRequired},
		{name: "wrong current password", userID: user.ID, current: "wrongpassword", newPassword: "newpassword1", wantErr: domain.ErrCurrentPasswordIncorrect},
		{name: "unknown user", userID: "no-such-user", current: "oldpassword", newPassword: "newpassword1", wantErr: domain.ErrUserNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.ChangeOwnPassword(ctx, tt.userID, tt.current, tt.newPassword)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, got)
		})
	}

	// None of the failures changed the stored password.
	_, err := svc.Login(ctx, "selferr@test.com", "oldpassword")
	assert.NoError(t, err)
}

func TestAuthService_ChangeOwnPassword_DisabledAccount(t *testing.T) {
	svc, repo := setupAuthTest(t)
	ctx := context.Background()
	user := createTestUser(t, repo, "selfdisabled@test.com", "oldpassword", domain.RoleAdmin)
	require.NoError(t, repo.Deactivate(ctx, user.ID))

	// Per the contract, changing a password does not depend on or alter is_active.
	result, err := svc.ChangeOwnPassword(ctx, user.ID, "oldpassword", "newpassword1")
	require.NoError(t, err)
	assert.False(t, result.User.IsActive)
	assert.NotEmpty(t, result.AccessToken)

	stored, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.False(t, stored.IsActive, "the account stays disabled")
	assert.Equal(t, result.User.PasswordHash, stored.PasswordHash)

	// The returned refresh token cannot extend the session of a disabled account.
	_, err = svc.RefreshToken(ctx, result.RefreshToken)
	assert.ErrorIs(t, err, domain.ErrAccountDisabled)
}

func TestAuthService_RefreshToken_RevokedByPasswordSet(t *testing.T) {
	tests := []struct {
		name        string
		newPassword string
	}{
		{name: "different password", newPassword: "newpassword1"},
		// bcrypt salts are random, so re-setting the same password still rotates
		// the fingerprint and revokes outstanding refresh tokens.
		{name: "same password set again", newPassword: "oldpassword"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := setupAuthTest(t)
			ctx := context.Background()
			user := createTestUser(t, repo, "revoke@test.com", "oldpassword", domain.RoleInvestor)

			before, err := svc.Login(ctx, "revoke@test.com", "oldpassword")
			require.NoError(t, err)

			_, err = svc.SetPassword(ctx, user.ID, tt.newPassword)
			require.NoError(t, err)

			_, err = svc.RefreshToken(ctx, before.RefreshToken)
			assert.ErrorIs(t, err, domain.ErrUnauthorized)

			// A session started after the change refreshes normally.
			after, err := svc.Login(ctx, "revoke@test.com", tt.newPassword)
			require.NoError(t, err)
			_, err = svc.RefreshToken(ctx, after.RefreshToken)
			assert.NoError(t, err)
		})
	}
}

func TestAuthService_RefreshToken_UnchangedPasswordStillRefreshes(t *testing.T) {
	svc, repo := setupAuthTest(t)
	ctx := context.Background()
	user := createTestUser(t, repo, "stable@test.com", "password1", domain.RoleInvestor)

	result, err := svc.Login(ctx, "stable@test.com", "password1")
	require.NoError(t, err)

	refreshClaims, err := svc.ValidateToken(result.RefreshToken)
	require.NoError(t, err)
	assert.Len(t, refreshClaims.PasswordVersion, 2*passwordVersionBytes)
	assert.NotContains(t, user.PasswordHash, refreshClaims.PasswordVersion)

	for i := 0; i < 2; i++ {
		access, err := svc.RefreshToken(ctx, result.RefreshToken)
		require.NoError(t, err)

		accessClaims, err := svc.ValidateToken(access)
		require.NoError(t, err)
		assert.Equal(t, refreshClaims.PasswordVersion, accessClaims.PasswordVersion)
	}
}

func TestAuthService_RefreshToken_RejectsTokenWithoutPasswordVersion(t *testing.T) {
	svc, repo := setupAuthTest(t)
	ctx := context.Background()
	user := createTestUser(t, repo, "legacy@test.com", "password1", domain.RoleInvestor)

	// A refresh token as minted before the pwv claim existed.
	now := time.Now()
	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    tokenIssuer,
		},
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		TokenType: TokenTypeRefresh,
	}
	legacy, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))
	require.NoError(t, err)

	// The token is otherwise valid, so the rejection is due to the missing claim.
	parsed, err := svc.ValidateToken(legacy)
	require.NoError(t, err)
	assert.Empty(t, parsed.PasswordVersion)

	_, err = svc.RefreshToken(ctx, legacy)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := New(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { db.Close() })
	return db
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{
		ID:           "user-001",
		Email:        "admin@test.com",
		Name:         "Admin User",
		PasswordHash: "$2a$12$fakehash",
		Role:         domain.RoleAdmin,
		IsActive:     true,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)
	assert.False(t, user.CreatedAt.IsZero())
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user1 := &domain.User{ID: "u1", Email: "dup@test.com", Name: "A", PasswordHash: "h", Role: domain.RoleAdmin, IsActive: true}
	user2 := &domain.User{ID: "u2", Email: "dup@test.com", Name: "B", PasswordHash: "h", Role: domain.RoleAdmin, IsActive: true}

	require.NoError(t, repo.Create(ctx, user1))
	err := repo.Create(ctx, user2)
	assert.Error(t, err, "duplicate email should fail")
}

func TestUserRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{ID: "u1", Email: "get@test.com", Name: "Get User", PasswordHash: "h", Role: domain.RoleInvestor, IsActive: true}
	require.NoError(t, repo.Create(ctx, user))

	found, err := repo.GetByID(ctx, "u1")
	require.NoError(t, err)
	assert.Equal(t, "get@test.com", found.Email)
	assert.Equal(t, "Get User", found.Name)
	assert.Equal(t, domain.RoleInvestor, found.Role)
	assert.True(t, found.IsActive)
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{ID: "u1", Email: "email@test.com", Name: "Email User", PasswordHash: "h", Role: domain.RoleCompanyMember, IsActive: true}
	require.NoError(t, repo.Create(ctx, user))

	found, err := repo.GetByEmail(ctx, "email@test.com")
	require.NoError(t, err)
	assert.Equal(t, "u1", found.ID)
}

func TestUserRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		u := &domain.User{
			ID: fmt.Sprintf("u%d", i), Email: fmt.Sprintf("u%d@test.com", i),
			Name: fmt.Sprintf("User %d", i), PasswordHash: "h", Role: domain.RoleInvestor, IsActive: true,
		}
		require.NoError(t, repo.Create(ctx, u))
	}

	users, total, err := repo.List(ctx, 3, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, users, 3)

	users2, _, err := repo.List(ctx, 3, 3)
	require.NoError(t, err)
	assert.Len(t, users2, 2)
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{ID: "u1", Email: "up@test.com", Name: "Before", PasswordHash: "h", Role: domain.RoleInvestor, IsActive: true}
	require.NoError(t, repo.Create(ctx, user))

	user.Name = "After"
	user.Role = domain.RoleCompanyMember
	err := repo.Update(ctx, user)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, "u1")
	require.NoError(t, err)
	assert.Equal(t, "After", found.Name)
	assert.Equal(t, domain.RoleCompanyMember, found.Role)
}

func TestUserRepository_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{ID: "nonexistent", Name: "X", Email: "x@x.com", Role: domain.RoleAdmin}
	err := repo.Update(ctx, user)
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestUserRepository_Deactivate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{ID: "u1", Email: "deact@test.com", Name: "Deact", PasswordHash: "h", Role: domain.RoleInvestor, IsActive: true}
	require.NoError(t, repo.Create(ctx, user))

	err := repo.Deactivate(ctx, "u1")
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, "u1")
	require.NoError(t, err)
	assert.False(t, found.IsActive)
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{ID: "u1", Email: "login@test.com", Name: "Login", PasswordHash: "h", Role: domain.RoleAdmin, IsActive: true}
	require.NoError(t, repo.Create(ctx, user))

	err := repo.UpdateLastLogin(ctx, "u1")
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, "u1")
	require.NoError(t, err)
	assert.NotNil(t, found.LastLoginAt)
}

func TestUserRepository_InviteToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create inviting user first.
	admin := &domain.User{ID: "admin1", Email: "admin@test.com", Name: "Admin", PasswordHash: "h", Role: domain.RoleAdmin, IsActive: true}
	require.NoError(t, repo.Create(ctx, admin))

	token := &domain.InviteToken{
		ID:        "tok1",
		Token:     "abc123",
		Email:     "investor@test.com",
		Role:      domain.RoleInvestor,
		InvitedBy: "admin1",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := repo.CreateInviteToken(ctx, token)
	require.NoError(t, err)

	found, err := repo.GetInviteToken(ctx, "abc123")
	require.NoError(t, err)
	assert.Equal(t, "investor@test.com", found.Email)
	assert.Equal(t, domain.RoleInvestor, found.Role)
	assert.Nil(t, found.UsedAt)

	err = repo.MarkInviteTokenUsed(ctx, "abc123")
	require.NoError(t, err)

	found, err = repo.GetInviteToken(ctx, "abc123")
	require.NoError(t, err)
	assert.NotNil(t, found.UsedAt)
}

func TestUserRepository_InviteToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.GetInviteToken(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrTokenNotFound)
}

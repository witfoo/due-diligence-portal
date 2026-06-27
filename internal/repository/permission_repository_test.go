package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

// helper: creates two users (granter and grantee) for permission tests.
func seedPermissionDeps(t *testing.T, db *DB) (granterID, granteeID string) {
	t.Helper()
	ctx := context.Background()
	granterID = "perm-admin"
	granteeID = "perm-investor"

	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, email, name, password_hash, role, is_active) VALUES (?, ?, ?, ?, ?, ?)`,
		granterID, "admin@perm.com", "Admin", "hash", domain.RoleAdmin, true)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx,
		`INSERT INTO users (id, email, name, password_hash, role, is_active) VALUES (?, ?, ?, ?, ?, ?)`,
		granteeID, "investor@perm.com", "Investor", "hash", domain.RoleInvestor, true)
	require.NoError(t, err)

	return granterID, granteeID
}

func TestPermissionRepository_Grant(t *testing.T) {
	db := setupTestDB(t)
	granterID, granteeID := seedPermissionDeps(t, db)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	grant := &domain.AccessGrant{
		ID:           "grant-001",
		UserID:       granteeID,
		ResourceType: domain.ResourceCategory,
		ResourceID:   "cat-corporate",
		AccessLevel:  domain.AccessView,
		GrantedBy:    granterID,
	}

	err := repo.Grant(ctx, grant)
	require.NoError(t, err)
	assert.False(t, grant.CreatedAt.IsZero())
}

func TestPermissionRepository_Grant_Upsert(t *testing.T) {
	db := setupTestDB(t)
	granterID, granteeID := seedPermissionDeps(t, db)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	g1 := &domain.AccessGrant{
		ID: "g1", UserID: granteeID, ResourceType: domain.ResourceCategory,
		ResourceID: "cat-corporate", AccessLevel: domain.AccessView, GrantedBy: granterID,
	}
	g2 := &domain.AccessGrant{
		ID: "g2", UserID: granteeID, ResourceType: domain.ResourceCategory,
		ResourceID: "cat-corporate", AccessLevel: domain.AccessDownload, GrantedBy: granterID,
	}

	// Re-granting the same user+resource upserts the access level rather than failing
	// the UNIQUE constraint (lets admins upgrade view -> download).
	require.NoError(t, repo.Grant(ctx, g1))
	require.NoError(t, repo.Grant(ctx, g2))

	grants, err := repo.ListByUser(ctx, granteeID)
	require.NoError(t, err)
	require.Len(t, grants, 1, "upsert must not create a second row")
	assert.Equal(t, domain.AccessDownload, grants[0].AccessLevel)
}

func TestPermissionRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	granterID, granteeID := seedPermissionDeps(t, db)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	grant := &domain.AccessGrant{
		ID: "grant-001", UserID: granteeID, ResourceType: domain.ResourceCategory,
		ResourceID: "cat-financials", AccessLevel: domain.AccessDownload, GrantedBy: granterID,
	}
	require.NoError(t, repo.Grant(ctx, grant))

	found, err := repo.GetByID(ctx, "grant-001")
	require.NoError(t, err)
	assert.Equal(t, granteeID, found.UserID)
	assert.Equal(t, domain.AccessDownload, found.AccessLevel)
	assert.Equal(t, "investor@perm.com", found.UserEmail)
	assert.Equal(t, "Investor", found.UserName)
}

func TestPermissionRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrGrantNotFound)
}

func TestPermissionRepository_ListByUser(t *testing.T) {
	db := setupTestDB(t)
	granterID, granteeID := seedPermissionDeps(t, db)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	grants := []*domain.AccessGrant{
		{ID: "g1", UserID: granteeID, ResourceType: domain.ResourceCategory, ResourceID: "cat-corporate", AccessLevel: domain.AccessView, GrantedBy: granterID},
		{ID: "g2", UserID: granteeID, ResourceType: domain.ResourceCategory, ResourceID: "cat-financials", AccessLevel: domain.AccessDownload, GrantedBy: granterID},
		{ID: "g3", UserID: granterID, ResourceType: domain.ResourceCategory, ResourceID: "cat-legal", AccessLevel: domain.AccessManage, GrantedBy: granterID},
	}
	for _, g := range grants {
		require.NoError(t, repo.Grant(ctx, g))
	}

	list, err := repo.ListByUser(ctx, granteeID)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestPermissionRepository_ListByResource(t *testing.T) {
	db := setupTestDB(t)
	granterID, granteeID := seedPermissionDeps(t, db)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	grants := []*domain.AccessGrant{
		{ID: "g1", UserID: granteeID, ResourceType: domain.ResourceCategory, ResourceID: "cat-corporate", AccessLevel: domain.AccessView, GrantedBy: granterID},
		{ID: "g2", UserID: granterID, ResourceType: domain.ResourceCategory, ResourceID: "cat-corporate", AccessLevel: domain.AccessManage, GrantedBy: granterID},
	}
	for _, g := range grants {
		require.NoError(t, repo.Grant(ctx, g))
	}

	list, err := repo.ListByResource(ctx, domain.ResourceCategory, "cat-corporate")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestPermissionRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	granterID, granteeID := seedPermissionDeps(t, db)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	grant := &domain.AccessGrant{
		ID: "g1", UserID: granteeID, ResourceType: domain.ResourceCategory,
		ResourceID: "cat-corporate", AccessLevel: domain.AccessView, GrantedBy: granterID,
	}
	require.NoError(t, repo.Grant(ctx, grant))

	grant.AccessLevel = domain.AccessManage
	expires := time.Now().Add(24 * time.Hour)
	grant.ExpiresAt = &expires
	err := repo.Update(ctx, grant)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, "g1")
	require.NoError(t, err)
	assert.Equal(t, domain.AccessManage, found.AccessLevel)
	assert.NotNil(t, found.ExpiresAt)
}

func TestPermissionRepository_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	grant := &domain.AccessGrant{ID: "nonexistent", AccessLevel: domain.AccessView}
	err := repo.Update(ctx, grant)
	assert.ErrorIs(t, err, domain.ErrGrantNotFound)
}

func TestPermissionRepository_Revoke(t *testing.T) {
	db := setupTestDB(t)
	granterID, granteeID := seedPermissionDeps(t, db)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	grant := &domain.AccessGrant{
		ID: "g1", UserID: granteeID, ResourceType: domain.ResourceCategory,
		ResourceID: "cat-corporate", AccessLevel: domain.AccessView, GrantedBy: granterID,
	}
	require.NoError(t, repo.Grant(ctx, grant))

	err := repo.Revoke(ctx, "g1")
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, "g1")
	assert.ErrorIs(t, err, domain.ErrGrantNotFound)
}

func TestPermissionRepository_Revoke_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	err := repo.Revoke(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrGrantNotFound)
}

func TestPermissionRepository_HasAccess(t *testing.T) {
	db := setupTestDB(t)
	granterID, granteeID := seedPermissionDeps(t, db)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	// Grant "manage" level.
	grant := &domain.AccessGrant{
		ID: "g1", UserID: granteeID, ResourceType: domain.ResourceCategory,
		ResourceID: "cat-corporate", AccessLevel: domain.AccessManage, GrantedBy: granterID,
	}
	require.NoError(t, repo.Grant(ctx, grant))

	tests := []struct {
		name     string
		required string
		want     bool
	}{
		{"manage includes view", domain.AccessView, true},
		{"manage includes upload", domain.AccessUpload, true},
		{"manage includes download", domain.AccessDownload, true},
		{"manage includes manage", domain.AccessManage, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has, err := repo.HasAccess(ctx, granteeID, domain.ResourceCategory, "cat-corporate", tt.required)
			require.NoError(t, err)
			assert.Equal(t, tt.want, has)
		})
	}

	// No grant at all.
	has, err := repo.HasAccess(ctx, granteeID, domain.ResourceCategory, "cat-financials", domain.AccessView)
	require.NoError(t, err)
	assert.False(t, has)
}

func TestPermissionRepository_HasAccess_Expired(t *testing.T) {
	db := setupTestDB(t)
	granterID, granteeID := seedPermissionDeps(t, db)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	expired := time.Now().Add(-1 * time.Hour)
	grant := &domain.AccessGrant{
		ID: "g1", UserID: granteeID, ResourceType: domain.ResourceCategory,
		ResourceID: "cat-corporate", AccessLevel: domain.AccessManage,
		GrantedBy: granterID, ExpiresAt: &expired,
	}
	require.NoError(t, repo.Grant(ctx, grant))

	has, err := repo.HasAccess(ctx, granteeID, domain.ResourceCategory, "cat-corporate", domain.AccessView)
	require.NoError(t, err)
	assert.False(t, has, "expired grant should not provide access")
}

func TestPermissionRepository_HasAccess_ViewOnlyDeniesUpload(t *testing.T) {
	db := setupTestDB(t)
	granterID, granteeID := seedPermissionDeps(t, db)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	grant := &domain.AccessGrant{
		ID: "g1", UserID: granteeID, ResourceType: domain.ResourceCategory,
		ResourceID: "cat-corporate", AccessLevel: domain.AccessView, GrantedBy: granterID,
	}
	require.NoError(t, repo.Grant(ctx, grant))

	has, err := repo.HasAccess(ctx, granteeID, domain.ResourceCategory, "cat-corporate", domain.AccessUpload)
	require.NoError(t, err)
	assert.False(t, has, "view-only should not grant upload access")
}

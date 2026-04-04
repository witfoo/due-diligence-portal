package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

func TestAuditRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "audit@test.com")

	entry := &domain.AuditEntry{
		ID:           "a1",
		UserID:       "u1",
		UserEmail:    "audit@test.com",
		Action:       domain.AuditUserLogin,
		ResourceType: "user",
		ResourceID:   "u1",
		IPAddress:    "127.0.0.1",
		UserAgent:    "TestAgent/1.0",
	}

	err := repo.Create(ctx, entry)
	require.NoError(t, err)
	assert.False(t, entry.CreatedAt.IsZero())
}

func TestAuditRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "audit@test.com")

	for i := 0; i < 5; i++ {
		action := domain.AuditUserLogin
		if i >= 3 {
			action = domain.AuditDocumentViewed
		}
		entry := &domain.AuditEntry{
			ID: fmt.Sprintf("a%d", i), UserID: "u1", UserEmail: "audit@test.com",
			Action: action, ResourceType: "user", ResourceID: "u1",
		}
		require.NoError(t, repo.Create(ctx, entry))
	}

	// List all.
	entries, total, err := repo.List(ctx, AuditFilter{}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, entries, 5)

	// Filter by action.
	entries, total, err = repo.List(ctx, AuditFilter{Action: domain.AuditUserLogin}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, entries, 3)

	// Pagination.
	entries, total, err = repo.List(ctx, AuditFilter{}, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, entries, 2)
}

func TestAuditRepository_List_FilterByUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "user1@test.com")
	createTestUser(t, db, "u2", "user2@test.com")

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, &domain.AuditEntry{
			ID: fmt.Sprintf("a1-%d", i), UserID: "u1", UserEmail: "user1@test.com",
			Action: domain.AuditUserLogin,
		}))
	}
	require.NoError(t, repo.Create(ctx, &domain.AuditEntry{
		ID: "a2-0", UserID: "u2", UserEmail: "user2@test.com",
		Action: domain.AuditUserLogin,
	}))

	entries, total, err := repo.List(ctx, AuditFilter{UserID: "u1"}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, entries, 3)
}

func TestAuditRepository_GetByDocument(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "audit@test.com")

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, &domain.AuditEntry{
			ID: fmt.Sprintf("a%d", i), UserID: "u1", UserEmail: "audit@test.com",
			Action: domain.AuditDocumentViewed, ResourceType: "document", ResourceID: "doc1",
		}))
	}
	require.NoError(t, repo.Create(ctx, &domain.AuditEntry{
		ID: "a-other", UserID: "u1", UserEmail: "audit@test.com",
		Action: domain.AuditDocumentViewed, ResourceType: "document", ResourceID: "doc2",
	}))

	entries, total, err := repo.GetByDocument(ctx, "doc1", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, entries, 3)
}

func TestAuditRepository_GetByUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "user1@test.com")
	createTestUser(t, db, "u2", "user2@test.com")

	require.NoError(t, repo.Create(ctx, &domain.AuditEntry{
		ID: "a1", UserID: "u1", UserEmail: "user1@test.com", Action: domain.AuditUserLogin,
	}))
	require.NoError(t, repo.Create(ctx, &domain.AuditEntry{
		ID: "a2", UserID: "u2", UserEmail: "user2@test.com", Action: domain.AuditUserLogin,
	}))

	entries, total, err := repo.GetByUser(ctx, "u1", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, entries, 1)
	assert.Equal(t, "u1", entries[0].UserID)
}

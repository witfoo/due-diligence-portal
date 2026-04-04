package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

func TestAnalyticsRepository_RecordView(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAnalyticsRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "viewer@test.com")
	createTestCategory(t, db, "cat1", "test-cat")
	createTestDocument(t, db, "doc1", "cat1", "u1")

	dur := int64(5000)
	pages := 3
	event := &domain.ViewEvent{
		ID:         "v1",
		UserID:     "u1",
		DocumentID: "doc1",
		DurationMs: &dur,
		PageCount:  &pages,
	}

	err := repo.RecordView(ctx, event)
	require.NoError(t, err)
	assert.False(t, event.CreatedAt.IsZero())
}

func TestAnalyticsRepository_GetDocumentAnalytics(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAnalyticsRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "viewer1@test.com")
	createTestUser(t, db, "u2", "viewer2@test.com")
	createTestCategory(t, db, "cat1", "test-cat")
	createTestDocument(t, db, "doc1", "cat1", "u1")

	dur := int64(3000)
	for i, uid := range []string{"u1", "u1", "u2"} {
		require.NoError(t, repo.RecordView(ctx, &domain.ViewEvent{
			ID: fmt.Sprintf("v%d", i), UserID: uid, DocumentID: "doc1", DurationMs: &dur,
		}))
	}

	da, err := repo.GetDocumentAnalytics(ctx, "doc1")
	require.NoError(t, err)
	assert.Equal(t, "doc1", da.DocumentID)
	assert.Equal(t, 3, da.ViewCount)
	assert.Equal(t, 2, da.UniqueViewers)
	assert.Equal(t, int64(3000), da.AvgDurationMs)
	assert.NotNil(t, da.LastViewedAt)
}

func TestAnalyticsRepository_GetDocumentAnalytics_NoViews(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAnalyticsRepository(db)
	ctx := context.Background()

	da, err := repo.GetDocumentAnalytics(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, 0, da.ViewCount)
}

func TestAnalyticsRepository_GetUserAnalytics(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAnalyticsRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "viewer@test.com")
	createTestCategory(t, db, "cat1", "test-cat")
	createTestDocument(t, db, "doc1", "cat1", "u1")
	createTestDocument(t, db, "doc2", "cat1", "u1")

	dur := int64(2000)
	require.NoError(t, repo.RecordView(ctx, &domain.ViewEvent{
		ID: "v1", UserID: "u1", DocumentID: "doc1", DurationMs: &dur,
	}))
	require.NoError(t, repo.RecordView(ctx, &domain.ViewEvent{
		ID: "v2", UserID: "u1", DocumentID: "doc2", DurationMs: &dur,
	}))
	require.NoError(t, repo.RecordView(ctx, &domain.ViewEvent{
		ID: "v3", UserID: "u1", DocumentID: "doc1", DurationMs: &dur,
	}))

	ua, err := repo.GetUserAnalytics(ctx, "u1")
	require.NoError(t, err)
	assert.Equal(t, "u1", ua.UserID)
	assert.Equal(t, 2, ua.DocumentsViewed)
	assert.Equal(t, 3, ua.TotalViews)
	assert.Equal(t, int64(6000), ua.TotalDurationMs)
	assert.NotNil(t, ua.LastActiveAt)
}

func TestAnalyticsRepository_GetUserAnalytics_NoViews(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAnalyticsRepository(db)
	ctx := context.Background()

	ua, err := repo.GetUserAnalytics(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, 0, ua.TotalViews)
}

func TestAnalyticsRepository_GetEngagementSummary(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAnalyticsRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "investor@test.com")
	createTestCategory(t, db, "cat1", "test-cat")
	createTestDocument(t, db, "doc1", "cat1", "u1")

	dur := int64(1000)
	require.NoError(t, repo.RecordView(ctx, &domain.ViewEvent{
		ID: "v1", UserID: "u1", DocumentID: "doc1", DurationMs: &dur,
	}))

	summary, err := repo.GetEngagementSummary(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.TotalDocuments)
	assert.Equal(t, 1, summary.TotalViews)
	assert.Equal(t, 1, summary.UniqueViewers)
}

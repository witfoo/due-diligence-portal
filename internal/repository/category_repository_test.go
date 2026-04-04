package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

func TestCategoryRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	cat := &domain.Category{
		ID:          "cat-new",
		Name:        "New Category",
		Slug:        "new-category",
		Description: "A new test category",
		SortOrder:   99,
		Icon:        "Folder",
	}

	err := repo.Create(ctx, cat)
	require.NoError(t, err)
	assert.False(t, cat.CreatedAt.IsZero())
}

func TestCategoryRepository_Create_DuplicateSlug(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	cat1 := &domain.Category{ID: "c1", Name: "A", Slug: "dup-slug", SortOrder: 1}
	cat2 := &domain.Category{ID: "c2", Name: "B", Slug: "dup-slug", SortOrder: 2}

	require.NoError(t, repo.Create(ctx, cat1))
	err := repo.Create(ctx, cat2)
	assert.Error(t, err, "duplicate slug should fail")
}

func TestCategoryRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	// Use a seeded category.
	found, err := repo.GetByID(ctx, "cat-corporate")
	require.NoError(t, err)
	assert.Equal(t, "Corporate", found.Name)
	assert.Equal(t, "corporate", found.Slug)
}

func TestCategoryRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
}

func TestCategoryRepository_GetBySlug(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	found, err := repo.GetBySlug(ctx, "financials")
	require.NoError(t, err)
	assert.Equal(t, "cat-financials", found.ID)
	assert.Equal(t, "Financials", found.Name)
}

func TestCategoryRepository_GetBySlug_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	_, err := repo.GetBySlug(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
}

func TestCategoryRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	cats, err := repo.List(ctx)
	require.NoError(t, err)
	// 10 seeded categories from 003_seed_categories.sql.
	assert.Len(t, cats, 10)
	// Verify sort order: first should be Corporate (sort_order=1).
	assert.Equal(t, "Corporate", cats[0].Name)
}

func TestCategoryRepository_ListAsTree(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	// Create a child category under "Corporate".
	parentID := "cat-corporate"
	child := &domain.Category{
		ID: "cat-corp-sub", Name: "Board Minutes", Slug: "board-minutes",
		ParentID: &parentID, SortOrder: 1,
	}
	require.NoError(t, repo.Create(ctx, child))

	tree, err := repo.ListAsTree(ctx)
	require.NoError(t, err)
	// Root nodes should be the 10 seeded categories (child is nested).
	assert.Len(t, tree, 10)

	// Find Corporate and check its children.
	var corporate *domain.Category
	for _, c := range tree {
		if c.ID == "cat-corporate" {
			corporate = c
			break
		}
	}
	require.NotNil(t, corporate)
	assert.Len(t, corporate.Children, 1)
	assert.Equal(t, "Board Minutes", corporate.Children[0].Name)
}

func TestCategoryRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	cat := &domain.Category{ID: "cat-new", Name: "Before", Slug: "before-slug", SortOrder: 1}
	require.NoError(t, repo.Create(ctx, cat))

	cat.Name = "After"
	cat.Slug = "after-slug"
	cat.Description = "updated desc"
	err := repo.Update(ctx, cat)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, "cat-new")
	require.NoError(t, err)
	assert.Equal(t, "After", found.Name)
	assert.Equal(t, "after-slug", found.Slug)
	assert.Equal(t, "updated desc", found.Description)
}

func TestCategoryRepository_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	cat := &domain.Category{ID: "nonexistent", Name: "X", Slug: "x"}
	err := repo.Update(ctx, cat)
	assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
}

func TestCategoryRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	cat := &domain.Category{ID: "cat-del", Name: "To Delete", Slug: "to-delete", SortOrder: 99}
	require.NoError(t, repo.Create(ctx, cat))

	err := repo.Delete(ctx, "cat-del")
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, "cat-del")
	assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
}

func TestCategoryRepository_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
}

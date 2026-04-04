package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

// helper: creates a user and a category so documents can reference them via FK.
func seedDocumentDeps(t *testing.T, db *DB) (userID, categoryID string) {
	t.Helper()
	ctx := context.Background()
	userID = "doc-test-user"
	categoryID = "doc-test-cat"

	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, email, name, password_hash, role, is_active) VALUES (?, ?, ?, ?, ?, ?)`,
		userID, "docuser@test.com", "Doc User", "hash", domain.RoleAdmin, true)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx,
		`INSERT INTO categories (id, name, slug, sort_order) VALUES (?, ?, ?, ?)`,
		categoryID, "Test Cat", "test-cat", 1)
	require.NoError(t, err)

	return userID, categoryID
}

func TestDocumentRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	userID, catID := seedDocumentDeps(t, db)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	doc := &domain.Document{
		ID:             "doc-001",
		Name:           "Business Plan",
		Description:    "Q4 business plan",
		CategoryID:     catID,
		UploadedBy:     userID,
		CurrentVersion: 1,
		MimeType:       "application/pdf",
		FileSize:       1024,
		Tags:           "plan,business",
	}

	err := repo.Create(ctx, doc)
	require.NoError(t, err)
	assert.False(t, doc.CreatedAt.IsZero())
}

func TestDocumentRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	userID, catID := seedDocumentDeps(t, db)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	doc := &domain.Document{
		ID: "doc-001", Name: "Test Doc", CategoryID: catID, UploadedBy: userID,
		CurrentVersion: 1, MimeType: "application/pdf", FileSize: 512,
	}
	require.NoError(t, repo.Create(ctx, doc))

	found, err := repo.GetByID(ctx, "doc-001")
	require.NoError(t, err)
	assert.Equal(t, "Test Doc", found.Name)
	assert.Equal(t, "Test Cat", found.CategoryName)
	assert.Equal(t, "Doc User", found.UploaderName)
}

func TestDocumentRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrDocumentNotFound)
}

func TestDocumentRepository_List(t *testing.T) {
	db := setupTestDB(t)
	userID, catID := seedDocumentDeps(t, db)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	// Create a second category (use unique slug to avoid seed conflicts).
	_, err := db.ExecContext(ctx,
		`INSERT INTO categories (id, name, slug, sort_order) VALUES (?, ?, ?, ?)`,
		"cat-doctest-list-other", "Other Cat", "doctest-list-other", 99)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		c := catID
		if i >= 3 {
			c = "cat-doctest-list-other"
		}
		d := &domain.Document{
			ID: fmt.Sprintf("doc-%d", i), Name: fmt.Sprintf("Doc %d", i),
			CategoryID: c, UploadedBy: userID, CurrentVersion: 1,
			MimeType: "text/plain", FileSize: 100,
		}
		require.NoError(t, repo.Create(ctx, d))
	}

	// List all.
	docs, total, err := repo.List(ctx, "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, docs, 5)

	// Filter by category.
	docs, total, err = repo.List(ctx, catID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, docs, 3)

	// Pagination.
	docs, _, err = repo.List(ctx, "", 2, 0)
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestDocumentRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	userID, catID := seedDocumentDeps(t, db)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	doc := &domain.Document{
		ID: "doc-001", Name: "Original", CategoryID: catID, UploadedBy: userID,
		CurrentVersion: 1, MimeType: "text/plain", FileSize: 100,
	}
	require.NoError(t, repo.Create(ctx, doc))

	doc.Name = "Updated"
	doc.Description = "new description"
	err := repo.Update(ctx, doc)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, "doc-001")
	require.NoError(t, err)
	assert.Equal(t, "Updated", found.Name)
	assert.Equal(t, "new description", found.Description)
}

func TestDocumentRepository_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	doc := &domain.Document{ID: "nonexistent", Name: "X", CategoryID: "x", MimeType: "x"}
	err := repo.Update(ctx, doc)
	assert.ErrorIs(t, err, domain.ErrDocumentNotFound)
}

func TestDocumentRepository_Archive(t *testing.T) {
	db := setupTestDB(t)
	userID, catID := seedDocumentDeps(t, db)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	doc := &domain.Document{
		ID: "doc-001", Name: "To Archive", CategoryID: catID, UploadedBy: userID,
		CurrentVersion: 1, MimeType: "text/plain", FileSize: 100,
	}
	require.NoError(t, repo.Create(ctx, doc))

	err := repo.Archive(ctx, "doc-001")
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, "doc-001")
	require.NoError(t, err)
	assert.True(t, found.IsArchived)

	// Archived docs should not appear in list.
	docs, total, err := repo.List(ctx, "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Len(t, docs, 0)
}

func TestDocumentRepository_Archive_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	err := repo.Archive(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrDocumentNotFound)
}

func TestDocumentRepository_Versions(t *testing.T) {
	db := setupTestDB(t)
	userID, catID := seedDocumentDeps(t, db)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	doc := &domain.Document{
		ID: "doc-001", Name: "Versioned", CategoryID: catID, UploadedBy: userID,
		CurrentVersion: 1, MimeType: "application/pdf", FileSize: 100,
	}
	require.NoError(t, repo.Create(ctx, doc))

	v1 := &domain.DocumentVersion{
		ID: "v1", DocumentID: "doc-001", VersionNumber: 1,
		FileData: []byte("content v1"), FileSize: 10, MimeType: "application/pdf",
		ChecksumSHA256: "sha256-v1", ChangeNote: "initial upload", UploadedBy: userID,
	}
	err := repo.CreateVersion(ctx, v1)
	require.NoError(t, err)
	assert.False(t, v1.CreatedAt.IsZero())

	v2 := &domain.DocumentVersion{
		ID: "v2", DocumentID: "doc-001", VersionNumber: 2,
		FileData: []byte("content v2"), FileSize: 10, MimeType: "application/pdf",
		ChecksumSHA256: "sha256-v2", ChangeNote: "revision", UploadedBy: userID,
	}
	require.NoError(t, repo.CreateVersion(ctx, v2))

	// GetVersion.
	got, err := repo.GetVersion(ctx, "doc-001", 1)
	require.NoError(t, err)
	assert.Equal(t, "v1", got.ID)
	assert.Equal(t, []byte("content v1"), got.FileData)
	assert.Equal(t, "initial upload", got.ChangeNote)

	// GetVersion not found.
	_, err = repo.GetVersion(ctx, "doc-001", 99)
	assert.ErrorIs(t, err, domain.ErrVersionNotFound)

	// ListVersions (desc order).
	versions, err := repo.ListVersions(ctx, "doc-001")
	require.NoError(t, err)
	assert.Len(t, versions, 2)
	assert.Equal(t, 2, versions[0].VersionNumber)
	assert.Equal(t, 1, versions[1].VersionNumber)
}

func TestDocumentRepository_Search(t *testing.T) {
	db := setupTestDB(t)
	userID, catID := seedDocumentDeps(t, db)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	docs := []struct {
		id, name, desc, tags string
	}{
		{"d1", "Financial Report 2024", "Annual financial statements", "finance,annual"},
		{"d2", "Technical Architecture", "System design document", "tech,architecture"},
		{"d3", "Financial Projections", "Revenue projections for Q4", "finance,projections"},
	}
	for _, d := range docs {
		doc := &domain.Document{
			ID: d.id, Name: d.name, Description: d.desc, Tags: d.tags,
			CategoryID: catID, UploadedBy: userID, CurrentVersion: 1,
			MimeType: "application/pdf", FileSize: 100,
		}
		require.NoError(t, repo.Create(ctx, doc))
	}

	// Search for "financial".
	results, total, err := repo.Search(ctx, "financial", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, results, 2)

	// Search for "architecture".
	results, total, err = repo.Search(ctx, "architecture", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, results, 1)
	assert.Equal(t, "d2", results[0].ID)

	// Search with no results.
	results, total, err = repo.Search(ctx, "nonexistent", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Len(t, results, 0)
}

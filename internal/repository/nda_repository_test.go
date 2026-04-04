package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

func TestNDARepository_CreateTemplate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNDARepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "nda@test.com")

	tmpl := &domain.NDATemplate{
		ID:        "tmpl1",
		Name:      "Standard NDA",
		Content:   "NDA content here...",
		IsActive:  true,
		Version:   1,
		CreatedBy: "u1",
	}

	err := repo.CreateTemplate(ctx, tmpl)
	require.NoError(t, err)
	assert.False(t, tmpl.CreatedAt.IsZero())
}

func TestNDARepository_GetTemplate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNDARepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "nda@test.com")

	tmpl := &domain.NDATemplate{
		ID: "tmpl1", Name: "Standard NDA", Content: "NDA content",
		IsActive: true, Version: 1, CreatedBy: "u1",
	}
	require.NoError(t, repo.CreateTemplate(ctx, tmpl))

	found, err := repo.GetTemplate(ctx, "tmpl1")
	require.NoError(t, err)
	assert.Equal(t, "Standard NDA", found.Name)
	assert.Equal(t, "NDA content", found.Content)
	assert.True(t, found.IsActive)
	assert.Equal(t, 1, found.Version)
}

func TestNDARepository_GetTemplate_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNDARepository(db)
	ctx := context.Background()

	_, err := repo.GetTemplate(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrTemplateNotFound)
}

func TestNDARepository_ListTemplates(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNDARepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "nda@test.com")

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.CreateTemplate(ctx, &domain.NDATemplate{
			ID: fmt.Sprintf("tmpl%d", i), Name: fmt.Sprintf("NDA %d", i),
			Content: "content", IsActive: true, Version: 1, CreatedBy: "u1",
		}))
	}

	templates, err := repo.ListTemplates(ctx)
	require.NoError(t, err)
	assert.Len(t, templates, 3)
}

func TestNDARepository_UpdateTemplate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNDARepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "nda@test.com")

	tmpl := &domain.NDATemplate{
		ID: "tmpl1", Name: "V1", Content: "old content",
		IsActive: true, Version: 1, CreatedBy: "u1",
	}
	require.NoError(t, repo.CreateTemplate(ctx, tmpl))

	tmpl.Name = "V2"
	tmpl.Content = "new content"
	tmpl.Version = 2
	err := repo.UpdateTemplate(ctx, tmpl)
	require.NoError(t, err)

	found, err := repo.GetTemplate(ctx, "tmpl1")
	require.NoError(t, err)
	assert.Equal(t, "V2", found.Name)
	assert.Equal(t, "new content", found.Content)
	assert.Equal(t, 2, found.Version)
}

func TestNDARepository_UpdateTemplate_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNDARepository(db)
	ctx := context.Background()

	tmpl := &domain.NDATemplate{ID: "nonexistent", Name: "X", Content: "x", Version: 1}
	err := repo.UpdateTemplate(ctx, tmpl)
	assert.ErrorIs(t, err, domain.ErrTemplateNotFound)
}

func TestNDARepository_CreateSignature(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNDARepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "nda@test.com")
	require.NoError(t, repo.CreateTemplate(ctx, &domain.NDATemplate{
		ID: "tmpl1", Name: "NDA", Content: "content",
		IsActive: true, Version: 1, CreatedBy: "u1",
	}))

	sig := &domain.NDASignature{
		ID:          "sig1",
		TemplateID:  "tmpl1",
		UserID:      "u1",
		SignerName:  "Test User",
		SignerEmail: "nda@test.com",
		IPAddress:   "127.0.0.1",
	}

	err := repo.CreateSignature(ctx, sig)
	require.NoError(t, err)
	assert.False(t, sig.SignedAt.IsZero())
}

func TestNDARepository_GetSignature(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNDARepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "nda@test.com")
	require.NoError(t, repo.CreateTemplate(ctx, &domain.NDATemplate{
		ID: "tmpl1", Name: "NDA", Content: "content",
		IsActive: true, Version: 1, CreatedBy: "u1",
	}))
	require.NoError(t, repo.CreateSignature(ctx, &domain.NDASignature{
		ID: "sig1", TemplateID: "tmpl1", UserID: "u1",
		SignerName: "Test User", SignerEmail: "nda@test.com", IPAddress: "127.0.0.1",
	}))

	found, err := repo.GetSignature(ctx, "sig1")
	require.NoError(t, err)
	assert.Equal(t, "tmpl1", found.TemplateID)
	assert.Equal(t, "u1", found.UserID)
	assert.Equal(t, "Test User", found.SignerName)
}

func TestNDARepository_ListSignatures(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNDARepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "user1@test.com")
	createTestUser(t, db, "u2", "user2@test.com")
	require.NoError(t, repo.CreateTemplate(ctx, &domain.NDATemplate{
		ID: "tmpl1", Name: "NDA", Content: "content",
		IsActive: true, Version: 1, CreatedBy: "u1",
	}))

	require.NoError(t, repo.CreateSignature(ctx, &domain.NDASignature{
		ID: "sig1", TemplateID: "tmpl1", UserID: "u1",
		SignerName: "User 1", SignerEmail: "user1@test.com", IPAddress: "127.0.0.1",
	}))
	require.NoError(t, repo.CreateSignature(ctx, &domain.NDASignature{
		ID: "sig2", TemplateID: "tmpl1", UserID: "u2",
		SignerName: "User 2", SignerEmail: "user2@test.com", IPAddress: "127.0.0.1",
	}))

	sigs, err := repo.ListSignatures(ctx, "tmpl1")
	require.NoError(t, err)
	assert.Len(t, sigs, 2)
}

func TestNDARepository_HasSigned(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNDARepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "nda@test.com")
	require.NoError(t, repo.CreateTemplate(ctx, &domain.NDATemplate{
		ID: "tmpl1", Name: "NDA", Content: "content",
		IsActive: true, Version: 1, CreatedBy: "u1",
	}))

	// Not signed yet.
	signed, err := repo.HasSigned(ctx, "u1", "tmpl1")
	require.NoError(t, err)
	assert.False(t, signed)

	// Sign it.
	require.NoError(t, repo.CreateSignature(ctx, &domain.NDASignature{
		ID: "sig1", TemplateID: "tmpl1", UserID: "u1",
		SignerName: "Test User", SignerEmail: "nda@test.com", IPAddress: "127.0.0.1",
	}))

	// Now signed.
	signed, err = repo.HasSigned(ctx, "u1", "tmpl1")
	require.NoError(t, err)
	assert.True(t, signed)
}

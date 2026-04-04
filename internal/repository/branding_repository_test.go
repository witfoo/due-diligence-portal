package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

func TestBrandingRepository_GetConfig_Default(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBrandingRepository(db)
	ctx := context.Background()

	config, err := repo.GetConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Company", config.CompanyName)
	assert.Equal(t, "#0f62fe", config.PrimaryColor)
}

func TestBrandingRepository_UpsertConfig(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBrandingRepository(db)
	ctx := context.Background()

	config := domain.DefaultBrandingConfig()
	config.CompanyName = "Acme Corp"
	config.PrimaryColor = "#ff0000"

	err := repo.UpsertConfig(ctx, &config)
	require.NoError(t, err)
	assert.False(t, config.UpdatedAt.IsZero())

	found, err := repo.GetConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Acme Corp", found.CompanyName)
	assert.Equal(t, "#ff0000", found.PrimaryColor)
}

func TestBrandingRepository_UpsertConfig_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBrandingRepository(db)
	ctx := context.Background()

	config := domain.DefaultBrandingConfig()
	config.CompanyName = "First"
	require.NoError(t, repo.UpsertConfig(ctx, &config))

	config.CompanyName = "Second"
	require.NoError(t, repo.UpsertConfig(ctx, &config))

	found, err := repo.GetConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Second", found.CompanyName)
}

func TestBrandingRepository_ResetConfig(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBrandingRepository(db)
	ctx := context.Background()

	config := domain.DefaultBrandingConfig()
	config.CompanyName = "Custom"
	require.NoError(t, repo.UpsertConfig(ctx, &config))

	err := repo.ResetConfig(ctx)
	require.NoError(t, err)

	// After reset, should return defaults.
	found, err := repo.GetConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Company", found.CompanyName)
}

func TestBrandingRepository_Asset_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBrandingRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "brand@test.com")

	asset := &domain.BrandingAsset{
		Key:            "logo",
		FileData:       []byte("fake-png-data"),
		MimeType:       "image/png",
		FileSize:       13,
		ChecksumSHA256: "abc123",
		UploadedBy:     "u1",
	}

	// Create.
	err := repo.UpsertAsset(ctx, asset)
	require.NoError(t, err)
	assert.False(t, asset.CreatedAt.IsZero())

	// Get.
	found, err := repo.GetAsset(ctx, "logo")
	require.NoError(t, err)
	assert.Equal(t, "logo", found.Key)
	assert.Equal(t, []byte("fake-png-data"), found.FileData)
	assert.Equal(t, "image/png", found.MimeType)

	// Update via upsert.
	asset.FileData = []byte("updated-png-data")
	asset.FileSize = 16
	require.NoError(t, repo.UpsertAsset(ctx, asset))

	found, err = repo.GetAsset(ctx, "logo")
	require.NoError(t, err)
	assert.Equal(t, []byte("updated-png-data"), found.FileData)

	// Delete.
	err = repo.DeleteAsset(ctx, "logo")
	require.NoError(t, err)

	_, err = repo.GetAsset(ctx, "logo")
	assert.Error(t, err)
}

func TestBrandingRepository_GetAsset_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBrandingRepository(db)
	ctx := context.Background()

	_, err := repo.GetAsset(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrDocumentNotFound)
}

func TestBrandingRepository_DeleteAsset_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBrandingRepository(db)
	ctx := context.Background()

	err := repo.DeleteAsset(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrDocumentNotFound)
}

func TestBrandingRepository_ListAssets(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBrandingRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "brand@test.com")

	for _, key := range []string{"logo", "favicon", "logo_dark"} {
		require.NoError(t, repo.UpsertAsset(ctx, &domain.BrandingAsset{
			Key: key, FileData: []byte("data"), MimeType: "image/png",
			FileSize: 4, ChecksumSHA256: "hash", UploadedBy: "u1",
		}))
	}

	assets, err := repo.ListAssets(ctx)
	require.NoError(t, err)
	assert.Len(t, assets, 3)
	// Should be ordered by key.
	assert.Equal(t, "favicon", assets[0].Key)
	assert.Equal(t, "logo", assets[1].Key)
	assert.Equal(t, "logo_dark", assets[2].Key)
	// ListAssets should not include file_data.
	assert.Nil(t, assets[0].FileData)
}

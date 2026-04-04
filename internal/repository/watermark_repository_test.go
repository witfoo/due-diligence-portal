package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

func TestWatermarkRepository_GetConfig_Default(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWatermarkRepository(db)

	config, err := repo.GetConfig(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "default", config.ID)
	assert.False(t, config.Enabled)
	assert.Equal(t, "diagonal", config.Position)
	assert.Equal(t, 0.15, config.Opacity)
	assert.Equal(t, 12, config.FontSize)
	assert.Equal(t, "#888888", config.Color)
	assert.Contains(t, config.TextTemplate, "{{user_email}}")
}

func TestWatermarkRepository_UpsertConfig(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWatermarkRepository(db)
	ctx := context.Background()

	config := &domain.WatermarkConfig{
		Enabled:      true,
		TextTemplate: "Confidential - {{user_email}}",
		Position:     "bottom",
		Opacity:      0.25,
		FontSize:     14,
		Color:        "#ff0000",
	}

	err := repo.UpsertConfig(ctx, config)
	require.NoError(t, err)

	got, err := repo.GetConfig(ctx)
	require.NoError(t, err)
	assert.True(t, got.Enabled)
	assert.Equal(t, "Confidential - {{user_email}}", got.TextTemplate)
	assert.Equal(t, "bottom", got.Position)
	assert.Equal(t, 0.25, got.Opacity)
	assert.Equal(t, 14, got.FontSize)
	assert.Equal(t, "#ff0000", got.Color)
}

func TestWatermarkRepository_UpsertConfig_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWatermarkRepository(db)
	ctx := context.Background()

	// First upsert.
	err := repo.UpsertConfig(ctx, &domain.WatermarkConfig{
		Enabled: true, TextTemplate: "v1", Position: "top",
		Opacity: 0.1, FontSize: 10, Color: "#000",
	})
	require.NoError(t, err)

	// Second upsert (update).
	err = repo.UpsertConfig(ctx, &domain.WatermarkConfig{
		Enabled: false, TextTemplate: "v2", Position: "center",
		Opacity: 0.5, FontSize: 16, Color: "#fff",
	})
	require.NoError(t, err)

	got, err := repo.GetConfig(ctx)
	require.NoError(t, err)
	assert.False(t, got.Enabled)
	assert.Equal(t, "v2", got.TextTemplate)
	assert.Equal(t, "center", got.Position)
}

func TestWatermarkRepository_ResetConfig(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWatermarkRepository(db)
	ctx := context.Background()

	// Modify.
	err := repo.UpsertConfig(ctx, &domain.WatermarkConfig{
		Enabled: true, TextTemplate: "custom", Position: "top",
		Opacity: 0.9, FontSize: 20, Color: "#abc",
	})
	require.NoError(t, err)

	// Reset.
	err = repo.ResetConfig(ctx)
	require.NoError(t, err)

	got, err := repo.GetConfig(ctx)
	require.NoError(t, err)
	assert.False(t, got.Enabled)
	assert.Equal(t, "diagonal", got.Position)
	assert.Equal(t, 0.15, got.Opacity)
	assert.Equal(t, 12, got.FontSize)
	assert.Equal(t, "#888888", got.Color)
}

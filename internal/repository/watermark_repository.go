package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

// WatermarkRepository defines the interface for watermark config data access.
type WatermarkRepository interface {
	GetConfig(ctx context.Context) (*domain.WatermarkConfig, error)
	UpsertConfig(ctx context.Context, config *domain.WatermarkConfig) error
	ResetConfig(ctx context.Context) error
}

type watermarkRepository struct {
	db *DB
}

// NewWatermarkRepository creates a new WatermarkRepository backed by SQLite.
func NewWatermarkRepository(db *DB) WatermarkRepository {
	return &watermarkRepository{db: db}
}

func (r *watermarkRepository) GetConfig(ctx context.Context) (*domain.WatermarkConfig, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, enabled, text_template, position, opacity, font_size, color, updated_at
		 FROM watermark_config WHERE id = 'default'`)

	var wc domain.WatermarkConfig
	var updatedAt string
	err := row.Scan(&wc.ID, &wc.Enabled, &wc.TextTemplate, &wc.Position,
		&wc.Opacity, &wc.FontSize, &wc.Color, &updatedAt)
	if err == sql.ErrNoRows {
		// Return defaults if no row exists.
		return &domain.WatermarkConfig{
			ID:           "default",
			Enabled:      false,
			TextTemplate: "{{user_email}} - {{date}}",
			Position:     "diagonal",
			Opacity:      0.15,
			FontSize:     12,
			Color:        "#888888",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get watermark config: %w", err)
	}
	wc.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &wc, nil
}

func (r *watermarkRepository) UpsertConfig(ctx context.Context, config *domain.WatermarkConfig) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO watermark_config (id, enabled, text_template, position, opacity, font_size, color, updated_at)
		 VALUES ('default', ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   enabled = excluded.enabled,
		   text_template = excluded.text_template,
		   position = excluded.position,
		   opacity = excluded.opacity,
		   font_size = excluded.font_size,
		   color = excluded.color,
		   updated_at = excluded.updated_at`,
		config.Enabled, config.TextTemplate, config.Position,
		config.Opacity, config.FontSize, config.Color, now,
	)
	if err != nil {
		return fmt.Errorf("upsert watermark config: %w", err)
	}
	return nil
}

func (r *watermarkRepository) ResetConfig(ctx context.Context) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`UPDATE watermark_config SET
		   enabled = 0,
		   text_template = '{{user_email}} - {{date}}',
		   position = 'diagonal',
		   opacity = 0.15,
		   font_size = 12,
		   color = '#888888',
		   updated_at = ?
		 WHERE id = 'default'`, now)
	if err != nil {
		return fmt.Errorf("reset watermark config: %w", err)
	}
	return nil
}

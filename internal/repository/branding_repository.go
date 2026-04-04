package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

// BrandingRepository defines the interface for branding data access.
type BrandingRepository interface {
	GetConfig(ctx context.Context) (*domain.BrandingConfig, error)
	UpsertConfig(ctx context.Context, config *domain.BrandingConfig) error
	ResetConfig(ctx context.Context) error
	GetAsset(ctx context.Context, key string) (*domain.BrandingAsset, error)
	UpsertAsset(ctx context.Context, asset *domain.BrandingAsset) error
	DeleteAsset(ctx context.Context, key string) error
	ListAssets(ctx context.Context) ([]*domain.BrandingAsset, error)
}

type brandingRepository struct {
	db *DB
}

// NewBrandingRepository creates a new BrandingRepository backed by SQLite.
func NewBrandingRepository(db *DB) BrandingRepository {
	return &brandingRepository{db: db}
}

func (r *brandingRepository) GetConfig(ctx context.Context) (*domain.BrandingConfig, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, company_name, primary_color, secondary_color, accent_color,
			error_color, warning_color, success_color, info_color,
			background_color, surface_color, text_color, text_secondary_color,
			border_color, hover_color, active_color, header_color, sidebar_color,
			font_family, custom_css, document_title, updated_at
		 FROM branding_config WHERE id = 'default'`)

	var c domain.BrandingConfig
	var fontFamily, customCSS, documentTitle sql.NullString
	var updatedAt string

	err := row.Scan(&c.ID, &c.CompanyName, &c.PrimaryColor, &c.SecondaryColor,
		&c.AccentColor, &c.ErrorColor, &c.WarningColor, &c.SuccessColor,
		&c.InfoColor, &c.BackgroundColor, &c.SurfaceColor, &c.TextColor,
		&c.TextSecondaryColor, &c.BorderColor, &c.HoverColor, &c.ActiveColor,
		&c.HeaderColor, &c.SidebarColor,
		&fontFamily, &customCSS, &documentTitle, &updatedAt)
	if err == sql.ErrNoRows {
		def := domain.DefaultBrandingConfig()
		return &def, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get branding config: %w", err)
	}

	if fontFamily.Valid {
		c.FontFamily = fontFamily.String
	}
	if customCSS.Valid {
		c.CustomCSS = customCSS.String
	}
	if documentTitle.Valid {
		c.DocumentTitle = documentTitle.String
	}
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &c, nil
}

func (r *brandingRepository) UpsertConfig(ctx context.Context, config *domain.BrandingConfig) error {
	now := time.Now().UTC().Format(time.RFC3339)
	config.ID = "default"

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO branding_config (id, company_name, primary_color, secondary_color, accent_color,
			error_color, warning_color, success_color, info_color,
			background_color, surface_color, text_color, text_secondary_color,
			border_color, hover_color, active_color, header_color, sidebar_color,
			font_family, custom_css, document_title, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
			company_name = excluded.company_name,
			primary_color = excluded.primary_color,
			secondary_color = excluded.secondary_color,
			accent_color = excluded.accent_color,
			error_color = excluded.error_color,
			warning_color = excluded.warning_color,
			success_color = excluded.success_color,
			info_color = excluded.info_color,
			background_color = excluded.background_color,
			surface_color = excluded.surface_color,
			text_color = excluded.text_color,
			text_secondary_color = excluded.text_secondary_color,
			border_color = excluded.border_color,
			hover_color = excluded.hover_color,
			active_color = excluded.active_color,
			header_color = excluded.header_color,
			sidebar_color = excluded.sidebar_color,
			font_family = excluded.font_family,
			custom_css = excluded.custom_css,
			document_title = excluded.document_title,
			updated_at = excluded.updated_at`,
		config.ID, config.CompanyName, config.PrimaryColor, config.SecondaryColor,
		config.AccentColor, config.ErrorColor, config.WarningColor, config.SuccessColor,
		config.InfoColor, config.BackgroundColor, config.SurfaceColor, config.TextColor,
		config.TextSecondaryColor, config.BorderColor, config.HoverColor, config.ActiveColor,
		config.HeaderColor, config.SidebarColor,
		nullString(config.FontFamily), nullString(config.CustomCSS), nullString(config.DocumentTitle),
		now,
	)
	if err != nil {
		return fmt.Errorf("upsert branding config: %w", err)
	}
	config.UpdatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *brandingRepository) ResetConfig(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM branding_config WHERE id = 'default'`)
	if err != nil {
		return fmt.Errorf("reset branding config: %w", err)
	}
	return nil
}

func (r *brandingRepository) GetAsset(ctx context.Context, key string) (*domain.BrandingAsset, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT key, file_data, mime_type, file_size, checksum_sha256, uploaded_by, created_at
		 FROM branding_assets WHERE key = ?`, key)

	var a domain.BrandingAsset
	var createdAt string

	err := row.Scan(&a.Key, &a.FileData, &a.MimeType, &a.FileSize,
		&a.ChecksumSHA256, &a.UploadedBy, &createdAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrDocumentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get branding asset: %w", err)
	}

	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &a, nil
}

func (r *brandingRepository) UpsertAsset(ctx context.Context, asset *domain.BrandingAsset) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO branding_assets (key, file_data, mime_type, file_size, checksum_sha256, uploaded_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET
			file_data = excluded.file_data,
			mime_type = excluded.mime_type,
			file_size = excluded.file_size,
			checksum_sha256 = excluded.checksum_sha256,
			uploaded_by = excluded.uploaded_by,
			created_at = excluded.created_at`,
		asset.Key, asset.FileData, asset.MimeType, asset.FileSize,
		asset.ChecksumSHA256, asset.UploadedBy, now,
	)
	if err != nil {
		return fmt.Errorf("upsert branding asset: %w", err)
	}
	asset.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *brandingRepository) DeleteAsset(ctx context.Context, key string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM branding_assets WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("delete branding asset: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrDocumentNotFound
	}
	return nil
}

func (r *brandingRepository) ListAssets(ctx context.Context) ([]*domain.BrandingAsset, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT key, mime_type, file_size, checksum_sha256, uploaded_by, created_at
		 FROM branding_assets ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("list branding assets: %w", err)
	}
	defer rows.Close()

	var assets []*domain.BrandingAsset
	for rows.Next() {
		var a domain.BrandingAsset
		var createdAt string
		if err := rows.Scan(&a.Key, &a.MimeType, &a.FileSize,
			&a.ChecksumSHA256, &a.UploadedBy, &createdAt); err != nil {
			return nil, fmt.Errorf("scan branding asset row: %w", err)
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		assets = append(assets, &a)
	}
	return assets, rows.Err()
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

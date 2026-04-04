package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

// NDARepository defines the interface for NDA data access.
type NDARepository interface {
	CreateTemplate(ctx context.Context, tmpl *domain.NDATemplate) error
	GetTemplate(ctx context.Context, id string) (*domain.NDATemplate, error)
	ListTemplates(ctx context.Context) ([]*domain.NDATemplate, error)
	UpdateTemplate(ctx context.Context, tmpl *domain.NDATemplate) error
	CreateSignature(ctx context.Context, sig *domain.NDASignature) error
	GetSignature(ctx context.Context, id string) (*domain.NDASignature, error)
	ListSignatures(ctx context.Context, templateID string) ([]*domain.NDASignature, error)
	HasSigned(ctx context.Context, userID, templateID string) (bool, error)
}

type ndaRepository struct {
	db *DB
}

// NewNDARepository creates a new NDARepository backed by SQLite.
func NewNDARepository(db *DB) NDARepository {
	return &ndaRepository{db: db}
}

func (r *ndaRepository) CreateTemplate(ctx context.Context, tmpl *domain.NDATemplate) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO nda_templates (id, name, content, is_active, version, created_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		tmpl.ID, tmpl.Name, tmpl.Content, tmpl.IsActive, tmpl.Version, tmpl.CreatedBy, now, now,
	)
	if err != nil {
		return fmt.Errorf("create nda template: %w", err)
	}
	tmpl.CreatedAt, _ = time.Parse(time.RFC3339, now)
	tmpl.UpdatedAt = tmpl.CreatedAt
	return nil
}

func (r *ndaRepository) GetTemplate(ctx context.Context, id string) (*domain.NDATemplate, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, content, is_active, version, created_by, created_at, updated_at
		 FROM nda_templates WHERE id = ?`, id)

	var tmpl domain.NDATemplate
	var createdAt, updatedAt string

	err := row.Scan(&tmpl.ID, &tmpl.Name, &tmpl.Content, &tmpl.IsActive,
		&tmpl.Version, &tmpl.CreatedBy, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrTemplateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get nda template: %w", err)
	}

	tmpl.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	tmpl.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &tmpl, nil
}

func (r *ndaRepository) ListTemplates(ctx context.Context) ([]*domain.NDATemplate, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, content, is_active, version, created_by, created_at, updated_at
		 FROM nda_templates ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list nda templates: %w", err)
	}
	defer rows.Close()

	var templates []*domain.NDATemplate
	for rows.Next() {
		var tmpl domain.NDATemplate
		var createdAt, updatedAt string
		if err := rows.Scan(&tmpl.ID, &tmpl.Name, &tmpl.Content, &tmpl.IsActive,
			&tmpl.Version, &tmpl.CreatedBy, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan nda template row: %w", err)
		}
		tmpl.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		tmpl.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		templates = append(templates, &tmpl)
	}
	return templates, rows.Err()
}

func (r *ndaRepository) UpdateTemplate(ctx context.Context, tmpl *domain.NDATemplate) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE nda_templates SET name = ?, content = ?, is_active = ?, version = ?, updated_at = ?
		 WHERE id = ?`,
		tmpl.Name, tmpl.Content, tmpl.IsActive, tmpl.Version, now, tmpl.ID)
	if err != nil {
		return fmt.Errorf("update nda template (id=%s): %w", tmpl.ID, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrTemplateNotFound
	}
	return nil
}

func (r *ndaRepository) CreateSignature(ctx context.Context, sig *domain.NDASignature) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO nda_signatures (id, template_id, user_id, signer_name, signer_email, signer_company, ip_address, signed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sig.ID, sig.TemplateID, sig.UserID, sig.SignerName, sig.SignerEmail,
		nullString(sig.SignerCompany), sig.IPAddress, now,
	)
	if err != nil {
		return fmt.Errorf("create nda signature: %w", err)
	}
	sig.SignedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *ndaRepository) GetSignature(ctx context.Context, id string) (*domain.NDASignature, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, template_id, user_id, signer_name, signer_email, signer_company, ip_address, signed_at
		 FROM nda_signatures WHERE id = ?`, id)

	var sig domain.NDASignature
	var signerCompany sql.NullString
	var signedAt string

	err := row.Scan(&sig.ID, &sig.TemplateID, &sig.UserID, &sig.SignerName,
		&sig.SignerEmail, &signerCompany, &sig.IPAddress, &signedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrDocumentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get nda signature: %w", err)
	}

	if signerCompany.Valid {
		sig.SignerCompany = signerCompany.String
	}
	sig.SignedAt, _ = time.Parse(time.RFC3339, signedAt)
	return &sig, nil
}

func (r *ndaRepository) ListSignatures(ctx context.Context, templateID string) ([]*domain.NDASignature, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, template_id, user_id, signer_name, signer_email, signer_company, ip_address, signed_at
		 FROM nda_signatures WHERE template_id = ? ORDER BY signed_at DESC`, templateID)
	if err != nil {
		return nil, fmt.Errorf("list nda signatures: %w", err)
	}
	defer rows.Close()

	var sigs []*domain.NDASignature
	for rows.Next() {
		var sig domain.NDASignature
		var signerCompany sql.NullString
		var signedAt string
		if err := rows.Scan(&sig.ID, &sig.TemplateID, &sig.UserID, &sig.SignerName,
			&sig.SignerEmail, &signerCompany, &sig.IPAddress, &signedAt); err != nil {
			return nil, fmt.Errorf("scan nda signature row: %w", err)
		}
		if signerCompany.Valid {
			sig.SignerCompany = signerCompany.String
		}
		sig.SignedAt, _ = time.Parse(time.RFC3339, signedAt)
		sigs = append(sigs, &sig)
	}
	return sigs, rows.Err()
}

func (r *ndaRepository) HasSigned(ctx context.Context, userID, templateID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM nda_signatures WHERE user_id = ? AND template_id = ?`,
		userID, templateID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check nda signature: %w", err)
	}
	return count > 0, nil
}

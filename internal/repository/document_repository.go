package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

// DocumentRepository defines the interface for document data access.
type DocumentRepository interface {
	Create(ctx context.Context, doc *domain.Document) error
	GetByID(ctx context.Context, id string) (*domain.Document, error)
	List(ctx context.Context, categoryID string, limit, offset int) ([]*domain.Document, int, error)
	Update(ctx context.Context, doc *domain.Document) error
	Archive(ctx context.Context, id string) error

	CreateVersion(ctx context.Context, v *domain.DocumentVersion) error
	GetVersion(ctx context.Context, documentID string, versionNumber int) (*domain.DocumentVersion, error)
	ListVersions(ctx context.Context, documentID string) ([]*domain.DocumentVersion, error)

	Search(ctx context.Context, query string, limit, offset int) ([]*domain.Document, int, error)
}

type documentRepository struct {
	db *DB
}

// NewDocumentRepository creates a new DocumentRepository backed by SQLite.
func NewDocumentRepository(db *DB) DocumentRepository {
	return &documentRepository{db: db}
}

func (r *documentRepository) Create(ctx context.Context, doc *domain.Document) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO documents (id, name, description, category_id, uploaded_by, current_version, mime_type, file_size, is_archived, tags, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doc.ID, doc.Name, doc.Description, doc.CategoryID, doc.UploadedBy,
		doc.CurrentVersion, doc.MimeType, doc.FileSize, doc.IsArchived, doc.Tags, now, now,
	)
	if err != nil {
		return fmt.Errorf("create document (id=%s): %w", doc.ID, err)
	}
	doc.CreatedAt, _ = time.Parse(time.RFC3339, now)
	doc.UpdatedAt = doc.CreatedAt
	return nil
}

func (r *documentRepository) GetByID(ctx context.Context, id string) (*domain.Document, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT d.id, d.name, d.description, d.category_id, d.uploaded_by, d.current_version,
		        d.mime_type, d.file_size, d.is_archived, d.tags, d.created_at, d.updated_at,
		        COALESCE(c.name, '') AS category_name, COALESCE(u.name, '') AS uploader_name
		 FROM documents d
		 LEFT JOIN categories c ON c.id = d.category_id
		 LEFT JOIN users u ON u.id = d.uploaded_by
		 WHERE d.id = ?`, id)
	return r.scanDocument(row)
}

func (r *documentRepository) List(ctx context.Context, categoryID string, limit, offset int) ([]*domain.Document, int, error) {
	var total int
	var countErr error
	if categoryID != "" {
		countErr = r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM documents WHERE category_id = ? AND is_archived = 0`, categoryID).Scan(&total)
	} else {
		countErr = r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM documents WHERE is_archived = 0`).Scan(&total)
	}
	if countErr != nil {
		return nil, 0, fmt.Errorf("count documents: %w", countErr)
	}

	var rows *sql.Rows
	var err error
	if categoryID != "" {
		rows, err = r.db.QueryContext(ctx,
			`SELECT d.id, d.name, d.description, d.category_id, d.uploaded_by, d.current_version,
			        d.mime_type, d.file_size, d.is_archived, d.tags, d.created_at, d.updated_at,
			        COALESCE(c.name, '') AS category_name, COALESCE(u.name, '') AS uploader_name
			 FROM documents d
			 LEFT JOIN categories c ON c.id = d.category_id
			 LEFT JOIN users u ON u.id = d.uploaded_by
			 WHERE d.category_id = ? AND d.is_archived = 0
			 ORDER BY d.created_at DESC LIMIT ? OFFSET ?`, categoryID, limit, offset)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT d.id, d.name, d.description, d.category_id, d.uploaded_by, d.current_version,
			        d.mime_type, d.file_size, d.is_archived, d.tags, d.created_at, d.updated_at,
			        COALESCE(c.name, '') AS category_name, COALESCE(u.name, '') AS uploader_name
			 FROM documents d
			 LEFT JOIN categories c ON c.id = d.category_id
			 LEFT JOIN users u ON u.id = d.uploaded_by
			 WHERE d.is_archived = 0
			 ORDER BY d.created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	var docs []*domain.Document
	for rows.Next() {
		d, err := r.scanDocumentRow(rows)
		if err != nil {
			return nil, 0, err
		}
		docs = append(docs, d)
	}
	return docs, total, rows.Err()
}

func (r *documentRepository) Update(ctx context.Context, doc *domain.Document) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE documents SET name = ?, description = ?, category_id = ?, mime_type = ?, file_size = ?,
		        current_version = ?, tags = ?, updated_at = ? WHERE id = ?`,
		doc.Name, doc.Description, doc.CategoryID, doc.MimeType, doc.FileSize,
		doc.CurrentVersion, doc.Tags, now, doc.ID)
	if err != nil {
		return fmt.Errorf("update document (id=%s): %w", doc.ID, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrDocumentNotFound
	}
	return nil
}

func (r *documentRepository) Archive(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE documents SET is_archived = 1, updated_at = ? WHERE id = ?`, now, id)
	if err != nil {
		return fmt.Errorf("archive document (id=%s): %w", id, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrDocumentNotFound
	}
	return nil
}

func (r *documentRepository) CreateVersion(ctx context.Context, v *domain.DocumentVersion) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO document_versions (id, document_id, version_number, file_data, file_size, mime_type, checksum_sha256, change_note, uploaded_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.DocumentID, v.VersionNumber, v.FileData, v.FileSize,
		v.MimeType, v.ChecksumSHA256, v.ChangeNote, v.UploadedBy, now,
	)
	if err != nil {
		return fmt.Errorf("create document version (doc=%s, v=%d): %w", v.DocumentID, v.VersionNumber, err)
	}
	v.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *documentRepository) GetVersion(ctx context.Context, documentID string, versionNumber int) (*domain.DocumentVersion, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, document_id, version_number, file_data, file_size, mime_type, checksum_sha256, change_note, uploaded_by, created_at
		 FROM document_versions WHERE document_id = ? AND version_number = ?`, documentID, versionNumber)

	var v domain.DocumentVersion
	var createdAt string
	var changeNote sql.NullString
	err := row.Scan(&v.ID, &v.DocumentID, &v.VersionNumber, &v.FileData, &v.FileSize,
		&v.MimeType, &v.ChecksumSHA256, &changeNote, &v.UploadedBy, &createdAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrVersionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get document version: %w", err)
	}
	v.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if changeNote.Valid {
		v.ChangeNote = changeNote.String
	}
	return &v, nil
}

func (r *documentRepository) ListVersions(ctx context.Context, documentID string) ([]*domain.DocumentVersion, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, document_id, version_number, file_size, mime_type, checksum_sha256, change_note, uploaded_by, created_at
		 FROM document_versions WHERE document_id = ? ORDER BY version_number DESC`, documentID)
	if err != nil {
		return nil, fmt.Errorf("list document versions (doc=%s): %w", documentID, err)
	}
	defer rows.Close()

	var versions []*domain.DocumentVersion
	for rows.Next() {
		var v domain.DocumentVersion
		var createdAt string
		var changeNote sql.NullString
		err := rows.Scan(&v.ID, &v.DocumentID, &v.VersionNumber, &v.FileSize,
			&v.MimeType, &v.ChecksumSHA256, &changeNote, &v.UploadedBy, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("scan document version row: %w", err)
		}
		v.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if changeNote.Valid {
			v.ChangeNote = changeNote.String
		}
		versions = append(versions, &v)
	}
	return versions, rows.Err()
}

func (r *documentRepository) Search(ctx context.Context, query string, limit, offset int) ([]*domain.Document, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM documents_fts WHERE documents_fts MATCH ?`, query).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count search results: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT d.id, d.name, d.description, d.category_id, d.uploaded_by, d.current_version,
		        d.mime_type, d.file_size, d.is_archived, d.tags, d.created_at, d.updated_at,
		        COALESCE(c.name, '') AS category_name, COALESCE(u.name, '') AS uploader_name
		 FROM documents_fts fts
		 JOIN documents d ON d.rowid = fts.rowid
		 LEFT JOIN categories c ON c.id = d.category_id
		 LEFT JOIN users u ON u.id = d.uploaded_by
		 WHERE documents_fts MATCH ?
		 ORDER BY rank
		 LIMIT ? OFFSET ?`, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("search documents: %w", err)
	}
	defer rows.Close()

	var docs []*domain.Document
	for rows.Next() {
		d, err := r.scanDocumentRow(rows)
		if err != nil {
			return nil, 0, err
		}
		docs = append(docs, d)
	}
	return docs, total, rows.Err()
}

func (r *documentRepository) scanDocument(row *sql.Row) (*domain.Document, error) {
	var d domain.Document
	var description, tags sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(&d.ID, &d.Name, &description, &d.CategoryID, &d.UploadedBy,
		&d.CurrentVersion, &d.MimeType, &d.FileSize, &d.IsArchived, &tags,
		&createdAt, &updatedAt, &d.CategoryName, &d.UploaderName)
	if err == sql.ErrNoRows {
		return nil, domain.ErrDocumentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan document: %w", err)
	}
	if description.Valid {
		d.Description = description.String
	}
	if tags.Valid {
		d.Tags = tags.String
	}
	d.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	d.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &d, nil
}

func (r *documentRepository) scanDocumentRow(row scannable) (*domain.Document, error) {
	var d domain.Document
	var description, tags sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(&d.ID, &d.Name, &description, &d.CategoryID, &d.UploadedBy,
		&d.CurrentVersion, &d.MimeType, &d.FileSize, &d.IsArchived, &tags,
		&createdAt, &updatedAt, &d.CategoryName, &d.UploaderName)
	if err != nil {
		return nil, fmt.Errorf("scan document row: %w", err)
	}
	if description.Valid {
		d.Description = description.String
	}
	if tags.Valid {
		d.Tags = tags.String
	}
	d.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	d.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &d, nil
}

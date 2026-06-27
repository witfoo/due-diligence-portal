package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

// QARepository defines the interface for Q&A data access.
type QARepository interface {
	CreateThread(ctx context.Context, thread *domain.QAThread) error
	GetThread(ctx context.Context, id string) (*domain.QAThread, error)
	// ListThreads returns threads, optionally restricted to a status and/or to a
	// single asker (askedBy != "" scopes results to that user's own threads).
	ListThreads(ctx context.Context, status, askedBy string, limit, offset int) ([]*domain.QAThread, int, error)
	UpdateThreadStatus(ctx context.Context, id, status string) error
	CreateMessage(ctx context.Context, msg *domain.QAMessage) error
	// ListMessages returns a thread's messages; when includeInternal is false,
	// company-internal messages (is_internal = 1) are excluded.
	ListMessages(ctx context.Context, threadID string, includeInternal bool) ([]*domain.QAMessage, error)
}

type qaRepository struct {
	db *DB
}

// NewQARepository creates a new QARepository backed by SQLite.
func NewQARepository(db *DB) QARepository {
	return &qaRepository{db: db}
}

func (r *qaRepository) CreateThread(ctx context.Context, thread *domain.QAThread) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO qa_threads (id, subject, document_id, category_id, status, asked_by, assigned_to, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		thread.ID, thread.Subject, thread.DocumentID, thread.CategoryID, thread.Status,
		thread.AskedBy, thread.AssignedTo, now, now,
	)
	if err != nil {
		return fmt.Errorf("create qa thread: %w", err)
	}
	thread.CreatedAt, _ = time.Parse(time.RFC3339, now)
	thread.UpdatedAt = thread.CreatedAt
	return nil
}

func (r *qaRepository) GetThread(ctx context.Context, id string) (*domain.QAThread, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, subject, document_id, category_id, status, asked_by, assigned_to, created_at, updated_at
		 FROM qa_threads WHERE id = ?`, id)

	var t domain.QAThread
	var documentID, categoryID, assignedTo sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(&t.ID, &t.Subject, &documentID, &categoryID, &t.Status,
		&t.AskedBy, &assignedTo, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrThreadNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get qa thread: %w", err)
	}

	if documentID.Valid {
		t.DocumentID = &documentID.String
	}
	if categoryID.Valid {
		t.CategoryID = &categoryID.String
	}
	if assignedTo.Valid {
		t.AssignedTo = &assignedTo.String
	}
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &t, nil
}

func (r *qaRepository) ListThreads(ctx context.Context, status, askedBy string, limit, offset int) ([]*domain.QAThread, int, error) {
	// Build the WHERE clause from the optional status and askedBy filters.
	var where []string
	var args []any
	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}
	if askedBy != "" {
		where = append(where, "asked_by = ?")
		args = append(args, askedBy)
	}
	whereClause := ""
	if len(where) > 0 {
		whereClause = " WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM qa_threads"+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count qa threads: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, subject, document_id, category_id, status, asked_by, assigned_to, created_at, updated_at
		 FROM qa_threads`+whereClause+` ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list qa threads: %w", err)
	}
	defer rows.Close()

	var threads []*domain.QAThread
	for rows.Next() {
		var t domain.QAThread
		var documentID, categoryID, assignedTo sql.NullString
		var createdAt, updatedAt string

		if err := rows.Scan(&t.ID, &t.Subject, &documentID, &categoryID, &t.Status,
			&t.AskedBy, &assignedTo, &createdAt, &updatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan qa thread row: %w", err)
		}

		if documentID.Valid {
			t.DocumentID = &documentID.String
		}
		if categoryID.Valid {
			t.CategoryID = &categoryID.String
		}
		if assignedTo.Valid {
			t.AssignedTo = &assignedTo.String
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		threads = append(threads, &t)
	}
	return threads, total, rows.Err()
}

func (r *qaRepository) UpdateThreadStatus(ctx context.Context, id, status string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE qa_threads SET status = ?, updated_at = ? WHERE id = ?`, status, now, id)
	if err != nil {
		return fmt.Errorf("update qa thread status (id=%s): %w", id, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrThreadNotFound
	}
	return nil
}

func (r *qaRepository) CreateMessage(ctx context.Context, msg *domain.QAMessage) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO qa_messages (id, thread_id, author_id, body, is_internal, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.ThreadID, msg.AuthorID, msg.Body, msg.IsInternal, now,
	)
	if err != nil {
		return fmt.Errorf("create qa message: %w", err)
	}
	msg.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *qaRepository) ListMessages(ctx context.Context, threadID string, includeInternal bool) ([]*domain.QAMessage, error) {
	query := `SELECT id, thread_id, author_id, body, is_internal, created_at
		 FROM qa_messages WHERE thread_id = ?`
	if !includeInternal {
		query += ` AND is_internal = 0`
	}
	query += ` ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, query, threadID)
	if err != nil {
		return nil, fmt.Errorf("list qa messages: %w", err)
	}
	defer rows.Close()

	var messages []*domain.QAMessage
	for rows.Next() {
		var m domain.QAMessage
		var createdAt string
		if err := rows.Scan(&m.ID, &m.ThreadID, &m.AuthorID, &m.Body, &m.IsInternal, &createdAt); err != nil {
			return nil, fmt.Errorf("scan qa message row: %w", err)
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		messages = append(messages, &m)
	}
	return messages, rows.Err()
}

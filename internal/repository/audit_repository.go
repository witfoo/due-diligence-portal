package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

// AuditFilter defines filtering options for audit log queries.
type AuditFilter struct {
	Action       string
	UserID       string
	ResourceType string
	ResourceID   string
	StartDate    string // RFC3339
	EndDate      string // RFC3339
}

// AuditRepository defines the interface for audit log data access.
type AuditRepository interface {
	Create(ctx context.Context, entry *domain.AuditEntry) error
	List(ctx context.Context, filter AuditFilter, limit, offset int) ([]*domain.AuditEntry, int, error)
	GetByDocument(ctx context.Context, documentID string, limit, offset int) ([]*domain.AuditEntry, int, error)
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.AuditEntry, int, error)
}

type auditRepository struct {
	db *DB
}

// NewAuditRepository creates a new AuditRepository backed by SQLite.
func NewAuditRepository(db *DB) AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Create(ctx context.Context, entry *domain.AuditEntry) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_log (id, user_id, user_email, action, resource_type, resource_id, resource_name, details, ip_address, user_agent, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.UserID, entry.UserEmail, entry.Action,
		entry.ResourceType, entry.ResourceID, entry.ResourceName,
		entry.Details, entry.IPAddress, entry.UserAgent, now,
	)
	if err != nil {
		return fmt.Errorf("create audit entry: %w", err)
	}
	entry.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *auditRepository) List(ctx context.Context, filter AuditFilter, limit, offset int) ([]*domain.AuditEntry, int, error) {
	where, args := r.buildFilterWhere(filter)

	// Count.
	var total int
	countQuery := "SELECT COUNT(*) FROM audit_log" + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit entries: %w", err)
	}

	// Query.
	query := `SELECT id, user_id, user_email, action, resource_type, resource_id, resource_name, details, ip_address, user_agent, created_at
		 FROM audit_log` + where + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	queryArgs := append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit entries: %w", err)
	}
	defer rows.Close()

	var entries []*domain.AuditEntry
	for rows.Next() {
		e, err := r.scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

func (r *auditRepository) GetByDocument(ctx context.Context, documentID string, limit, offset int) ([]*domain.AuditEntry, int, error) {
	return r.List(ctx, AuditFilter{ResourceType: "document", ResourceID: documentID}, limit, offset)
}

func (r *auditRepository) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.AuditEntry, int, error) {
	return r.List(ctx, AuditFilter{UserID: userID}, limit, offset)
}

func (r *auditRepository) buildFilterWhere(filter AuditFilter) (string, []any) {
	var clauses []string
	var args []any

	if filter.Action != "" {
		clauses = append(clauses, "action = ?")
		args = append(args, filter.Action)
	}
	if filter.UserID != "" {
		clauses = append(clauses, "user_id = ?")
		args = append(args, filter.UserID)
	}
	if filter.ResourceType != "" {
		clauses = append(clauses, "resource_type = ?")
		args = append(args, filter.ResourceType)
	}
	if filter.ResourceID != "" {
		clauses = append(clauses, "resource_id = ?")
		args = append(args, filter.ResourceID)
	}
	if filter.StartDate != "" {
		clauses = append(clauses, "created_at >= ?")
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != "" {
		clauses = append(clauses, "created_at <= ?")
		args = append(args, filter.EndDate)
	}

	if len(clauses) == 0 {
		return "", nil
	}

	where := " WHERE "
	for i, c := range clauses {
		if i > 0 {
			where += " AND "
		}
		where += c
	}
	return where, args
}

type auditScannable interface {
	Scan(dest ...any) error
}

func (r *auditRepository) scanRow(row auditScannable) (*domain.AuditEntry, error) {
	var e domain.AuditEntry
	var userID, resourceType, resourceID, resourceName, details, ipAddress, userAgent sql.NullString
	var createdAt string

	err := row.Scan(&e.ID, &userID, &e.UserEmail, &e.Action,
		&resourceType, &resourceID, &resourceName,
		&details, &ipAddress, &userAgent, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("scan audit row: %w", err)
	}

	if userID.Valid {
		e.UserID = userID.String
	}
	if resourceType.Valid {
		e.ResourceType = resourceType.String
	}
	if resourceID.Valid {
		e.ResourceID = resourceID.String
	}
	if resourceName.Valid {
		e.ResourceName = resourceName.String
	}
	if details.Valid {
		e.Details = details.String
	}
	if ipAddress.Valid {
		e.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		e.UserAgent = userAgent.String
	}
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &e, nil
}

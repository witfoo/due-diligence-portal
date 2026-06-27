package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

// PermissionRepository defines the interface for access grant data access.
type PermissionRepository interface {
	Grant(ctx context.Context, grant *domain.AccessGrant) error
	GetByID(ctx context.Context, id string) (*domain.AccessGrant, error)
	ListByUser(ctx context.Context, userID string) ([]*domain.AccessGrant, error)
	ListByResource(ctx context.Context, resourceType, resourceID string) ([]*domain.AccessGrant, error)
	Update(ctx context.Context, grant *domain.AccessGrant) error
	Revoke(ctx context.Context, id string) error
	HasAccess(ctx context.Context, userID, resourceType, resourceID, requiredLevel string) (bool, error)
}

type permissionRepository struct {
	db *DB
}

// NewPermissionRepository creates a new PermissionRepository backed by SQLite.
func NewPermissionRepository(db *DB) PermissionRepository {
	return &permissionRepository{db: db}
}

func (r *permissionRepository) Grant(ctx context.Context, grant *domain.AccessGrant) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var expiresAt *string
	if grant.ExpiresAt != nil {
		s := grant.ExpiresAt.UTC().Format(time.RFC3339)
		expiresAt = &s
	}
	// Upsert: re-granting on a resource the user already has a grant for updates the
	// access level (the common "upgrade view to download") instead of failing the
	// UNIQUE(user_id, resource_type, resource_id) constraint with an opaque 500.
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO access_grants (id, user_id, resource_type, resource_id, access_level, granted_by, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id, resource_type, resource_id)
		 DO UPDATE SET access_level = excluded.access_level,
		               granted_by   = excluded.granted_by,
		               expires_at   = excluded.expires_at`,
		grant.ID, grant.UserID, grant.ResourceType, grant.ResourceID,
		grant.AccessLevel, grant.GrantedBy, expiresAt, now,
	)
	if err != nil {
		return fmt.Errorf("grant access (user=%s, resource=%s/%s): %w",
			grant.UserID, grant.ResourceType, grant.ResourceID, err)
	}
	grant.CreatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (r *permissionRepository) GetByID(ctx context.Context, id string) (*domain.AccessGrant, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT ag.id, ag.user_id, ag.resource_type, ag.resource_id, ag.access_level,
		        ag.granted_by, ag.expires_at, ag.created_at,
		        COALESCE(u.email, '') AS user_email, COALESCE(u.name, '') AS user_name
		 FROM access_grants ag
		 LEFT JOIN users u ON u.id = ag.user_id
		 WHERE ag.id = ?`, id)
	return r.scanGrant(row)
}

func (r *permissionRepository) ListByUser(ctx context.Context, userID string) ([]*domain.AccessGrant, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT ag.id, ag.user_id, ag.resource_type, ag.resource_id, ag.access_level,
		        ag.granted_by, ag.expires_at, ag.created_at,
		        COALESCE(u.email, '') AS user_email, COALESCE(u.name, '') AS user_name
		 FROM access_grants ag
		 LEFT JOIN users u ON u.id = ag.user_id
		 WHERE ag.user_id = ?
		 ORDER BY ag.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list grants by user (user=%s): %w", userID, err)
	}
	defer rows.Close()
	return r.scanGrantRows(rows)
}

func (r *permissionRepository) ListByResource(ctx context.Context, resourceType, resourceID string) ([]*domain.AccessGrant, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT ag.id, ag.user_id, ag.resource_type, ag.resource_id, ag.access_level,
		        ag.granted_by, ag.expires_at, ag.created_at,
		        COALESCE(u.email, '') AS user_email, COALESCE(u.name, '') AS user_name
		 FROM access_grants ag
		 LEFT JOIN users u ON u.id = ag.user_id
		 WHERE ag.resource_type = ? AND ag.resource_id = ?
		 ORDER BY ag.created_at DESC`, resourceType, resourceID)
	if err != nil {
		return nil, fmt.Errorf("list grants by resource (%s/%s): %w", resourceType, resourceID, err)
	}
	defer rows.Close()
	return r.scanGrantRows(rows)
}

func (r *permissionRepository) Update(ctx context.Context, grant *domain.AccessGrant) error {
	var expiresAt *string
	if grant.ExpiresAt != nil {
		s := grant.ExpiresAt.UTC().Format(time.RFC3339)
		expiresAt = &s
	}
	result, err := r.db.ExecContext(ctx,
		`UPDATE access_grants SET access_level = ?, expires_at = ? WHERE id = ?`,
		grant.AccessLevel, expiresAt, grant.ID)
	if err != nil {
		return fmt.Errorf("update access grant (id=%s): %w", grant.ID, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrGrantNotFound
	}
	return nil
}

func (r *permissionRepository) Revoke(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM access_grants WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("revoke access grant (id=%s): %w", id, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrGrantNotFound
	}
	return nil
}

func (r *permissionRepository) HasAccess(ctx context.Context, userID, resourceType, resourceID, requiredLevel string) (bool, error) {
	var accessLevel string
	err := r.db.QueryRowContext(ctx,
		`SELECT access_level FROM access_grants
		 WHERE user_id = ? AND resource_type = ? AND resource_id = ?
		   AND (expires_at IS NULL OR expires_at > ?)`,
		userID, resourceType, resourceID, time.Now().UTC().Format(time.RFC3339)).Scan(&accessLevel)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check access (user=%s, %s/%s): %w", userID, resourceType, resourceID, err)
	}
	return domain.HasAccess(accessLevel, requiredLevel), nil
}

func (r *permissionRepository) scanGrant(row *sql.Row) (*domain.AccessGrant, error) {
	var g domain.AccessGrant
	var expiresAt sql.NullString
	var createdAt string

	err := row.Scan(&g.ID, &g.UserID, &g.ResourceType, &g.ResourceID, &g.AccessLevel,
		&g.GrantedBy, &expiresAt, &createdAt, &g.UserEmail, &g.UserName)
	if err == sql.ErrNoRows {
		return nil, domain.ErrGrantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan access grant: %w", err)
	}
	g.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		g.ExpiresAt = &t
	}
	return &g, nil
}

func (r *permissionRepository) scanGrantRows(rows *sql.Rows) ([]*domain.AccessGrant, error) {
	var grants []*domain.AccessGrant
	for rows.Next() {
		var g domain.AccessGrant
		var expiresAt sql.NullString
		var createdAt string

		err := rows.Scan(&g.ID, &g.UserID, &g.ResourceType, &g.ResourceID, &g.AccessLevel,
			&g.GrantedBy, &expiresAt, &createdAt, &g.UserEmail, &g.UserName)
		if err != nil {
			return nil, fmt.Errorf("scan access grant row: %w", err)
		}
		g.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if expiresAt.Valid {
			t, _ := time.Parse(time.RFC3339, expiresAt.String)
			g.ExpiresAt = &t
		}
		grants = append(grants, &g)
	}
	return grants, rows.Err()
}

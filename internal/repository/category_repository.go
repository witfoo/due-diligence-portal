package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

// CategoryRepository defines the interface for category data access.
type CategoryRepository interface {
	Create(ctx context.Context, cat *domain.Category) error
	GetByID(ctx context.Context, id string) (*domain.Category, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Category, error)
	List(ctx context.Context) ([]*domain.Category, error)
	ListAsTree(ctx context.Context) ([]*domain.Category, error)
	Update(ctx context.Context, cat *domain.Category) error
	Delete(ctx context.Context, id string) error
}

type categoryRepository struct {
	db *DB
}

// NewCategoryRepository creates a new CategoryRepository backed by SQLite.
func NewCategoryRepository(db *DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) Create(ctx context.Context, cat *domain.Category) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO categories (id, name, slug, description, parent_id, sort_order, icon, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cat.ID, cat.Name, cat.Slug, cat.Description, cat.ParentID, cat.SortOrder, cat.Icon, now, now,
	)
	if err != nil {
		return fmt.Errorf("create category (slug=%s): %w", cat.Slug, err)
	}
	cat.CreatedAt, _ = time.Parse(time.RFC3339, now)
	cat.UpdatedAt = cat.CreatedAt
	return nil
}

func (r *categoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, slug, description, parent_id, sort_order, icon, created_at, updated_at
		 FROM categories WHERE id = ?`, id)
	return r.scanCategory(row)
}

func (r *categoryRepository) GetBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, slug, description, parent_id, sort_order, icon, created_at, updated_at
		 FROM categories WHERE slug = ?`, slug)
	return r.scanCategory(row)
}

func (r *categoryRepository) List(ctx context.Context) ([]*domain.Category, error) {
	// Include a per-category count of non-archived documents so list/filter views
	// can show how many documents each category holds and hide empty ones.
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.name, c.slug, c.description, c.parent_id, c.sort_order, c.icon, c.created_at, c.updated_at,
		        (SELECT COUNT(*) FROM documents d WHERE d.category_id = c.id AND d.is_archived = 0) AS document_count
		 FROM categories c ORDER BY c.sort_order, c.name`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var cats []*domain.Category
	for rows.Next() {
		c, err := r.scanCategoryRowWithCount(rows)
		if err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// scanCategoryRowWithCount scans a category row that includes the document_count
// column appended by List.
func (r *categoryRepository) scanCategoryRowWithCount(row scannable) (*domain.Category, error) {
	var c domain.Category
	var description, icon, parentID sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(&c.ID, &c.Name, &c.Slug, &description, &parentID, &c.SortOrder, &icon,
		&createdAt, &updatedAt, &c.DocumentCount)
	if err != nil {
		return nil, fmt.Errorf("scan category row: %w", err)
	}
	if description.Valid {
		c.Description = description.String
	}
	if parentID.Valid {
		c.ParentID = &parentID.String
	}
	if icon.Valid {
		c.Icon = icon.String
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &c, nil
}

func (r *categoryRepository) ListAsTree(ctx context.Context) ([]*domain.Category, error) {
	all, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	return buildTree(all), nil
}

func (r *categoryRepository) Update(ctx context.Context, cat *domain.Category) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE categories SET name = ?, slug = ?, description = ?, parent_id = ?, sort_order = ?, icon = ?, updated_at = ?
		 WHERE id = ?`,
		cat.Name, cat.Slug, cat.Description, cat.ParentID, cat.SortOrder, cat.Icon, now, cat.ID)
	if err != nil {
		return fmt.Errorf("update category (id=%s): %w", cat.ID, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

func (r *categoryRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete category (id=%s): %w", id, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

func (r *categoryRepository) scanCategory(row *sql.Row) (*domain.Category, error) {
	var c domain.Category
	var description, icon sql.NullString
	var parentID sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(&c.ID, &c.Name, &c.Slug, &description, &parentID, &c.SortOrder, &icon, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrCategoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan category: %w", err)
	}
	if description.Valid {
		c.Description = description.String
	}
	if parentID.Valid {
		c.ParentID = &parentID.String
	}
	if icon.Valid {
		c.Icon = icon.String
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &c, nil
}

// buildTree takes a flat list of categories and returns a hierarchical tree.
// Only root categories (parent_id == nil) appear at the top level; children
// are nested under their respective parents.
func buildTree(cats []*domain.Category) []*domain.Category {
	byID := make(map[string]*domain.Category, len(cats))
	for _, c := range cats {
		c.Children = nil // reset
		byID[c.ID] = c
	}

	var roots []*domain.Category
	for _, c := range cats {
		if c.ParentID == nil {
			roots = append(roots, c)
		} else if parent, ok := byID[*c.ParentID]; ok {
			parent.Children = append(parent.Children, c)
		}
	}
	return roots
}

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, limit, offset int) ([]*domain.User, int, error)
	Update(ctx context.Context, user *domain.User) error
	UpdateLastLogin(ctx context.Context, id string) error
	Deactivate(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)

	CreateInviteToken(ctx context.Context, token *domain.InviteToken) error
	GetInviteToken(ctx context.Context, token string) (*domain.InviteToken, error)
	MarkInviteTokenUsed(ctx context.Context, token string) error
}

type userRepository struct {
	db *DB
}

// NewUserRepository creates a new UserRepository backed by SQLite.
func NewUserRepository(db *DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, name, password_hash, role, is_active, invited_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Email, user.Name, user.PasswordHash, user.Role, user.IsActive, user.InvitedBy, now, now,
	)
	if err != nil {
		return fmt.Errorf("create user (email=%s): %w", user.Email, err)
	}
	user.CreatedAt, _ = time.Parse(time.RFC3339, now)
	user.UpdatedAt = user.CreatedAt
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx,
		`SELECT id, email, name, password_hash, role, is_active, invited_by, last_login_at, created_at, updated_at
		 FROM users WHERE id = ?`, id))
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx,
		`SELECT id, email, name, password_hash, role, is_active, invited_by, last_login_at, created_at, updated_at
		 FROM users WHERE email = ?`, email))
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, int, error) {
	total, err := r.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, name, password_hash, role, is_active, invited_by, last_login_at, created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u, err := r.scanUserRow(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET name = ?, email = ?, role = ?, is_active = ?, updated_at = ? WHERE id = ?`,
		user.Name, user.Email, user.Role, user.IsActive, now, user.ID)
	if err != nil {
		return fmt.Errorf("update user (id=%s): %w", user.ID, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ?`, now, now, id)
	if err != nil {
		return fmt.Errorf("update last login (id=%s): %w", id, err)
	}
	return nil
}

func (r *userRepository) Deactivate(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET is_active = 0, updated_at = ? WHERE id = ?`, now, id)
	if err != nil {
		return fmt.Errorf("deactivate user (id=%s): %w", id, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *userRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *userRepository) CreateInviteToken(ctx context.Context, token *domain.InviteToken) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO invite_tokens (id, token, email, role, invited_by, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		token.ID, token.Token, token.Email, token.Role, token.InvitedBy,
		token.ExpiresAt.UTC().Format(time.RFC3339),
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create invite token: %w", err)
	}
	return nil
}

func (r *userRepository) GetInviteToken(ctx context.Context, token string) (*domain.InviteToken, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, token, email, role, invited_by, expires_at, used_at, created_at
		 FROM invite_tokens WHERE token = ?`, token)

	var it domain.InviteToken
	var expiresAt, createdAt string
	var usedAt sql.NullString
	err := row.Scan(&it.ID, &it.Token, &it.Email, &it.Role, &it.InvitedBy, &expiresAt, &usedAt, &createdAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get invite token: %w", err)
	}
	it.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	it.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if usedAt.Valid {
		t, _ := time.Parse(time.RFC3339, usedAt.String)
		it.UsedAt = &t
	}
	return &it, nil
}

// MarkInviteTokenUsed atomically consumes an unused invite token. The conditional
// UPDATE (used_at IS NULL) makes consumption a compare-and-swap: only one concurrent
// caller can win, and a second attempt returns ErrTokenUsed instead of silently
// re-marking an already-used token.
func (r *userRepository) MarkInviteTokenUsed(ctx context.Context, token string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE invite_tokens SET used_at = ? WHERE token = ? AND used_at IS NULL`, now, token)
	if err != nil {
		return fmt.Errorf("mark invite token used: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark invite token used rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrTokenUsed
	}
	return nil
}

func (r *userRepository) scanUser(row *sql.Row) (*domain.User, error) {
	var u domain.User
	var invitedBy sql.NullString
	var lastLoginAt, createdAt, updatedAt sql.NullString

	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Role, &u.IsActive,
		&invitedBy, &lastLoginAt, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}

	if invitedBy.Valid {
		u.InvitedBy = &invitedBy.String
	}
	if lastLoginAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastLoginAt.String)
		u.LastLoginAt = &t
	}
	if createdAt.Valid {
		u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	if updatedAt.Valid {
		u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt.String)
	}
	return &u, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func (r *userRepository) scanUserRow(row scannable) (*domain.User, error) {
	var u domain.User
	var invitedBy sql.NullString
	var lastLoginAt, createdAt, updatedAt sql.NullString

	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Role, &u.IsActive,
		&invitedBy, &lastLoginAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan user row: %w", err)
	}

	if invitedBy.Valid {
		u.InvitedBy = &invitedBy.String
	}
	if lastLoginAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastLoginAt.String)
		u.LastLoginAt = &t
	}
	if createdAt.Valid {
		u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	if updatedAt.Valid {
		u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt.String)
	}
	return &u, nil
}

package middleware

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/repository"
	"github.com/witfoo/due-diligence-portal/pkg/sanitize"
)

// AuditLogger writes audit log entries for authenticated requests.
type AuditLogger struct {
	db *repository.DB
}

// NewAuditLogger creates a new AuditLogger.
func NewAuditLogger(db *repository.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

// Log writes an audit entry.
func (a *AuditLogger) Log(ctx context.Context, entry *domain.AuditEntry) error {
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	// Use NULL for user_id when empty (FK constraint allows NULL but not empty string).
	var userID any
	if entry.UserID != "" {
		userID = entry.UserID
	}
	// Ensure user_email has a value for audit readability.
	if entry.UserEmail == "" {
		entry.UserEmail = "system"
	}
	// Neutralize control characters in all free-text, user-influenced fields so the
	// audit trail cannot be forged with injected newlines/fake key=value pairs (CWE-117).
	_, err := a.db.ExecContext(ctx,
		`INSERT INTO audit_log (id, user_id, user_email, action, resource_type, resource_id, resource_name, details, ip_address, user_agent, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		generateAuditID(), userID, sanitize.LogValue(entry.UserEmail), entry.Action,
		entry.ResourceType, entry.ResourceID, sanitize.LogValue(entry.ResourceName), sanitize.LogValue(entry.Details),
		entry.IPAddress, sanitize.LogValue(entry.UserAgent), entry.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}

// LogFromContext creates and writes an audit entry from the Echo context.
func (a *AuditLogger) LogFromContext(c echo.Context, action, resourceType, resourceID, resourceName, details string) {
	entry := &domain.AuditEntry{
		UserID:       GetUserID(c),
		UserEmail:    GetUserEmail(c),
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Details:      details,
		IPAddress:    c.RealIP(),
		UserAgent:    c.Request().UserAgent(),
	}
	if err := a.Log(c.Request().Context(), entry); err != nil {
		fmt.Printf("[ERROR] Failed to write audit log: %v\n", err)
	}
}

func generateAuditID() string {
	b := make([]byte, 16)
	// Time-based prefix preserves rough insertion ordering...
	binary.BigEndian.PutUint64(b[:8], uint64(time.Now().UnixNano()))
	// ...and a crypto-random suffix guarantees uniqueness so concurrent writes in
	// the same nanosecond cannot collide on the audit_log primary key and be dropped.
	if _, err := rand.Read(b[8:]); err != nil {
		// Extremely unlikely; fall back to a time-derived suffix.
		binary.BigEndian.PutUint64(b[8:], uint64(time.Now().UnixNano()))
	}
	return hex.EncodeToString(b)
}

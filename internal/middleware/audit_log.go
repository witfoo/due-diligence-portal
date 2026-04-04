package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/repository"
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
	_, err := a.db.ExecContext(ctx,
		`INSERT INTO audit_log (id, user_id, user_email, action, resource_type, resource_id, resource_name, details, ip_address, user_agent, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		generateAuditID(), userID, entry.UserEmail, entry.Action,
		entry.ResourceType, entry.ResourceID, entry.ResourceName, entry.Details,
		entry.IPAddress, entry.UserAgent, entry.CreatedAt.Format(time.RFC3339),
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
	// Time-based prefix for ordering + random suffix for uniqueness.
	now := time.Now().UnixNano()
	for i := 0; i < 8; i++ {
		b[i] = byte(now >> (56 - 8*i))
	}
	for i := 8; i < 16; i++ {
		b[i] = byte(now + int64(i))
	}
	return fmt.Sprintf("%x", b)
}

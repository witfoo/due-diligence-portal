package domain

import "time"

// Audit action constants.
const (
	AuditUserLogin       = "user.login"
	AuditUserLogout      = "user.logout"
	AuditUserCreated     = "user.created"
	AuditUserUpdated     = "user.updated"
	AuditUserDeactivated = "user.deactivated"
	AuditUserInvited     = "user.invited"

	AuditDocumentUploaded   = "document.uploaded"
	AuditDocumentViewed     = "document.viewed"
	AuditDocumentDownloaded = "document.downloaded"
	AuditDocumentUpdated    = "document.updated"
	AuditDocumentArchived   = "document.archived"
	AuditDocumentNewVersion = "document.new_version"

	AuditCategoryCreated = "category.created"
	AuditCategoryUpdated = "category.updated"
	AuditCategoryDeleted = "category.deleted"

	AuditPermissionGranted = "permission.granted"
	AuditPermissionRevoked = "permission.revoked"
	AuditPermissionUpdated = "permission.updated"

	AuditQACreated       = "qa.created"
	AuditQAAnswered      = "qa.answered"
	AuditQAClosed        = "qa.closed"
	AuditQAMessagePosted = "qa.message_posted"

	AuditNDASigned  = "nda.signed"
	AuditNDACreated = "nda.created"
	AuditNDAUpdated = "nda.updated"

	AuditBrandingUpdated = "branding.updated"
	AuditBrandingReset   = "branding.reset"
	AuditAssetUploaded   = "asset.uploaded"
	AuditAssetDeleted    = "asset.deleted"
)

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id,omitempty"`
	UserEmail    string    `json:"user_email"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type,omitempty"`
	ResourceID   string    `json:"resource_id,omitempty"`
	ResourceName string    `json:"resource_name,omitempty"`
	Details      string    `json:"details,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

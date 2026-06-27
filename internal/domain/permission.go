package domain

import "time"

// Access level constants (hierarchical: manage > upload > download > view).
const (
	AccessView     = "view"
	AccessDownload = "download"
	AccessUpload   = "upload"
	AccessManage   = "manage"
)

// Resource type constants.
const (
	ResourceCategory = "category"
	ResourceDocument = "document"
)

// validAccessLevels is the set of accepted access levels.
var validAccessLevels = map[string]bool{
	AccessView: true, AccessDownload: true, AccessUpload: true, AccessManage: true,
}

// IsValidAccessLevel reports whether level is one of the accepted access levels.
func IsValidAccessLevel(level string) bool {
	return validAccessLevels[level]
}

// AccessGrant represents a permission grant on a resource.
type AccessGrant struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	ResourceType string     `json:"resource_type"`
	ResourceID   string     `json:"resource_id"`
	AccessLevel  string     `json:"access_level"`
	GrantedBy    string     `json:"granted_by"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`

	// Joined fields.
	UserEmail    string `json:"user_email,omitempty"`
	UserName     string `json:"user_name,omitempty"`
	ResourceName string `json:"resource_name,omitempty"`
}

// accessHierarchy defines which access levels include which others.
var accessHierarchy = map[string][]string{
	AccessManage:   {AccessUpload, AccessDownload, AccessView},
	AccessUpload:   {AccessView},
	AccessDownload: {AccessView},
	AccessView:     {},
}

// HasAccess returns true if the granted level satisfies the required level.
func HasAccess(granted, required string) bool {
	if granted == required {
		return true
	}
	for _, included := range accessHierarchy[granted] {
		if included == required || HasAccess(included, required) {
			return true
		}
	}
	return false
}

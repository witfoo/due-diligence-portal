package domain

import "time"

// Document represents a document's metadata (file content is in DocumentVersion).
type Document struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	CategoryID     string    `json:"category_id"`
	UploadedBy     string    `json:"uploaded_by"`
	CurrentVersion int       `json:"current_version"`
	MimeType       string    `json:"mime_type"`
	FileSize       int64     `json:"file_size"`
	IsArchived     bool      `json:"is_archived"`
	Tags           string    `json:"tags,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Joined fields (not stored directly).
	CategoryName string `json:"category_name,omitempty"`
	UploaderName string `json:"uploader_name,omitempty"`
}

// DocumentVersion represents a specific version of a document including file data.
type DocumentVersion struct {
	ID             string    `json:"id"`
	DocumentID     string    `json:"document_id"`
	VersionNumber  int       `json:"version_number"`
	FileData       []byte    `json:"-"`
	FileSize       int64     `json:"file_size"`
	MimeType       string    `json:"mime_type"`
	ChecksumSHA256 string    `json:"checksum_sha256"`
	ChangeNote     string    `json:"change_note,omitempty"`
	UploadedBy     string    `json:"uploaded_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// Category represents a document category in the category tree.
type Category struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	ParentID    *string   `json:"parent_id,omitempty"`
	SortOrder   int       `json:"sort_order"`
	Icon        string    `json:"icon,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Computed field for tree rendering.
	Children []*Category `json:"children,omitempty"`
}

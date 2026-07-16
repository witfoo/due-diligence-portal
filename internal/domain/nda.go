package domain

import "time"

// NDATemplate represents an NDA document template.
type NDATemplate struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	IsActive  bool      `json:"is_active"`
	Version   int       `json:"version"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NDAExemption waives the NDA click-through requirement for one user. It may
// carry an externally executed NDA document (uploaded by an admin) as evidence.
type NDAExemption struct {
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason,omitempty"`
	FileName  string    `json:"file_name,omitempty"`
	MimeType  string    `json:"mime_type,omitempty"`
	FileSize  int64     `json:"file_size,omitempty"`
	FileData  []byte    `json:"-"`
	GrantedBy string    `json:"granted_by"`
	CreatedAt time.Time `json:"created_at"`

	// Joined fields.
	UserName      string `json:"user_name,omitempty"`
	UserEmail     string `json:"user_email,omitempty"`
	GrantedByName string `json:"granted_by_name,omitempty"`
	HasDocument   bool   `json:"has_document"`
}

// NDASignature represents a user's signature on an NDA.
type NDASignature struct {
	ID            string    `json:"id"`
	TemplateID    string    `json:"template_id"`
	UserID        string    `json:"user_id"`
	SignerName    string    `json:"signer_name"`
	SignerEmail   string    `json:"signer_email"`
	SignerCompany string    `json:"signer_company,omitempty"`
	IPAddress     string    `json:"ip_address"`
	SignedAt      time.Time `json:"signed_at"`

	// Joined fields.
	TemplateName string `json:"template_name,omitempty"`
}

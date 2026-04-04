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

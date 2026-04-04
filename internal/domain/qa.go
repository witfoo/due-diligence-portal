package domain

import "time"

// QA thread status constants.
const (
	QAStatusOpen     = "open"
	QAStatusAnswered = "answered"
	QAStatusClosed   = "closed"
)

// QAThread represents a Q&A conversation thread.
type QAThread struct {
	ID         string     `json:"id"`
	Subject    string     `json:"subject"`
	DocumentID *string    `json:"document_id,omitempty"`
	CategoryID *string    `json:"category_id,omitempty"`
	Status     string     `json:"status"`
	AskedBy    string     `json:"asked_by"`
	AssignedTo *string    `json:"assigned_to,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// Joined fields.
	AskedByName    string `json:"asked_by_name,omitempty"`
	AssignedToName string `json:"assigned_to_name,omitempty"`
	MessageCount   int    `json:"message_count,omitempty"`
}

// QAMessage represents a single message in a Q&A thread.
type QAMessage struct {
	ID         string    `json:"id"`
	ThreadID   string    `json:"thread_id"`
	AuthorID   string    `json:"author_id"`
	Body       string    `json:"body"`
	IsInternal bool      `json:"is_internal"`
	CreatedAt  time.Time `json:"created_at"`

	// Joined fields.
	AuthorName  string `json:"author_name,omitempty"`
	AuthorEmail string `json:"author_email,omitempty"`
	AuthorRole  string `json:"author_role,omitempty"`
}

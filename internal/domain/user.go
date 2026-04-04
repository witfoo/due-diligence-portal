package domain

import (
	"regexp"
	"time"
)

// Role constants.
const (
	RoleAdmin         = "admin"
	RoleCompanyMember = "company_member"
	RoleInvestor      = "investor"
)

// ValidRoles contains all valid user roles.
var ValidRoles = map[string]bool{
	RoleAdmin:         true,
	RoleCompanyMember: true,
	RoleInvestor:      true,
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// User represents a portal user.
type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	IsActive     bool       `json:"is_active"`
	InvitedBy    *string    `json:"invited_by,omitempty"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// InviteToken represents an invitation to join the portal.
type InviteToken struct {
	ID        string     `json:"id"`
	Token     string     `json:"token"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	InvitedBy string     `json:"invited_by"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// ValidateEmail returns an error if the email is invalid.
func ValidateEmail(email string) error {
	if email == "" {
		return ErrEmailRequired
	}
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

// ValidateRole returns an error if the role is invalid.
func ValidateRole(role string) error {
	if !ValidRoles[role] {
		return ErrInvalidRole
	}
	return nil
}

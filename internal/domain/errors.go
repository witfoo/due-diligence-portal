// Package domain contains domain models and sentinel errors for the portal.
package domain

import "errors"

// Validation errors.
var (
	ErrEmailRequired    = errors.New("email is required")
	ErrNameRequired     = errors.New("name is required")
	ErrPasswordRequired = errors.New("password is required")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrInvalidRole      = errors.New("invalid role")
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrSubjectRequired  = errors.New("subject is required")
	ErrBodyRequired     = errors.New("body is required")
	ErrCategoryRequired = errors.New("category is required")
	ErrFileRequired     = errors.New("file is required")
	ErrFileTooLarge     = errors.New("file exceeds maximum size")
	ErrInvalidMimeType  = errors.New("unsupported file type")
)

// Not found errors.
var (
	ErrUserNotFound     = errors.New("user not found")
	ErrDocumentNotFound = errors.New("document not found")
	ErrCategoryNotFound = errors.New("category not found")
	ErrThreadNotFound   = errors.New("thread not found")
	ErrTemplateNotFound = errors.New("template not found")
	ErrGrantNotFound    = errors.New("access grant not found")
	ErrTokenNotFound    = errors.New("invite token not found")
	ErrVersionNotFound  = errors.New("document version not found")
)

// Auth errors.
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenUsed          = errors.New("token has already been used")
	ErrAccountDisabled    = errors.New("account is disabled")
	ErrEmailTaken         = errors.New("email already registered")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("insufficient permissions")
)

// NDA errors.
var (
	ErrNDARequired      = errors.New("NDA signature required")
	ErrNDAAlreadySigned = errors.New("NDA already signed")
)

// Repository errors.
var (
	ErrDatabaseNil = errors.New("database connection is nil")
)

// Package service contains business logic for the portal.
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/repository"
)

const (
	bcryptCost         = 12
	accessTokenExpiry  = 15 * time.Minute
	refreshTokenExpiry = 7 * 24 * time.Hour
)

// JWTClaims contains the custom claims for portal JWT tokens.
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
}

// AuthService handles authentication and token management.
type AuthService struct {
	userRepo  repository.UserRepository
	jwtSecret []byte
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

// LoginResult contains the result of a successful login.
type LoginResult struct {
	User         *domain.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

// Login authenticates a user with email and password.
func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, domain.ErrAccountDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	accessToken, err := s.generateToken(user, accessTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.generateToken(user, refreshTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		// Non-fatal: log but don't fail login.
		fmt.Printf("[WARN] Failed to update last login for %s: %v\n", user.ID, err)
	}

	return &LoginResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Register creates a new user account using an invite token.
func (s *AuthService) Register(ctx context.Context, token, name, password string) (*LoginResult, error) {
	invite, err := s.userRepo.GetInviteToken(ctx, token)
	if err != nil {
		return nil, domain.ErrTokenNotFound
	}

	if invite.UsedAt != nil {
		return nil, domain.ErrTokenUsed
	}

	if time.Now().After(invite.ExpiresAt) {
		return nil, domain.ErrTokenExpired
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	userID, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate user ID: %w", err)
	}

	user := &domain.User{
		ID:           userID,
		Email:        invite.Email,
		Name:         name,
		PasswordHash: hash,
		Role:         invite.Role,
		IsActive:     true,
		InvitedBy:    &invite.InvitedBy,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	if err := s.userRepo.MarkInviteTokenUsed(ctx, token); err != nil {
		fmt.Printf("[WARN] Failed to mark invite token used: %v\n", err)
	}

	return s.Login(ctx, user.Email, password)
}

// RefreshToken generates a new access token from a valid refresh token.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return "", domain.ErrUnauthorized
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return "", domain.ErrUnauthorized
	}

	if !user.IsActive {
		return "", domain.ErrAccountDisabled
	}

	return s.generateToken(user, accessTokenExpiry)
}

// ValidateToken parses and validates a JWT token.
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrUnauthorized
	}

	return claims, nil
}

// CreateInvite generates an invite token for a new user.
func (s *AuthService) CreateInvite(ctx context.Context, email, role, invitedBy string) (*domain.InviteToken, error) {
	if err := domain.ValidateEmail(email); err != nil {
		return nil, err
	}
	if err := domain.ValidateRole(role); err != nil {
		return nil, err
	}

	tokenStr, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate ID: %w", err)
	}

	invite := &domain.InviteToken{
		ID:        id,
		Token:     tokenStr,
		Email:     email,
		Role:      role,
		InvitedBy: invitedBy,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.userRepo.CreateInviteToken(ctx, invite); err != nil {
		return nil, fmt.Errorf("save invite: %w", err)
	}

	return invite, nil
}

// EnsureAdminExists creates the initial admin user if no users exist.
func (s *AuthService) EnsureAdminExists(ctx context.Context, email, password string) error {
	count, err := s.userRepo.Count(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	if password == "" {
		raw := make([]byte, 16)
		if _, err := rand.Read(raw); err != nil {
			return fmt.Errorf("generate random password: %w", err)
		}
		password = hex.EncodeToString(raw)
		fmt.Printf("[INFO] Generated admin password: %s\n", password)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	id, err := generateID()
	if err != nil {
		return fmt.Errorf("generate ID: %w", err)
	}

	admin := &domain.User{
		ID:           id,
		Email:        email,
		Name:         "Administrator",
		PasswordHash: hash,
		Role:         domain.RoleAdmin,
		IsActive:     true,
	}

	if err := s.userRepo.Create(ctx, admin); err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	fmt.Printf("[INFO] Admin user created: %s\n", email)
	return nil
}

func (s *AuthService) generateToken(user *domain.User, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "dd-portal",
		},
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
		Role:   user.Role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// HashPassword hashes a password using bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(hash), nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

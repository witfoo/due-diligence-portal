package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/witfoo/due-diligence-portal/internal/domain"
	"github.com/witfoo/due-diligence-portal/internal/repository"
)

// generateHandlerID generates a random hex ID for new resources.
func generateHandlerID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// documentAccessAllowed reports whether a user with the given role may access doc
// at the required level. Staff (admin/company member) always may; other roles
// (investors) need a matching grant on the document OR on its category. This is the
// single source of truth for per-document authorization across handlers.
func documentAccessAllowed(ctx context.Context, permRepo repository.PermissionRepository, userID, role string, doc *domain.Document, level string) (bool, error) {
	if role == domain.RoleAdmin || role == domain.RoleCompanyMember {
		return true, nil
	}
	if ok, err := permRepo.HasAccess(ctx, userID, domain.ResourceDocument, doc.ID, level); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	if doc.CategoryID == "" {
		return false, nil
	}
	return permRepo.HasAccess(ctx, userID, domain.ResourceCategory, doc.CategoryID, level)
}

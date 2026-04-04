package handler

import (
	"crypto/rand"
	"encoding/hex"
)

// generateHandlerID generates a random hex ID for new resources.
func generateHandlerID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

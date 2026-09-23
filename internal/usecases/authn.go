package usecases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/leomichalski/aigateway/internal/domain"
	"github.com/leomichalski/aigateway/internal/port/inbound"
	"github.com/leomichalski/aigateway/internal/port/outbound"
)

type authNUseCase struct {
	repo outbound.IdentityRepository
}

// NewAuthNUseCase creates a new authentication use case.
func NewAuthNUseCase(repo outbound.IdentityRepository) inbound.AuthNUseCase {
	return &authNUseCase{
		repo: repo,
	}
}

func (u *authNUseCase) Authenticate(ctx context.Context, rawKey string) (domain.Identity, error) {
	// Hash the raw key using SHA-256
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	// Lookup via repository using the hash, keeping the raw key out of logs/DB
	return u.repo.GetIdentityByAPIKey(ctx, keyHash)
}

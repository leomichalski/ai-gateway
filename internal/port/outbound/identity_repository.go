package outbound

import (
	"context"
	"github.com/leomichalski/aigateway/internal/domain"
)

// IdentityRepository defines how we interact with the data store for identities.
type IdentityRepository interface {
	GetIdentityByAPIKey(ctx context.Context, keyHash string) (domain.Identity, error)
}

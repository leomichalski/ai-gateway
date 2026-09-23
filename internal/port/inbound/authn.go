package inbound

import (
	"context"

	"github.com/leomichalski/aigateway/internal/domain"
)

// AuthNUseCase defines the boundary for authentication logic.
type AuthNUseCase interface {
	Authenticate(ctx context.Context, rawKey string) (domain.Identity, error)
}

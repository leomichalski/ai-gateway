package inbound

import (
	"context"
	"github.com/leomichalski/aigateway/internal/domain"
)

type AuthZUseCase interface {
	Authorize(ctx context.Context, identity domain.Identity, object string, action string) error
}

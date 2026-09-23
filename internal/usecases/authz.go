package usecases

import (
	"context"
	"log/slog"
	"github.com/leomichalski/aigateway/internal/domain"
	"github.com/leomichalski/aigateway/internal/port/inbound"
	"github.com/leomichalski/aigateway/internal/port/outbound"
)

type authZUseCase struct {
	enforcer outbound.PermissionEnforcer
}

func NewAuthZUseCase(enforcer outbound.PermissionEnforcer) inbound.AuthZUseCase {
	return &authZUseCase{enforcer: enforcer}
}

func (u *authZUseCase) Authorize(ctx context.Context, identity domain.Identity, object string, action string) error {
	// Action is strictly "use" for our LLM model requests
	allowed, err := u.enforcer.CheckAccess(ctx, identity.TenantID, object, action)
	slog.Debug(identity.TenantID, object, action)
	if err != nil {
		return domain.ErrEnforcerFailed
	}

	if !allowed {
		return domain.ErrForbidden
	}

	return nil
}
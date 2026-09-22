package outbound

import (
	"context"

	"github.com/leomichalski/aigateway/internal/domain"
)

type AgentRunner interface {
	ExecuteTask(ctx context.Context, secretProvider SecretProvider, sessionID string, prompt string) (*domain.ChatResult, error)
}

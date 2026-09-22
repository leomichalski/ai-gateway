package outbound

import (
	"context"

	"github.com/leomichalski/aigateway/internal/domain"
)

type AgentRunner interface {
	ExecuteTask(ctx context.Context, secretProvider SecretProvider, chatRequest domain.ChatRequest) (*domain.ChatResult, error)
}

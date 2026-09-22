package outbound

import (
	"context"

	"github.com/leomichalski/aigateway/internal/domain"
)

type LLMProvider interface {
	GenerateCompletion(ctx context.Context, secretProvider SecretProvider, chatRequest domain.ChatRequest) (*domain.ChatResult, error)
}

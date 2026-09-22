package outbound

import (
	"context"
	"iter"

	"github.com/leomichalski/aigateway/internal/domain"
)

type LLMProvider interface {
	GenerateCompletion(ctx context.Context, secretProvider SecretProvider, chatRequest domain.ChatRequest) (*domain.ChatResult, error)
	StreamCompletion(ctx context.Context, secretProvider SecretProvider, chatRequest domain.ChatRequest) (iter.Seq2[*domain.ChatChunk, error], error)
}

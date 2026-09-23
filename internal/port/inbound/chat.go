package inbound

import (
	"context"
	"iter"

	"github.com/leomichalski/aigateway/internal/domain"
)

// ChatUseCase defines the inbound operations available for chat interactions.
type ChatUseCase interface {
	ProcessChat(ctx context.Context, chatRequest domain.ChatRequest) (*domain.ChatResult, error)
	ProcessChatStream(ctx context.Context, chatRequest domain.ChatRequest) (iter.Seq2[*domain.ChatChunk, error], error)
}
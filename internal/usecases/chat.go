package usecases

import (
	"context"
	"iter"

	"github.com/leomichalski/aigateway/internal/domain"
	"github.com/leomichalski/aigateway/internal/port/inbound"
	"github.com/leomichalski/aigateway/internal/port/outbound"
)

// chatUseCase is the concrete implementation of inbound.ChatUseCase.
// Notice it is unexported (lowercase) to force the use of the interface.
type chatUseCase struct {
	agentRunner    outbound.LLMProvider
	secretProvider outbound.SecretProvider
}

// NewChatUseCase instantiates the use case and returns the inbound port interface.
func NewChatUseCase(llmProvider outbound.LLMProvider, secretProvider outbound.SecretProvider) inbound.ChatUseCase {
	return &chatUseCase{
		agentRunner:    llmProvider,
		secretProvider: secretProvider,
	}
}

func (uc *chatUseCase) ProcessChat(ctx context.Context, chatRequest domain.ChatRequest) (*domain.ChatResult, error) {
	// TODO: business rules like checking the semantic cache before calling LLMs
	return uc.agentRunner.GenerateCompletion(ctx, uc.secretProvider, chatRequest)
}

func (uc *chatUseCase) ProcessChatStream(ctx context.Context, chatRequest domain.ChatRequest) (iter.Seq2[*domain.ChatChunk, error], error) {
	// TODO: business rules like checking the semantic cache before calling LLMs
	return uc.agentRunner.StreamCompletion(ctx, uc.secretProvider, chatRequest)
}

package usecases

import (
	"context"
	"iter"

	"github.com/leomichalski/aigateway/internal/domain"
	"github.com/leomichalski/aigateway/internal/port/outbound"
)

type ChatUseCase struct {
	agentRunner    outbound.LLMProvider
	secretProvider outbound.SecretProvider
}

func NewChatUseCase(LLMProvider outbound.LLMProvider, SecretProvider outbound.SecretProvider) *ChatUseCase {
	return &ChatUseCase{
		agentRunner:    LLMProvider,
		secretProvider: SecretProvider,
	}
}

// func (uc *ChatUseCase) ProcessChat(ctx context.Context, sessionID, modelName string, prompt string) (*domain.ChatResult, error) {
func (uc *ChatUseCase) ProcessChat(ctx context.Context, chatRequest domain.ChatRequest) (*domain.ChatResult, error) {
	// TODO: business rules like checking the semantic cache before calling LLMs
	return uc.agentRunner.GenerateCompletion(ctx, uc.secretProvider, chatRequest)
	// return uc.agentRunner.GenerateCompletion(ctx, uc.secretProvider, sessionID, modelName, prompt)
}

func (uc *ChatUseCase) ProcessChatStream(ctx context.Context, chatRequest domain.ChatRequest) (iter.Seq2[*domain.ChatChunk, error], error) {
	// TODO: business rules like checking the semantic cache before calling LLMs
	return uc.agentRunner.StreamCompletion(ctx, uc.secretProvider, chatRequest)
}

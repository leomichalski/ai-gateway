package usecases

import (
	"context"

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

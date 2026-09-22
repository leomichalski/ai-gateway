package usecases

import (
	"context"

	"github.com/leomichalski/aigateway/internal/domain"
	"github.com/leomichalski/aigateway/internal/port/outbound"
)

type ChatUseCase struct {
	agentRunner outbound.AgentRunner
	secretProvider outbound.SecretProvider
}

func NewChatUseCase(AgentRunner outbound.AgentRunner, SecretProvider outbound.SecretProvider) *ChatUseCase {
	return &ChatUseCase{
		agentRunner: AgentRunner,
		secretProvider: SecretProvider,
	}
}

// func (uc *ChatUseCase) ProcessChat(ctx context.Context, sessionID, modelName string, prompt string) (*domain.ChatResult, error) {
func (uc *ChatUseCase) ProcessChat(ctx context.Context, chatRequest domain.ChatRequest) (*domain.ChatResult, error) {
	// TODO: business rules like checking the semantic cache before calling LLMs
	return uc.agentRunner.ExecuteTask(ctx, uc.secretProvider, chatRequest)
	// return uc.agentRunner.ExecuteTask(ctx, uc.secretProvider, sessionID, modelName, prompt)
}

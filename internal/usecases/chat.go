package usecases

import (
	"context"

	"github.com/leomichalski/surveil-api/internal/domain"
	"github.com/leomichalski/surveil-api/internal/port/outbound"
)

type ChatUseCase struct {
	agentRunner outbound.AgentRunner
}

func NewChatUseCase(runner outbound.AgentRunner) *ChatUseCase {
	return &ChatUseCase{
		agentRunner: runner,
	}
}

func (uc *ChatUseCase) ProcessChat(ctx context.Context, sessionID, prompt string) (*domain.ChatResult, error) {
	// TODO: business rules like checking the semantic cache before calling LLMs
	return uc.agentRunner.ExecuteTask(ctx, sessionID, prompt)
}

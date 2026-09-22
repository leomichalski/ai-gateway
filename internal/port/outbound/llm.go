package outbound

import (
	"context"

	"github.com/leomichalski/surveil-api/internal/domain"
)

type AgentRunner interface {
	ExecuteTask(ctx context.Context, sessionID string, prompt string) (*domain.ChatResult, error)
}

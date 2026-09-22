package http

import (
	"github.com/leomichalski/aigateway/internal/usecases"
)

type APIHandler struct {
	chatUseCase *usecases.ChatUseCase
	// mu    sync.RWMutex
	// tasks map[string]Task
}

func NewAPIHandler(uc *usecases.ChatUseCase) *APIHandler {
	return &APIHandler{
		chatUseCase: uc,
		// tasks: make(map[string]Task),
	}
}
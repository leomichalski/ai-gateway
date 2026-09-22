package http

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/leomichalski/aigateway/internal/domain"
)

// -----------------------------------------------------------------------------
// POST /v1/chat/completions
// -----------------------------------------------------------------------------
func (s *APIHandler) PostV1ChatCompletion(ctx context.Context, req PostV1ChatCompletionRequestObject) (PostV1ChatCompletionResponseObject, error) {
	// session := ""
	// if req.Body.SessionId != nil {
	// 	session = *req.Body.SessionId
	// }
	// result, err := s.chatUseCase.ProcessChat(ctx, session, req.Body.Model, req.Body.Prompt)


	// Map DTO to Domain Entity
	var messages []domain.Message
	for _, msg := range req.Body.Messages {
		messages = append(messages, domain.Message{
			Role:    string(msg.Role),
			Content: msg.Content, // Assuming text content for now
		})
	}

	stream := false
	if req.Body.Stream != nil {
		stream = *req.Body.Stream
	}

	domainReq := domain.ChatRequest{
		Model:       req.Body.Model,
		Messages:    messages,
		Temperature: req.Body.Temperature,
		Stream:      stream,
	}

	// Execute Use Case
	result, err := s.chatUseCase.ProcessChat(ctx, domainReq)
	if err != nil {
		// Log error internally, but return the standard OpenAI Error schema
		return PostV1ChatCompletion500JSONResponse{
			Error: ErrorObject{
				Message: err.Error(),
				Type:    "internal_server_error",
			},
		}, nil
	}


	// 3. Map Domain Result to OpenAI DTO Response
	// Generate a unique ID and Timestamp for the response
	responseID := "chatcmpl-" + uuid.New().String()
	createdTS := time.Now().Unix()

	return PostV1ChatCompletion200JSONResponse{
		Id:      responseID,
		Object:  "chat.completion",
		Created: createdTS,
		Model:   req.Body.Model, // Echo back the requested model
		Choices: []ChatCompletionChoice{
			{
				Index: 0,
				Message: ChatCompletionResponseMessage{
					Role:    "assistant",
					Content: result.Answer,
				},
				FinishReason: ChatCompletionChoiceFinishReason(result.FinishReason),
			},
		},
		Usage: ChatCompletionUsage{
			PromptTokens:     result.PromptTokens,
			CompletionTokens: result.CompletionTokens,
			TotalTokens:      result.TotalTokens,
		},
	}, nil
}

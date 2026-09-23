package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"iter"
	"net/http"
	"time"
	"errors"

	"github.com/google/uuid"
	"github.com/leomichalski/aigateway/internal/domain"
)


// mapDomainToErrorResponse translates pure domain errors into oapi-codegen responses.
func mapDomainToErrorResponse(err error) (string, string) {
	switch {
	case errors.Is(err, domain.ErrInvalidRequest):
		return "invalid_request_error", err.Error()

	case errors.Is(err, domain.ErrClientCanceled):
		return "client_canceled", "The request was canceled by the client"

	case errors.Is(err, domain.ErrGatewayTimeout):
		return "gateway_timeout", "The upstream AI provider timed out"

	case errors.Is(err, domain.ErrProviderFailure):
		return "upstream_provider_error", "The upstream AI provider failed to process the request"

	case errors.Is(err, domain.ErrSecretNotFound), errors.Is(err, domain.ErrSecretNotSet):
		return "configuration_error", "The requested AI provider is not properly configured on this gateway"

	default:
		return "internal_server_error", "An unexpected error occurred"
	}
}

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

	if domainReq.Stream {
		streamSeq, err := s.chatUseCase.ProcessChatStream(ctx, domainReq)
		if err != nil {
			errType, errMsg := mapDomainToErrorResponse(err)
			return PostV1ChatCompletion500JSONResponse{
				Error: ErrorObject{Message: errMsg, Type: errType},
			}, nil
		}

		// 3. Return our custom response struct cleanly
		return ChatCompletionSSEResponse{
			StreamSeq: streamSeq,
			Model:     req.Body.Model,
		}, nil
	}

	// Execute Use Case
	result, err := s.chatUseCase.ProcessChat(ctx, domainReq)
	if err != nil {
		errType, errMsg := mapDomainToErrorResponse(err)
		// Log error internally, but return the standard OpenAI Error schema
		return PostV1ChatCompletion500JSONResponse{
			Error: ErrorObject{
				Message: errMsg,
				Type:    errType,
			},
		}, nil
	}

	// Map Domain Result to OpenAI DTO Response
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

type ChatCompletionSSEResponse struct {
	StreamSeq iter.Seq2[*domain.ChatChunk, error]
	Model     string
}

// Implement the oapi-codegen ResponseObject interface
func (r ChatCompletionSSEResponse) VisitPostV1ChatCompletionResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// Type assert the ResponseWriter to http.Flusher
	flusher, _ := w.(http.Flusher)

	chatID := "chatcmpl-" + uuid.New().String()
	created := time.Now().Unix()

	for chunk, err := range r.StreamSeq {
		if err != nil {
			slog.Debug("Failed while streaming chat completion response.", "error", err)

			errType, errMsg := mapDomainToErrorResponse(err)
			// Send the error as a JSON event to the client
			errorPayload := map[string]any{
				"error": map[string]any{"message": errMsg, "type": errType},
			}
			errData, _ := json.Marshal(errorPayload)
			fmt.Fprintf(w, "data: %s\n\n", string(errData))
			if flusher != nil { flusher.Flush() }
			// Optional: Write an SSE error chunk here
			break
		}

		var finishReason any = nil
		if chunk.FinishReason != "" {
			finishReason = chunk.FinishReason
		}

		openaiChunk := map[string]any{
			"id":      chatID,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   r.Model,
			"choices": []map[string]any{
				{
					"index": 0,
					"delta": map[string]any{
						"content": chunk.ContentDelta,
					},
					"finish_reason": finishReason,
				},
			},
		}

		data, _ := json.Marshal(openaiChunk)
		fmt.Fprintf(w, "data: %s\n\n", string(data))
		if flusher != nil {
			flusher.Flush()
		}
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}

	return nil
}

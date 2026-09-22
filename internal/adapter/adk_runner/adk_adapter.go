package adk_runner

import (
	"context"
	"log/slog"
	"strings"
	"time"
	"iter"
	"errors"

	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/gemini"
	"google.golang.org/genai"

	// "google.golang.org/adk/v2/tool"
	// "google.golang.org/adk/v2/tool/geminitool"

	"github.com/leomichalski/aigateway/internal/domain"
	"github.com/leomichalski/aigateway/internal/port/outbound"
)

// Ensure struct implements the interface at compile time
var _ outbound.LLMProvider = (*GoogleADKAdapter)(nil)

type GoogleADKAdapter struct {
	// apiKey string, client *adk.Client, etc.
}

func NewGoogleADKAdapter() outbound.LLMProvider {
	// adkClient: adk.NewClient(os.Getenv("ADK_API_KEY")),
	return &GoogleADKAdapter{}
}


// mapError translates provider/context errors into clean Domain errors.
func (a *GoogleADKAdapter) mapError(err error, fallbackErr error) error {
	if err == nil {
		return nil
	}
	
	switch {
	case errors.Is(err, context.Canceled):
		slog.Info("Operation aborted by client (context canceled)")
		return domain.ErrClientCanceled
	case errors.Is(err, context.DeadlineExceeded):
		slog.Warn("Operation timed out (deadline exceeded)")
		return domain.ErrGatewayTimeout
	default:
		slog.Debug("Provider operation failed", "error", err)
		return fallbackErr
	}
}

// Implementa a porta LLMProvider exigida pelo Gateway
func (a *GoogleADKAdapter) GenerateCompletion(ctx context.Context, secretProvider outbound.SecretProvider, chatRequest domain.ChatRequest) (*domain.ChatResult, error) {
	// slog.Debug("Executing ADK task", slog.String("session_id", sessionID))
	// Enforce timeouts to protect the Gateway (Circuit Breaker baseline)

	// TODO: parameterize timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	key, err := secretProvider.GetProviderKey(ctx, "gemini")
	if err != nil {
		return nil, a.mapError(err, domain.ErrInternalSecret)
	}

	llmModel, err := gemini.NewModel(ctx, chatRequest.Model, &genai.ClientConfig{
		APIKey: key,
	})
	if err != nil {
		return nil, a.mapError(err, domain.ErrModelInitialization)
	}

	var contents []*genai.Content
	for _, msg := range chatRequest.Messages {
		var role genai.Role
		switch msg.Role {
		// case "system":
		// 	role = genai.RoleSystem
		case "user":
			role = genai.RoleUser
		case "assistant":
			role = genai.RoleModel // Gemini uses 'model' for the assistant role
		default:
			role = genai.RoleUser
		}
		contents = append(contents, genai.NewContentFromText(msg.Content, role))
	}

	var genConfig *genai.GenerateContentConfig
	if chatRequest.Temperature != nil {
		genConfig = &genai.GenerateContentConfig{
			Temperature: chatRequest.Temperature,
		}
	}
	// Direct Stateless Generation
	llmReq := &model.LLMRequest{
		Model:    chatRequest.Model,
		Contents: contents,
		Config:   genConfig,
		// Tools:
	}

	var lastResp *model.LLMResponse = nil
	for resp, err := range llmModel.GenerateContent(ctx, llmReq, false) {  // stream = false
		if err != nil {
			return nil, a.mapError(err, domain.ErrProviderFailure)
		}
		lastResp = resp
	}

	if lastResp == nil || lastResp.Content == nil {
		return nil, domain.ErrEmptyProviderResponse
	}

	// Extract the response text
	var responseBuilder strings.Builder
	for _, part := range lastResp.Content.Parts {
		if part.Text != "" {
			responseBuilder.WriteString(part.Text)
		}
	}

	// Extract Usage metadata
	var (
		promptTokens int32
		compTokens   int32
		totalTokens  int32
	)
	if lastResp.UsageMetadata != nil {
		promptTokens = lastResp.UsageMetadata.PromptTokenCount
		compTokens = lastResp.UsageMetadata.CandidatesTokenCount
		totalTokens = lastResp.UsageMetadata.TotalTokenCount
	}

	// Determine Finish Reason based on ADK's last response flag
	finishReason := "stop"
	if lastResp.FinishReason != "" {
		finishReason = strings.ToLower(string(lastResp.FinishReason))
	}
	return &domain.ChatResult{
		Answer:           responseBuilder.String(),
		PromptTokens:     promptTokens,
		CompletionTokens: compTokens,
		TotalTokens:      totalTokens,
		Provider:         "google-adk",
		FinishReason:     finishReason,
	}, nil
}

// Helper to normalize ADK finish reasons to OpenAI compatible finish reasons
func mapFinishReason(reason string) string {
	if reason == "" {
		return ""
	}
	switch strings.ToUpper(reason) {
	case "MAX_TOKENS", "LENGTH":
		return "length"
	case "SAFETY", "RECITATION", "BLOCKLIST":
		return "content_filter"
	default:
		return "stop"
	}
}

func (a *GoogleADKAdapter) StreamCompletion(ctx context.Context, secretProvider outbound.SecretProvider, chatRequest domain.ChatRequest) (iter.Seq2[*domain.ChatChunk, error], error) {
	// slog.Debug("Executing ADK task", slog.String("session_id", sessionID))
	// Enforce timeouts to protect the Gateway (Circuit Breaker baseline)

	// TODO: parameterize timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	// defer cancel()

	key, err := secretProvider.GetProviderKey(ctx, "gemini")
	if err != nil {
		cancel()
		return nil, a.mapError(err, domain.ErrInternalSecret)
	}

	llmModel, err := gemini.NewModel(ctx, chatRequest.Model, &genai.ClientConfig{
		APIKey: key,
	})
	if err != nil {
		return nil, a.mapError(err, domain.ErrModelInitialization)
	}

	var contents []*genai.Content
	for _, msg := range chatRequest.Messages {
		var role genai.Role
		switch msg.Role {
		// case "system":
		// 	role = genai.RoleSystem
		case "user":
			role = genai.RoleUser
		case "assistant":
			role = genai.RoleModel // Gemini uses 'model' for the assistant role
		default:
			role = genai.RoleUser
		}
		contents = append(contents, genai.NewContentFromText(msg.Content, role))
	}

	var genConfig *genai.GenerateContentConfig
	if chatRequest.Temperature != nil {
		genConfig = &genai.GenerateContentConfig{
			Temperature: chatRequest.Temperature,
		}
	}
	// Direct Stateless Generation
	llmReq := &model.LLMRequest{
		Model:    chatRequest.Model,
		Contents: contents,
		Config:   genConfig,
		// Tools:
	}

	// var lastResp *model.LLMResponse = nil
	seq := func(yield func(*domain.ChatChunk, error) bool) {
		defer cancel()
		
		for resp, err := range llmModel.GenerateContent(ctx, llmReq, true) { // stream = true
			if err != nil {
				yield(nil, a.mapError(err, domain.ErrProviderFailure))
				return
			}

			var textDelta string
			if resp.Content != nil {
				for _, part := range resp.Content.Parts {
					textDelta += part.Text
				}
			}

			chunk := &domain.ChatChunk{
				ContentDelta: textDelta,
			}
			
			if resp.FinishReason != "" {
				chunk.FinishReason = mapFinishReason(string(resp.FinishReason))
			}

			if !yield(chunk, nil) {
				return
			}
		}
	}

	return seq, nil
}

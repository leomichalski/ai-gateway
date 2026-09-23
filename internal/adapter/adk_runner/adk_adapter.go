package adk_runner

import (
	"context"
	"errors"
	"iter"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/gemini"
	"google.golang.org/genai"

	"github.com/leomichalski/aigateway/internal/domain"
	"github.com/leomichalski/aigateway/internal/port/outbound"
)

const (
	geminiProvider = "gemini"
	googleProvider = "google-adk"
	requestTimeout = 120 * time.Second
)

// Ensure struct implements the interface at compile time.
var _ outbound.LLMProvider = (*GoogleADKAdapter)(nil)

type GoogleADKAdapter struct{}

func NewGoogleADKAdapter() outbound.LLMProvider {
	return &GoogleADKAdapter{}
}

// mapError translates provider/context errors into domain errors.
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

// prepareRequest performs all provider-independent request preparation:
// timeout, API key retrieval, model initialization, and request construction.
func (a *GoogleADKAdapter) prepareRequest(
	ctx context.Context,
	secretProvider outbound.SecretProvider,
	chatRequest domain.ChatRequest,
) (context.Context, context.CancelFunc, model.LLM, *model.LLMRequest, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)

	key, err := secretProvider.GetProviderKey(ctx, geminiProvider)
	if err != nil {
		cancel()
		return nil, nil, nil, nil, a.mapError(err, domain.ErrInternalSecret)
	}

	llmModel, err := gemini.NewModel(ctx, chatRequest.Model, &genai.ClientConfig{
		APIKey: key,
	})
	if err != nil {
		cancel()
		return nil, nil, nil, nil, a.mapError(err, domain.ErrModelInitialization)
	}

	llmReq := buildLLMRequest(chatRequest)

	return ctx, cancel, llmModel, llmReq, nil
}

func buildLLMRequest(chatRequest domain.ChatRequest) *model.LLMRequest {
	return &model.LLMRequest{
		Model:    chatRequest.Model,
		Contents: buildContents(chatRequest.Messages),
		Config:   buildGenerateConfig(chatRequest),
	}
}

func buildContents(messages []domain.Message) []*genai.Content {
	contents := make([]*genai.Content, 0, len(messages))

	for _, msg := range messages {
		role := mapRole(msg.Role)
		contents = append(
			contents,
			genai.NewContentFromText(msg.Content, role),
		)
	}

	return contents
}

func mapRole(role string) genai.Role {
	switch role {
	// case "system":
	// 	fallback to genai.RoleUser, there's no genai.RoleSystem
	case "assistant":
		return genai.RoleModel
	case "user":
		return genai.RoleUser
	default:
		return genai.RoleUser
	}
}

func buildGenerateConfig(chatRequest domain.ChatRequest) *genai.GenerateContentConfig {
	if chatRequest.Temperature == nil {
		return nil
	}
	return &genai.GenerateContentConfig{
		Temperature: chatRequest.Temperature,
	}
}

func (a *GoogleADKAdapter) GenerateCompletion(
	ctx context.Context,
	secretProvider outbound.SecretProvider,
	chatRequest domain.ChatRequest,
) (*domain.ChatResult, error) {
	ctx, cancel, llmModel, llmReq, err := a.prepareRequest(
		ctx,
		secretProvider,
		chatRequest,
	)
	if err != nil {
		return nil, err
	}
	defer cancel()

	var lastResp *model.LLMResponse

	for resp, err := range llmModel.GenerateContent(ctx, llmReq, false) {
		if err != nil {
			return nil, a.mapError(err, domain.ErrProviderFailure)
		}

		lastResp = resp
	}

	if lastResp == nil || lastResp.Content == nil {
		return nil, domain.ErrEmptyProviderResponse
	}

	return buildChatResult(lastResp), nil
}

func buildChatResult(resp *model.LLMResponse) *domain.ChatResult {
	var (
		answer        strings.Builder
		promptTokens  int32
		compTokens    int32
		totalTokens   int32
	)

	for _, part := range resp.Content.Parts {
		if part.Text != "" {
			answer.WriteString(part.Text)
		}
	}

	if resp.UsageMetadata != nil {
		promptTokens = resp.UsageMetadata.PromptTokenCount
		compTokens = resp.UsageMetadata.CandidatesTokenCount
		totalTokens = resp.UsageMetadata.TotalTokenCount
	}

	return &domain.ChatResult{
		Answer:          answer.String(),
		PromptTokens:    promptTokens,
		CompletionTokens: compTokens,
		TotalTokens:     totalTokens,
		Provider:        googleProvider,
		FinishReason:    mapFinishReason(string(resp.FinishReason)),
	}
}

func (a *GoogleADKAdapter) StreamCompletion(
	ctx context.Context,
	secretProvider outbound.SecretProvider,
	chatRequest domain.ChatRequest,
) (iter.Seq2[*domain.ChatChunk, error], error) {
	ctx, cancel, llmModel, llmReq, err := a.prepareRequest(
		ctx,
		secretProvider,
		chatRequest,
	)
	if err != nil {
		return nil, err
	}

	seq := func(yield func(*domain.ChatChunk, error) bool) {
		defer cancel()

		for resp, err := range llmModel.GenerateContent(ctx, llmReq, true) {
			if err != nil {
				yield(nil, a.mapError(err, domain.ErrProviderFailure))
				return
			}

			chunk := buildChatChunk(resp)

			if !yield(chunk, nil) {
				return
			}
		}
	}

	return seq, nil
}

func buildChatChunk(resp *model.LLMResponse) *domain.ChatChunk {
	var content strings.Builder

	if resp.Content != nil {
		for _, part := range resp.Content.Parts {
			content.WriteString(part.Text)
		}
	}

	chunk := &domain.ChatChunk{
		ContentDelta: content.String(),
	}

	if resp.FinishReason != "" {
		chunk.FinishReason = mapFinishReason(string(resp.FinishReason))
	}

	return chunk
}

// mapFinishReason normalizes ADK/Gemini finish reasons to
// OpenAI-compatible finish reasons.
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
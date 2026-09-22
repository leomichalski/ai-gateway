package adk_runner

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/gemini"
	"google.golang.org/genai"

	// "google.golang.org/adk/v2/tool"
	// "google.golang.org/adk/v2/tool/geminitool"

	"github.com/leomichalski/aigateway/internal/domain"
	"github.com/leomichalski/aigateway/internal/port/outbound"
)

// Ensure struct implements the interface at compile time
var _ outbound.AgentRunner = (*GoogleADKAdapter)(nil)

type GoogleADKAdapter struct {
	// apiKey string, client *adk.Client, etc.
}

func NewGoogleADKAdapter() outbound.AgentRunner {
	// adkClient: adk.NewClient(os.Getenv("ADK_API_KEY")),
	return &GoogleADKAdapter{}
}

// Implementa a porta AgentRunner exigida pelo Gateway
func (a *GoogleADKAdapter) ExecuteTask(ctx context.Context, secretProvider outbound.SecretProvider, chatRequest domain.ChatRequest) (*domain.ChatResult, error) {
	// slog.Debug("Executing ADK task", slog.String("session_id", sessionID))
	// Enforce timeouts to protect the Gateway (Circuit Breaker baseline)

	// TODO: parameterize timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	key, err := secretProvider.GetProviderKey(ctx, "gemini")
	if err != nil {
		// slog.Error("%s", err)
		return nil, err
	}

	llmModel, err := gemini.NewModel(ctx, chatRequest.Model, &genai.ClientConfig{
		APIKey: key,
	})
	if err != nil {
		slog.Error("Failed to create model: %v", err)
		return nil, err
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
		Config: genConfig,
		// Tools:
	}

	var lastResp *model.LLMResponse = nil
	for resp, err := range llmModel.GenerateContent(ctx, llmReq, false) {
		if err != nil {
			return nil, err
		}
		lastResp = resp
	}

	// Extract the response text
	var responseBuilder strings.Builder
	if lastResp != nil && lastResp.Content != nil {
		for _, part := range lastResp.Content.Parts {
			if part.Text != "" {
				responseBuilder.WriteString(part.Text)
			}
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

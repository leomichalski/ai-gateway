package adk_runner

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model/gemini"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
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
func (a *GoogleADKAdapter) ExecuteTask(ctx context.Context, secretProvider outbound.SecretProvider, sessionID string, prompt string) (*domain.ChatResult, error) {
	slog.Debug("Executing ADK task", slog.String("session_id", sessionID))
	// Enforce timeouts to protect the Gateway (Circuit Breaker baseline)

	// TODO: parameterize timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	key, err := secretProvider.GetProviderKey(ctx, "gemini")
	if err != nil {
		// slog.Error("%s", err)
		return nil, err
	}

	// TODO: refactor when there is model storage
	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{
		APIKey: key,
	})
	if err != nil {
		slog.Error("Failed to create model: %v", err)
		return nil, err
	}

	// TODO: refactor when there is agent storage
	currentAgent, err := llmagent.New(llmagent.Config{
		Name:        "current_agent",
		Model:       model,
		Description: "Qual é a capital do Brasil?",
		Instruction: "Você é um guia de turismo brasileiro.",
	})
	if err != nil {
		slog.Error("Failed to create agent: %v", err)
		return nil, err
	}

	// Create an in-memory session service and instantiate a session
	appName := "aigateway"
	// TODO: refactor when there are users
	userID := "user-123"

	sessionService := session.InMemoryService()
	sessResp, err := sessionService.Create(ctx, &session.CreateRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		slog.Error("Failed to create session: %v", err)
		return nil, err

	}

	// Initialize the Runner (replacing launcher.Run)
	r, err := runner.New(runner.Config{
		AppName:        appName,
		Agent:          currentAgent,
		SessionService: sessionService,
	})
	if err != nil {
		slog.Error("Failed to create runner: %v", err)
		return nil, err

	}

	// Construct the user input message
	msg := genai.NewContentFromText(prompt, genai.RoleUser)

	// Run the agent and capture the response into a variable
	var responseBuilder strings.Builder
	var (
		promptTokens     int32
		candidatesTokens int32
		totalTokens      int32
	)

	for event, err := range r.Run(ctx, userID, sessResp.Session.ID(), msg, agent.RunConfig{}) {
		if err != nil {
			slog.Error("Error during agent run: %v", err)
			return nil, err
		}

		// Streamed parts can contain text, tool calls, or other payloads.
		if event.Content != nil {
			for _, part := range event.Content.Parts {
				if part.Text != "" {
					responseBuilder.WriteString(part.Text)
				}
			}
		}
		// Accumulate token counts
		if event.UsageMetadata != nil {
			promptTokens += event.UsageMetadata.PromptTokenCount
			candidatesTokens += event.UsageMetadata.CandidatesTokenCount
			totalTokens += event.UsageMetadata.TotalTokenCount
		}
	}

	result := &domain.ChatResult{
		Answer:     responseBuilder.String(),
		TokensUsed: totalTokens, // TODO: change
		Provider:   "google-adk",
	}
	return result, nil
}

package adk_runner

import (
	// "context"
	// "errors"
	"testing"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"

	"github.com/leomichalski/aigateway/internal/domain"
)

func TestBuildGenerateConfig(t *testing.T) {
	t.Run("nil temperature", func(t *testing.T) {
		req := domain.ChatRequest{}

		got := buildGenerateConfig(req)

		if got != nil {
			t.Fatalf("buildGenerateConfig() = %#v, want nil", got)
		}
	})

	t.Run("temperature is configured", func(t *testing.T) {
		temperature := float32(0.7)

		req := domain.ChatRequest{
			Temperature: &temperature,
		}

		got := buildGenerateConfig(req)

		if got == nil {
			t.Fatal("buildGenerateConfig() = nil, want config")
		}

		if got.Temperature == nil {
			t.Fatal("Temperature = nil, want configured temperature")
		}

		if *got.Temperature != temperature {
			t.Fatalf(
				"Temperature = %v, want %v",
				*got.Temperature,
				temperature,
			)
		}
	})
}

func TestBuildContents(t *testing.T) {
	messages := []domain.Message{
		{
			Role:    "user",
			Content: "Hello",
		},
		{
			Role:    "assistant",
			Content: "Hi there!",
		},
		{
			Role:    "system",
			Content: "You are helpful.",
		},
	}

	got := buildContents(messages)

	if len(got) != len(messages) {
		t.Fatalf(
			"buildContents() returned %d contents, want %d",
			len(got),
			len(messages),
		)
	}

	tests := []struct {
		index int
		role  genai.Role
		text  string
	}{
		{
			index: 0,
			role:  genai.RoleUser,
			text:  "Hello",
		},
		{
			index: 1,
			role:  genai.RoleModel,
			text:  "Hi there!",
		},
		{
			index: 2,
			role:  genai.RoleUser,
			text:  "You are helpful.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			content := got[tt.index]

			if content == nil {
				t.Fatal("content = nil")
			}

			if content.Role != string(tt.role) {
				t.Fatalf(
					"content.Role = %q, want %q",
					content.Role,
					tt.role,
				)
			}

			if len(content.Parts) != 1 {
				t.Fatalf(
					"len(content.Parts) = %d, want 1",
					len(content.Parts),
				)
			}

			if content.Parts[0].Text != tt.text {
				t.Fatalf(
					"content text = %q, want %q",
					content.Parts[0].Text,
					tt.text,
				)
			}
		})
	}
}

func TestBuildChatResult(t *testing.T) {
	resp := &model.LLMResponse{
		Content: &genai.Content{
			Parts: []*genai.Part{
				{
					Text: "Hello ",
				},
				{
					Text: "world!",
				},
				{
					Text: "",
				},
			},
		},
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 20,
			TotalTokenCount:      30,
		},
		FinishReason: "STOP",
	}

	got := buildChatResult(resp)

	if got == nil {
		t.Fatal("buildChatResult() = nil")
	}

	if got.Answer != "Hello world!" {
		t.Fatalf(
			"Answer = %q, want %q",
			got.Answer,
			"Hello world!",
		)
	}

	if got.PromptTokens != 10 {
		t.Fatalf(
			"PromptTokens = %d, want 10",
			got.PromptTokens,
		)
	}

	if got.CompletionTokens != 20 {
		t.Fatalf(
			"CompletionTokens = %d, want 20",
			got.CompletionTokens,
		)
	}

	if got.TotalTokens != 30 {
		t.Fatalf(
			"TotalTokens = %d, want 30",
			got.TotalTokens,
		)
	}

	if got.Provider != googleProvider {
		t.Fatalf(
			"Provider = %q, want %q",
			got.Provider,
			googleProvider,
		)
	}

	if got.FinishReason != "stop" {
		t.Fatalf(
			"FinishReason = %q, want %q",
			got.FinishReason,
			"stop",
		)
	}
}

func TestBuildChatResultWithoutUsageMetadata(t *testing.T) {
	resp := &model.LLMResponse{
		Content: &genai.Content{
			Parts: []*genai.Part{
				{
					Text: "Hello",
				},
			},
		},
	}

	got := buildChatResult(resp)

	if got == nil {
		t.Fatal("buildChatResult() = nil")
	}

	if got.Answer != "Hello" {
		t.Fatalf(
			"Answer = %q, want %q",
			got.Answer,
			"Hello",
		)
	}

	if got.PromptTokens != 0 {
		t.Fatalf("PromptTokens = %d, want 0", got.PromptTokens)
	}

	if got.CompletionTokens != 0 {
		t.Fatalf("CompletionTokens = %d, want 0", got.CompletionTokens)
	}

	if got.TotalTokens != 0 {
		t.Fatalf("TotalTokens = %d, want 0", got.TotalTokens)
	}

	if got.FinishReason != "" {
		t.Fatalf(
			"FinishReason = %q, want empty",
			got.FinishReason,
		)
	}
}

func TestBuildChatChunk(t *testing.T) {
	resp := &model.LLMResponse{
		Content: &genai.Content{
			Parts: []*genai.Part{
				{
					Text: "Hello ",
				},
				{
					Text: "world!",
				},
			},
		},
		FinishReason: "MAX_TOKENS",
	}

	got := buildChatChunk(resp)

	if got == nil {
		t.Fatal("buildChatChunk() = nil")
	}

	if got.ContentDelta != "Hello world!" {
		t.Fatalf(
			"ContentDelta = %q, want %q",
			got.ContentDelta,
			"Hello world!",
		)
	}

	if got.FinishReason != "length" {
		t.Fatalf(
			"FinishReason = %q, want %q",
			got.FinishReason,
			"length",
		)
	}
}

func TestBuildChatChunkWithoutContent(t *testing.T) {
	resp := &model.LLMResponse{
		FinishReason: "SAFETY",
	}

	got := buildChatChunk(resp)

	if got == nil {
		t.Fatal("buildChatChunk() = nil")
	}

	if got.ContentDelta != "" {
		t.Fatalf(
			"ContentDelta = %q, want empty",
			got.ContentDelta,
		)
	}

	if got.FinishReason != "content_filter" {
		t.Fatalf(
			"FinishReason = %q, want %q",
			got.FinishReason,
			"content_filter",
		)
	}
}

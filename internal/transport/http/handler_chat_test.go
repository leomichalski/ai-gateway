package api

import (
	"bufio"
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/leomichalski/aigateway/internal/domain"
	"github.com/leomichalski/aigateway/internal/port/outbound"
	"github.com/leomichalski/aigateway/internal/usecases"
)

type fakeLLMProvider struct{}

var _ outbound.LLMProvider = (*fakeLLMProvider)(nil)

func (f *fakeLLMProvider) GenerateCompletion(
	context.Context,
	outbound.SecretProvider,
	domain.ChatRequest,
) (*domain.ChatResult, error) {
	return &domain.ChatResult{
		Answer:           "Brasília",
		PromptTokens:     5,
		CompletionTokens: 2,
		TotalTokens:      7,
		FinishReason:     "stop",
	}, nil
}

func (f *fakeLLMProvider) StreamCompletion(
	context.Context,
	outbound.SecretProvider,
	domain.ChatRequest,
) (iter.Seq2[*domain.ChatChunk, error], error) {
	return func(yield func(*domain.ChatChunk, error) bool) {
		if !yield(&domain.ChatChunk{ContentDelta: "Bras"}, nil) {
			return
		}
		yield(&domain.ChatChunk{ContentDelta: "ília", FinishReason: "stop"}, nil)
	}, nil
}

func newChatTestClient(t *testing.T) *ClientWithResponses {
	t.Helper()

	e := echo.New()
	e.Use(middleware.Recover())

	chatUC := usecases.NewChatUseCase(&fakeLLMProvider{}, nil)
	chatHandler := NewChatHandler(chatUC)
	apiHandler := NewAPIHandler(chatHandler)
	strictHandler := NewStrictHandler(apiHandler, nil)
	RegisterHandlers(e, strictHandler)

	server := httptest.NewServer(e)
	t.Cleanup(server.Close)

	client, err := NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatalf("create API client: %v", err)
	}

	return client
}

func TestPostV1ChatCompletion(t *testing.T) {
	client := newChatTestClient(t)
	modelName := "gemini-3.5-flash-lite"
	body := PostV1ChatCompletionJSONRequestBody{
		Model: modelName,
		Messages: []ChatCompletionRequestMessage{
			{
				Role:    "user",
				Content: "What is the capital of Brazil?",
			},
		},
	}

	resp, err := client.PostV1ChatCompletionWithResponse(context.Background(), body)
	if err != nil {
		t.Fatalf("post chat completion: %v", err)
	}
	if got, want := resp.StatusCode(), http.StatusOK; got != want {
		t.Fatalf("status code = %d, want %d", got, want)
	}
	if resp.JSON200 == nil {
		t.Fatal("expected an OpenAI-compatible completion response")
	}
	if got, want := resp.JSON200.Model, modelName; got != want {
		t.Errorf("model = %q, want %q", got, want)
	}
	if len(resp.JSON200.Choices) != 1 {
		t.Fatalf("choices count = %d, want 1", len(resp.JSON200.Choices))
	}

	choice := resp.JSON200.Choices[0]
	if !strings.Contains(choice.Message.Content, "Brasília") {
		t.Errorf("completion content = %q, want it to contain %q", choice.Message.Content, "Brasília")
	}
	if got, want := string(choice.FinishReason), "stop"; got != want {
		t.Errorf("finish reason = %q, want %q", got, want)
	}
	if resp.JSON200.Usage.TotalTokens <= 0 {
		t.Errorf("total tokens = %d, want > 0", resp.JSON200.Usage.TotalTokens)
	}
}

func TestPostV1ChatCompletionStream(t *testing.T) {
	client := newChatTestClient(t)
	modelName := "gemini-3.5-flash-lite"
	stream := true
	body := PostV1ChatCompletionJSONRequestBody{
		Model: modelName,
		Messages: []ChatCompletionRequestMessage{
			{
				Role:    "user",
				Content: "What is the capital of Brazil?",
			},
		},
		Stream: &stream,
	}

	resp, err := client.PostV1ChatCompletion(context.Background(), body)
	if err != nil {
		t.Fatalf("post streaming chat completion: %v", err)
	}
	defer resp.Body.Close()

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Fatalf("status code = %d, want %d", got, want)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("content type = %q, want text/event-stream", contentType)
	}

	scanner := bufio.NewScanner(resp.Body)
	var fullAnswer string
	var foundDone bool

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			t.Fatalf("SSE line = %q, want data prefix", line)
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			foundDone = true
			break
		}

		var chunk struct {
			Object  string `json:"object"`
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			t.Fatalf("unmarshal SSE chunk %q: %v", data, err)
		}
		if got, want := chunk.Object, "chat.completion.chunk"; got != want {
			t.Errorf("chunk object = %q, want %q", got, want)
		}
		if got, want := chunk.Model, modelName; got != want {
			t.Errorf("chunk model = %q, want %q", got, want)
		}
		if len(chunk.Choices) != 1 {
			t.Fatalf("chunk choices count = %d, want 1", len(chunk.Choices))
		}

		fullAnswer += chunk.Choices[0].Delta.Content
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("read SSE response: %v", err)
	}
	if !foundDone {
		t.Fatal("expected data: [DONE] event")
	}
	if !strings.Contains(fullAnswer, "Brasília") {
		t.Errorf("streamed completion content = %q, want it to contain %q", fullAnswer, "Brasília")
	}
}

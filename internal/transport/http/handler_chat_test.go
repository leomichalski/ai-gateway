package api

import (
	"bufio"
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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

var _ = Describe("Chat API Endpoint", func() {
	var (
		server *httptest.Server
		client *ClientWithResponses
	)

	BeforeEach(func() {
		e := echo.New()
		e.Use(middleware.Recover())

		chatUC := usecases.NewChatUseCase(&fakeLLMProvider{}, nil)
		chatHandler := NewChatHandler(chatUC)
		apiHandler := NewAPIHandler(chatHandler)
		strictHandler := NewStrictHandler(apiHandler, nil)
		RegisterHandlers(e, strictHandler)

		server = httptest.NewServer(e)

		client, _ = NewClientWithResponses(server.URL)
	})

	AfterEach(func() {
		server.Close()
	})

	Context("When I send a valid prompt", func() {
		It("must return 200 OK with an OpenAI compatible answer", func() {
			modelName := "gemini-3.5-flash-lite"
			body := PostV1ChatCompletionJSONRequestBody{
				Model: modelName,
				Messages: []ChatCompletionRequestMessage{
					{
						Role:    "user",
						Content: "Qual a capital do Brasil?",
					},
				},
			}

			ctx := context.Background()
			resp, err := client.PostV1ChatCompletionWithResponse(ctx, body)

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode()).To(Equal(http.StatusOK))
			Expect(resp.JSON200).NotTo(BeNil())

			Expect(resp.JSON200.Choices).NotTo(BeEmpty())
			Expect(resp.JSON200.Choices[0].Message.Content).To(ContainSubstring("Brasília"))
			Expect(string(resp.JSON200.Choices[0].FinishReason)).To(Equal("stop"))

			Expect(resp.JSON200.Model).To(Equal(modelName))
			Expect(resp.JSON200.Usage.TotalTokens).To(BeNumerically(">", 0))
		})
	})

	Context("When I send a valid prompt with stream=true", func() {
		It("must return 200 OK and stream OpenAI compatible Server-Sent Events", func() {
			modelName := "gemini-3.5-flash-lite"
			streamFlag := true
			body := PostV1ChatCompletionJSONRequestBody{
				Model: modelName,
				Messages: []ChatCompletionRequestMessage{
					{
						Role:    "user",
						Content: "Qual a capital do Brasil?",
					},
				},
				Stream: &streamFlag,
			}

			ctx := context.Background()
			resp, err := client.PostV1ChatCompletion(ctx, body)

			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("Content-Type")).To(ContainSubstring("text/event-stream"))

			scanner := bufio.NewScanner(resp.Body)
			var fullAnswer string
			var foundDone bool

			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					continue
				}

				Expect(line).To(HavePrefix("data: "))
				dataStr := strings.TrimPrefix(line, "data: ")

				if dataStr == "[DONE]" {
					foundDone = true
					break
				}

				var chunk map[string]any
				err := json.Unmarshal([]byte(dataStr), &chunk)
				Expect(err).NotTo(HaveOccurred(), "Failed to parse chunk JSON")

				Expect(chunk["object"]).To(Equal("chat.completion.chunk"))
				Expect(chunk["model"]).To(Equal(modelName))

				choices, ok := chunk["choices"].([]any)
				Expect(ok).To(BeTrue())
				Expect(choices).NotTo(BeEmpty())

				choice := choices[0].(map[string]any)
				delta := choice["delta"].(map[string]any)

				if content, ok := delta["content"].(string); ok {
					fullAnswer += content
				}
			}

			Expect(scanner.Err()).NotTo(HaveOccurred())
			Expect(foundDone).To(BeTrue(), "Expected to receive data: [DONE]")
			Expect(fullAnswer).To(ContainSubstring("Brasília"))
		})
	})
})

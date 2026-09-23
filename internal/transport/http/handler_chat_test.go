package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"bufio"
	"strings"
	"encoding/json"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/leomichalski/aigateway/internal/adapter/adk_runner"
	"github.com/leomichalski/aigateway/internal/adapter/secrets"
	"github.com/leomichalski/aigateway/internal/usecases"
)

var _ = Describe("Chat API Endpoint", func() {
	var (
		server *httptest.Server
		client *ClientWithResponses
		chatUC *usecases.ChatUseCase
	)

	BeforeEach(func() {
		e := echo.New()
		e.Use(middleware.Recover())
		// e.Use(middleware.Logger())
		// Inicialize com Mocks (para testar regras, não a rede real)
		// mockAdapter := mocks.NewMockLLMProvider()
		// mockUC = usecases.NewChatUseCase(mockAdapter)
		// apiHandler := NewAPIHandler(mockUC)

		llmAdapter := adk_runner.NewGoogleADKAdapter()
		envSecretProvider := secrets.NewEnvSecretProvider()
		chatUC = usecases.NewChatUseCase(llmAdapter, envSecretProvider)
		// Initialize Sub-Handlers
		chatHandler := NewChatHandler(chatUC)
		// Assemble the Root API Handler
		apiHandler := NewAPIHandler(chatHandler)
		// Register with oapi-codegen and Echo
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

			// ASSERT (Ginkgo/Gomega)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode()).To(Equal(http.StatusOK))
			Expect(resp.JSON200).NotTo(BeNil())

			// OpenAI compatibility validation
			Expect(resp.JSON200.Choices).NotTo(BeEmpty())
			Expect(resp.JSON200.Choices[0].Message.Content).To(ContainSubstring("Brasília"))
			Expect(string(resp.JSON200.Choices[0].FinishReason)).To(Equal("stop"))

			// Metadata
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
			
			// Use the base client method (without WithResponse) to read the raw HTTP stream
			resp, err := client.PostV1ChatCompletion(ctx, body)

			// ASSERT (Ginkgo/Gomega)
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
					continue // Skip empty lines between SSE events
				}

				Expect(line).To(HavePrefix("data: "))
				dataStr := strings.TrimPrefix(line, "data: ")

				if dataStr == "[DONE]" {
					foundDone = true
					break
				}

				// Unmarshal the OpenAI chunk
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

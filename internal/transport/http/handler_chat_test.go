package http

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

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
		// mockAdapter := mocks.NewMockAgentRunner()
		// mockUC = usecases.NewChatUseCase(mockAdapter)
		// apiHandler := NewAPIHandler(mockUC)

		llmAdapter := adk_runner.NewGoogleADKAdapter()
		envSecretProvider := secrets.NewEnvSecretProvider()
		chatUC = usecases.NewChatUseCase(llmAdapter, envSecretProvider)
		apiHandler := NewAPIHandler(chatUC) 
		
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
			// ARRANGE
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

			// ACT - Call the API using the typed client (based on operationId: PostV1ChatCompletion)
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
})
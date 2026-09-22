package http

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/labstack/echo/v5"

	"github.com/leomichalski/surveil-api/internal/adapter/adk_runner"
	"github.com/leomichalski/surveil-api/internal/usecases"
)


var _ = Describe("Chat API Endpoint", func() {
	var (
		server *httptest.Server
		client *ClientWithResponses // O Client gerado pelo OpenAPI
		chatUC *usecases.ChatUseCase              // Seu UseCase mockado
	)

	BeforeEach(func() {
		// 1. Configura o servidor do Echo em memória
		e := echo.New()
		
		// Inicialize com Mocks (para testar regras, não a rede real)
		// mockAdapter := mocks.NewMockAgentRunner()
		// mockUC = usecases.NewChatUseCase(mockAdapter)
		// apiHandler := NewAPIHandler(mockUC)

		llmAdapter := adk_runner.NewGoogleADKAdapter()
		chatUC = usecases.NewChatUseCase(llmAdapter)
		apiHandler := NewAPIHandler(chatUC) 
		
		strictHandler := NewStrictHandler(apiHandler, nil)
		RegisterHandlers(e, strictHandler)

		server = httptest.NewServer(e)

		// 2. Cria o Client tipado do OpenAPI apontando pro servidor de teste
		client, _ = NewClientWithResponses(server.URL)
	})

	AfterEach(func() {
		server.Close()
	})

	Context("Quando envio um prompt válido", func() {
		It("deve retornar 200 OK e a resposta do LLM", func() {
			// ARRANGE
			session := "sess-123"
			body := PostApiV1ChatJSONRequestBody{
				Prompt:    "Qual a capital do Brasil?",
				SessionId: &session,
			}

			// ACT - Chama a API usando o client gerado (Tipagem Forte)
			ctx := context.Background()
			resp, err := client.PostApiV1ChatWithResponse(ctx, body)

			// ASSERT (Ginkgo/Gomega)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode()).To(Equal(http.StatusOK))
			
			// O .JSON200 já existe no client tipado! Fim do unmarshal manual.
			Expect(resp.JSON200.Answer).To(ContainSubstring("Brasília"))
			// Expect(resp.JSON200.Answer).To(ContainSubstring("Simulada"))
			Expect(*resp.JSON200.Provider).To(Equal("google-adk"))
		})
	})
})
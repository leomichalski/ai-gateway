package http

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/labstack/echo/v5"
)

var _ = Describe("Health Check Endpoint", func() {
	var (
		server *httptest.Server
		client *ClientWithResponses
	)

	BeforeEach(func() {
		e := echo.New()
		
		// For the health check, the UseCase may be nil
		apiHandler := NewAPIHandler(nil)
		
		strictHandler := NewStrictHandler(apiHandler, nil)
		RegisterHandlers(e, strictHandler)

		server = httptest.NewServer(e)

		var err error
		client, err = NewClientWithResponses(server.URL)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})

	Context("Quando o K8s ou Load Balancer requisitar o status de vida", func() {
		It("deve retornar 200 OK confirmando que a API está no ar", func() {
			// Calls the API using the typed client (no body, no complex headers)
			ctx := context.Background()
			resp, err := client.GetHealthzWithResponse(ctx)

			// ASSERT
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode()).To(Equal(http.StatusOK))
		})
	})
})
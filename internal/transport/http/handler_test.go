package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/labstack/echo/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {
	var (
		e      *echo.Echo
		rec    *httptest.ResponseRecorder
		server *Server
	)

	BeforeEach(func() {
		e = echo.New()
		rec = httptest.NewRecorder()
		server = NewServer()
	})

	Describe("Unit: Direct Handler Calls", func() {
		Context("when GET /healthz is invoked", func() {
			It("returns HTTP 200 and health status ok", func() {
				req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
				ctx := e.NewContext(req, rec)

				err := server.GetHealthz(ctx)
				Expect(err).NotTo(HaveOccurred())

				Expect(rec.Code).To(Equal(http.StatusOK))
				Expect(rec.Header().Get(echo.HeaderContentType)).To(ContainSubstring(echo.MIMEApplicationJSON))

				var resp HealthResponse
				err = json.Unmarshal(rec.Body.Bytes(), &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Status).To(Equal(Ok))
			})
		})

		Context("when HEAD /healthz is invoked", func() {
			It("delegates to GetHealthz and returns HTTP 200", func() {
				req := httptest.NewRequest(http.MethodHead, "/healthz", nil)
				ctx := e.NewContext(req, rec)

				err := server.HeadHealthz(ctx)
				Expect(err).NotTo(HaveOccurred())

				Expect(rec.Code).To(Equal(http.StatusOK))

				var resp HealthResponse
				err = json.Unmarshal(rec.Body.Bytes(), &resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Status).To(Equal(Ok))
			})
		})
	})

	Describe("Integration: Routing and HTTP Methods", func() {
		BeforeEach(func() {
			// Register generated OpenAPI routes onto Echo v5
			RegisterHandlers(e, server)
		})

		DescribeTable("verifying endpoint routing behavior",
			func(method, path string, expectedStatus int) {
				req := httptest.NewRequest(method, path, nil)
				e.ServeHTTP(rec, req)

				Expect(rec.Code).To(Equal(expectedStatus))
			},
			Entry("allows GET /healthz", http.MethodGet, "/healthz", http.StatusOK),
			Entry("allows HEAD /healthz", http.MethodHead, "/healthz", http.StatusOK),
			Entry("rejects POST /healthz with 405", http.MethodPost, "/healthz", http.StatusMethodNotAllowed),
			Entry("rejects DELETE /healthz with 405", http.MethodDelete, "/healthz", http.StatusMethodNotAllowed),
		)
	})
})
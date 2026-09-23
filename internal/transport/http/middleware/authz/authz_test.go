package authz_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	// Adjust these import paths to match your actual module name
	"github.com/leomichalski/aigateway/internal/domain"
	"github.com/leomichalski/aigateway/internal/transport/http/middleware/authz"
)

// ==========================================
// Mocks
// ==========================================

// mockAuthZUseCase implements inbound.AuthZUseCase for testing
type mockAuthZUseCase struct {
	err error
}

func (m *mockAuthZUseCase) Authorize(ctx context.Context, identity domain.Identity, object string, action string) error {
	return m.err
}

// ==========================================
// Test Suite
// ==========================================

func TestAuthZMiddleware(t *testing.T) {
	e := echo.New()

	tests := []struct {
		name           string
		skip           bool // Should the skipper bypass the middleware?
		setIdentity    bool // Should we inject an identity into the context?
		extractObj     string
		extractAct     string
		extractErr     error // Error returned by the ResourceExtractor
		usecaseErr     error // Error returned by the AuthZ Use Case
		expectedStatus int
	}{
		{
			name:           "success: authorized",
			skip:           false,
			setIdentity:    true,
			extractObj:     "llm_model",
			extractAct:     "write",
			extractErr:     nil,
			usecaseErr:     nil, // Authorized
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success: skipped via skipper",
			skip:           true,
			setIdentity:    false, // Identity check should be bypassed
			expectedStatus: http.StatusOK,
		},
		{
			name:           "fail: missing identity",
			skip:           false,
			setIdentity:    false, // No identity in context
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "fail: extractor error",
			skip:           false,
			setIdentity:    true,
			extractErr:     errors.New("invalid payload"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "fail: forbidden by usecase",
			skip:           false,
			setIdentity:    true,
			extractObj:     "llm_model",
			extractAct:     "read",
			extractErr:     nil,
			usecaseErr:     errors.New("permission denied"), // Use case rejects it
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "success: empty object/action skips usecase evaluation",
			skip:           false,
			setIdentity:    true,
			extractObj:     "",
			extractAct:     "",
			extractErr:     nil,
			usecaseErr:     errors.New("should not be called"), // If called, it would fail
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup Request & Recorder
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Setup Context (Simulate AuthN middleware behavior)
			if tc.setIdentity {
				c.Set("identity", domain.Identity{TenantID: "test-tenant"})
			}

			// Setup Dependencies
			mockUC := &mockAuthZUseCase{err: tc.usecaseErr}

			skipper := func(c *echo.Context) bool {
				return tc.skip
			}

			extractor := func(c *echo.Context) (string, string, error) {
				return tc.extractObj, tc.extractAct, tc.extractErr
			}

			// Initialize Middleware
			mw := authz.Middleware(mockUC, skipper, extractor)

			// Create a dummy next handler
			nextHandler := func(c *echo.Context) error {
				return c.String(http.StatusOK, "OK")
			}

			// Execute Middleware
			err := mw(nextHandler)(c)

			// Evaluate Results
			var actualStatus int

			// In Echo v5, middleware returns *echo.HTTPError for early termination.
			// If it's successful, err is nil, and the next handler writes the OK status.
			if err != nil {
				var httpErr *echo.HTTPError
				if errors.As(err, &httpErr) {
					actualStatus = httpErr.Code
				} else {
					t.Fatalf("expected *echo.HTTPError, got %T: %v", err, err)
				}
			} else {
				actualStatus = rec.Code
			}

			if actualStatus != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, actualStatus)
			}
		})
	}
}
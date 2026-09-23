package authn_test

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/leomichalski/aigateway/internal/adapter/repository"
	"github.com/leomichalski/aigateway/internal/domain"
	"github.com/leomichalski/aigateway/internal/transport/http/middleware/authn"
	"github.com/leomichalski/aigateway/internal/usecases"
)

func TestAuthNMiddleware(t *testing.T) {
	// ==========================================
	// Setup Dependencies
	// ==========================================
	rawKey := "sk_test_not_secret"
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	// Seed the static repository with our test key
	repo := repository.NewStaticIdentityRepository(map[string]domain.Identity{
		keyHash: {
			TenantID:   "tenant-test",
			Tier:       "pro",
			APIKeyHash: keyHash,
		},
	})

	// Initialize the real use case
	authNUseCase := usecases.NewAuthNUseCase(repo)

	// ==========================================
	// Setup Skipper Logic
	// ==========================================
	skipper := func(c *echo.Context) bool {
		path := c.Request().URL.Path
		return path == "/health" || strings.HasPrefix(path, "/docs")
	}

	// ==========================================
	// Define Test Cases
	// ==========================================
	tests := []struct {
		name           string
		path           string
		authHeader     string
		expectedStatus int
		expectIdentity bool
	}{
		{
			name:           "valid token injects identity",
			path:           "/api/chat",
			authHeader:     "Bearer " + rawKey,
			expectedStatus: http.StatusOK,
			expectIdentity: true, // Should pass AuthN and set Identity
		},
		{
			name:           "missing header fails authn",
			path:           "/api/chat",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectIdentity: false,
		},
		{
			name:           "invalid format (basic) fails authn",
			path:           "/api/chat",
			authHeader:     "Basic some-token",
			expectedStatus: http.StatusUnauthorized,
			expectIdentity: false,
		},
		{
			name:           "unauthorized from domain (wrong key)",
			path:           "/api/chat",
			authHeader:     "Bearer sk_wrong_key",
			expectedStatus: http.StatusUnauthorized,
			expectIdentity: false,
		},
		{
			name:           "skipper bypasses /health without auth",
			path:           "/health",
			authHeader:     "",
			expectedStatus: http.StatusOK,
			expectIdentity: false, // Skipped, so no identity is injected
		},
		{
			name:           "skipper bypasses /docs/openapi.yaml without auth",
			path:           "/docs/openapi.yaml",
			authHeader:     "",
			expectedStatus: http.StatusOK,
			expectIdentity: false, // Skipped, so no identity is injected
		},
	}

	// ==========================================
	// Run Tests
	// ==========================================
	e := echo.New()
	mw := authn.Middleware(authNUseCase, skipper)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()
			
			// Setup Echo Context
			c := e.NewContext(req, rec)

			// Create a dummy handler to terminate the middleware chain
			handler := mw(func(c *echo.Context) error {
				if tc.expectIdentity {
					// Verify identity exists and is of correct type
					val := c.Get(authn.IdentityContextKey)
					if val == nil {
						t.Errorf("expected identity to be set in context")
					}
					identity, ok := val.(domain.Identity)
					if !ok {
						t.Errorf("expected value to be domain.Identity type")
					}
					if identity.TenantID != "tenant-test" {
						t.Errorf("expected TenantID 'tenant-test', got '%s'", identity.TenantID)
					}
				} else {
					// Ensure identity was NOT accidentally set
					if c.Get(authn.IdentityContextKey) != nil {
						t.Errorf("expected identity to NOT be set")
					}
				}
				// Simulate successful downstream response
				return c.String(http.StatusOK, "OK")
			})

			// Execute middleware and handler
			_ = handler(c)

			// Assert Status Code matches expectation
			if rec.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, rec.Code)
			}
		})
	}
}
package authn

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/leomichalski/aigateway/internal/port/inbound"
)

const IdentityContextKey = "identity"

type ErrorResponse struct {
	Error string `json:"error"`
}

// Middleware creates the Echo v5 authentication middleware with a Skipper.
func Middleware(useCase inbound.AuthNUseCase, skip middleware.Skipper) echo.MiddlewareFunc {
	// Fallback if no skipper is provided
	if skip == nil {
		skip = middleware.DefaultSkipper
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Check if we should skip AuthN for this route
			if skip(c) {
				return next(c)
			}

			// Proceed with AuthN logic
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing authorization header"})
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid authorization format, expected Bearer"})
			}

			rawKey := strings.TrimSpace(parts[1])
			if rawKey == "" {
				return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "empty bearer token"})
			}

			// We explicitly do NOT log rawKey anywhere here.
			identity, err := useCase.Authenticate(c.Request().Context(), rawKey)
			if err != nil {
				// Instantly break chain, return generic 401 to prevent timing/data enumeration attacks
				return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
			}

			// Inject identity into context for downstream rate-limiting/authorization
			c.Set(IdentityContextKey, identity)

			return next(c)
		}
	}
}
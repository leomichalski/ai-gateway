package authz

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/leomichalski/aigateway/internal/domain"
	"github.com/leomichalski/aigateway/internal/port/inbound"
)

// ResourceExtractor is the Strategy we pass to the middleware
type ResourceExtractor func(c *echo.Context) (object string, action string, err error)

// Middleware now requires both the generic Use Case and the specific Extractor
func Middleware(usecase inbound.AuthZUseCase, skip middleware.Skipper, extract ResourceExtractor) echo.MiddlewareFunc {
	// Fallback if no skipper is provided
	if skip == nil {
		skip = middleware.DefaultSkipper
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Check if we should skip AuthZ for this route
			if skip(c) {
				return next(c)
			}

			identityRaw := c.Get("identity")
			identity, ok := identityRaw.(domain.Identity)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing identity")
			}

			// Delegate HTTP extraction to the injected strategy
			object, action, err := extract(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "failed to parse resource context")
			}

			// Delegate Business Authorization to the Use Case
			if object != "" && action != "" {
				err = usecase.Authorize(c.Request().Context(), identity, object, action)
				if err != nil {
					return echo.NewHTTPError(http.StatusForbidden, err.Error())
				}
			}

			return next(c)
		}
	}
}
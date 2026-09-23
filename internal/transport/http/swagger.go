// internal/transport/http/swagger.go
package api

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// ConfigureSwagger mounts the OpenAPI JSON and the Swagger UI on the Echo router.
func ConfigureSwagger(e *echo.Echo) {
	// Explicitly cast to echo.HandlerFunc to satisfy the Echo v5 compiler requirement
	e.GET("/docs/openapi.json", echo.HandlerFunc(func(c *echo.Context) error {
		swagger, err := GetSwagger()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load swagger spec"})
		}
		data, err := swagger.MarshalJSON()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to serialize swagger spec"})
		}
		return c.JSONBlob(http.StatusOK, data)
	}))

	// Explicitly cast to echo.HandlerFunc here as well
	e.GET("/docs", echo.HandlerFunc(func(c *echo.Context) error {
		html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>AI Gateway API Docs</title>
    <link rel="stylesheet" type="text/css" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.11.0/swagger-ui.css" />
    <style>body { margin: 0; padding: 0; }</style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.11.0/swagger-ui-bundle.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.11.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            window.ui = SwaggerUIBundle({
                url: "/docs/openapi.json",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`
		return c.HTML(http.StatusOK, html)
	}))
}
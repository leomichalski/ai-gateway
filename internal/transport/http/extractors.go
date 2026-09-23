package api

import (
	"bytes"
	"encoding/json"
	"io"
	"github.com/labstack/echo/v5"
)

// Extracts object from JSON body for completions (e.g. "gpt-4o", "use")
func ExtractLLMModelFromBody(c *echo.Context) (string, string, error) {
	if c.Request() == nil || c.Request().Body == nil || c.Request().ContentLength == 0 {
		return "", "", nil
	}

	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return "", "", err
	}
	// Restore body buffer
	c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var payload struct {
		Model string `json:"model"`
	}
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &payload)
	}

	return payload.Model, "use", nil
}

// // Extracts object from a path parameter (e.g. /v1/templates/:id)
// func ExtractTemplateFromPath(c *echo.Context) (string, string, error) {
// 	templateID := c.PathParam("id")
// 	action := "read"
	
// 	if c.Request().Method == "DELETE" {
// 		action = "delete"
// 	}
	
// 	return templateID, action, nil
// }

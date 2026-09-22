package http

import (
	"context"
)

// -----------------------------------------------------------------------------
// GET /healthz
// -----------------------------------------------------------------------------
// Como usamos StrictServerInterface, não manipulamos *echo.Context diretamente.
// Retornamos o DTO tipado e o oapi-codegen faz o ctx.JSON(StatusOK, resp) por debaixo dos panos.
func (s *APIHandler) GetHealthz(ctx context.Context, req GetHealthzRequestObject) (GetHealthzResponseObject, error) {
	// A struct GetHealthz200JSONResponse é gerada baseada no seu openapi.yaml
	return GetHealthz200JSONResponse{
		Status: "Ok",
	}, nil
}

func (s *APIHandler) HeadHealthz(ctx context.Context, req HeadHealthzRequestObject) (HeadHealthzResponseObject, error) {
	return HeadHealthz200JSONResponse{
		Status: "Ok",
	}, nil
}

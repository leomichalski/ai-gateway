package http

import (
	"context"
)

// -----------------------------------------------------------------------------
// GET /healthz
// -----------------------------------------------------------------------------
func (s *APIHandler) GetHealthz(ctx context.Context, req GetHealthzRequestObject) (GetHealthzResponseObject, error) {
	// struct GetHealthz200JSONResponse generated from openapi.yaml
	return GetHealthz200JSONResponse{
		Status: "Ok",
	}, nil
}

func (s *APIHandler) HeadHealthz(ctx context.Context, req HeadHealthzRequestObject) (HeadHealthzResponseObject, error) {
	return HeadHealthz200JSONResponse{
		Status: "Ok",
	}, nil
}

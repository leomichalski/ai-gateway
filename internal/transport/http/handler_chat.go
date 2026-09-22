package http

import (
	"context"
)

// -----------------------------------------------------------------------------
// POST /api/v1/chat
// -----------------------------------------------------------------------------
func (s *APIHandler) PostApiV1Chat(ctx context.Context, req PostApiV1ChatRequestObject) (PostApiV1ChatResponseObject, error) {
	session := ""
	if req.Body.SessionId != nil {
		session = *req.Body.SessionId
	}

	result, err := s.chatUseCase.ProcessChat(ctx, session, req.Body.Prompt)
	if err != nil {
		errStr := err.Error()
		return PostApiV1Chat500JSONResponse{Message: &errStr}, nil
	}

	return PostApiV1Chat200JSONResponse{
		Answer:     result.Answer,
		TokensUsed: &result.TokensUsed,
		Provider:   &result.Provider,
	}, nil
}

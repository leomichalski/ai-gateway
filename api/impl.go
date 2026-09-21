package api

import (
	"net/http"
	// "sync"

	// "github.com/google/uuid"
	"github.com/labstack/echo/v5"
)


type Server struct {
	// mu    sync.RWMutex
	// tasks map[string]Task
}

func NewServer() *Server {
	// return &Server{
		// tasks: make(map[string]Task),
	// }
	return &Server{}
}


func (Server) GetHealthz(ctx *echo.Context) error {
	resp := HealthResponse{
		Status: Ok,
	}

	return ctx.JSON(http.StatusOK, resp)
}

func (s *Server) HeadHealthz(ctx *echo.Context) error {
	return s.GetHealthz(ctx)
}

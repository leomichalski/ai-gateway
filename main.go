package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/leomichalski/surveil-api/api"
)

func main() {
	// Configure slog (Echo v5 default logging standard)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	e := echo.New()

	// Built-in middlewares
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	// Handlers and route registration
	server := api.NewServer()
	api.RegisterHandlers(e, server)

	// 1. Create a context that listens for OS interrupt signals.
	// When one of these signals occurs, the context is automatically canceled.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Define the server configuration using Echo v5's StartConfig.
	// GracefulTimeout configures how long to wait for active requests to finish.
	sc := echo.StartConfig{
		Address:         ":8080",
		GracefulTimeout: 10 * time.Second,
	}

	slog.Info("server starting on :8080")

	// 3. Start the server synchronously.
	// This blocks natively and handles the entire graceful shutdown flow
	// internally when the signal context (ctx) is canceled.
	if err := sc.Start(ctx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server encountered an error", "err", err)
		os.Exit(1)
	}

	slog.Info("server stopped completely")
}

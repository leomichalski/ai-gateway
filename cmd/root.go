package cmd

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

	httpTransport "github.com/leomichalski/surveil-api/internal/transport/http"

	"github.com/spf13/cobra"
)

var (
	serverPort string
)

var rootCmd = &cobra.Command{
	Use:   "surveil-api",
	Short: "Surveillance ingestion and search service.",
	Long: `Surveillance ingestion and search service1.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Configure slog (Echo v5 default logging standard)
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		slog.SetDefault(logger)

		e := echo.New()

		// Built-in middlewares
		e.Use(middleware.RequestLogger())
		e.Use(middleware.Recover())

		// Handlers and route registration
		server := httpTransport.NewServer()
		httpTransport.RegisterHandlers(e, server)

		// 1. Create a context that listens for OS interrupt signals.
		// When one of these signals occurs, the context is automatically canceled.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		// 2. Define the server configuration using Echo v5's StartConfig.
		// GracefulTimeout configures how long to wait for active requests to finish.
		sc := echo.StartConfig{
			Address:         ":" + serverPort,
			GracefulTimeout: 10 * time.Second,
		}

		slog.Info("server starting on " + sc.Address)

		// 3. Start the server synchronously.
		// This blocks natively and handles the entire graceful shutdown flow
		// internally when the signal context (ctx) is canceled.
		if err := sc.Start(ctx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server encountered an error", "err", err)
			os.Exit(1)
		}

		slog.Info("server stopped completely")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().IntVar(&aA, "a-a", 5, "Description.")
	// rootCmd.Flags().BoolVar(&aA, "a-a", true, "Description.")
	// rootCmd.Flags().DurationVar(&aA, "a-a", 30*time.Minute, "Description.")
	rootCmd.Flags().StringVar(&serverPort, "server-port", "8080", "Server port.")
}

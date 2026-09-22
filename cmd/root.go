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
	"github.com/leomichalski/surveil-api/internal/adapter/adk_runner"
	"github.com/leomichalski/surveil-api/internal/usecases"

	"github.com/spf13/cobra"
)

var (
	serverPort string
	debug bool
)

var rootCmd = &cobra.Command{
	Use:   "surveil-api",
	Short: "Surveillance ingestion and search service.",
	Long:  `Surveillance ingestion and search service1.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Configure slog (Echo v5 default logging standard)
		loggerHandlerOptions := &slog.HandlerOptions{}
		if debug {
			loggerHandlerOptions.Level = slog.LevelDebug
		}
		logger := slog.New(slog.NewJSONHandler(os.Stdout, loggerHandlerOptions))
		slog.SetDefault(logger)

		e := echo.New()

		e.Use(middleware.RequestLogger())
		e.Use(middleware.Recover())

		llmAdapter := adk_runner.NewGoogleADKAdapter()
		chatUC := usecases.NewChatUseCase(llmAdapter)

		// Handlers and route registration
		apiHandler := httpTransport.NewAPIHandler(chatUC)
		strictHandler := httpTransport.NewStrictHandler(apiHandler, nil)
		httpTransport.RegisterHandlers(e, strictHandler)

		if debug {
			httpTransport.ConfigureSwagger(e)
		}

		// Create a context that listens for OS interrupt signals.
		// When one of these signals occurs, the context is automatically canceled.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		// Define the server configuration using Echo v5's StartConfig.
		// GracefulTimeout configures how long to wait for active requests to finish.
		sc := echo.StartConfig{
			Address:         ":" + serverPort,
			GracefulTimeout: 10 * time.Second,
		}

		slog.Info("server starting on " + sc.Address)

		// Start the server synchronously.
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
	rootCmd.Flags().BoolVar(&debug, "debug", true, "Debug mode.")
}

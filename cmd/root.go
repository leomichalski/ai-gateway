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
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	httpTransport "github.com/leomichalski/aigateway/internal/transport/http"
	"github.com/leomichalski/aigateway/internal/adapter/adk_runner"
	"github.com/leomichalski/aigateway/internal/adapter/secrets"
	"github.com/leomichalski/aigateway/internal/adapter/casbin"
	"github.com/leomichalski/aigateway/internal/adapter/repository"
	"github.com/leomichalski/aigateway/internal/usecases"
	"github.com/leomichalski/aigateway/internal/domain"
	"github.com/leomichalski/aigateway/internal/transport/http/middleware/authn"
	"github.com/leomichalski/aigateway/internal/transport/http/middleware/authz"

	"github.com/spf13/cobra"
)

var (
	serverPort string
	debug bool
	// environment string
)

var rootCmd = &cobra.Command{
	Use:   "aigateway",
	Short: "AI Gateway.",
	Long:  `AI Gateway.`,
	Run: func(cmd *cobra.Command, args []string) {
		// =====================================================================
		// Configure logger
		// =====================================================================
		loggerHandlerOptions := &slog.HandlerOptions{}
		if debug {
			loggerHandlerOptions.Level = slog.LevelDebug
		}
		logger := slog.New(slog.NewJSONHandler(os.Stdout, loggerHandlerOptions))
		slog.SetDefault(logger)

		// =====================================================================
		// Echo server
		// =====================================================================
		e := echo.New()

		// MIDDLEWARES
		e.Use(middleware.RequestLogger())
		e.Use(middleware.Recover())

		// AuthN middleware
		// TODO: substitute seed with mongodb
		rawTestKey := "sk_test_123"
		hash := sha256.Sum256([]byte(rawTestKey))
		keyHash := hex.EncodeToString(hash[:])

		seedIdentities := map[string]domain.Identity{
			keyHash: {
				TenantID:   "tenant-local-dev",
				Tier:       "premium",
				APIKeyHash: keyHash,
			},
		}

		identityRepo := repository.NewStaticIdentityRepository(seedIdentities)
		authNUseCase := usecases.NewAuthNUseCase(identityRepo)

		// Skip authentication for public routes
		authn_skipper := func(c *echo.Context) bool {
			path := c.Request().URL.Path
			// Skip exactly /health, and any path starting with /docs
			return path == "/healthz" || (debug && strings.HasPrefix(path, "/docs"))
		}
		e.Use(authn.Middleware(authNUseCase, authn_skipper))

		casbinEnforcer, err := casbin.NewMongoEnforcer("casbin_model.conf")
		if err != nil {
			slog.Error("Failed to init Casbin Enforcer.", "error", err)
			os.Exit(1)
		}
		authZUseCase := usecases.NewAuthZUseCase(casbinEnforcer)
		// Skips checking model permissions for endpoints other than "/v1/chat/completions"
		authz_llmmodel_skipper := func(c *echo.Context) bool {
			path := c.Request().URL.Path
			return path != "/v1/chat/completions"
		}
		e.Use(authz.Middleware(authZUseCase, authz_llmmodel_skipper, httpTransport.ExtractLLMModelFromBody))
		// e.Use(authz.Middleware(authZUseCase, authz_llmmodel_skipper, httpTransport.xxxxxxxxxxx))

		// HANDLERS
		llmAdapter := adk_runner.NewGoogleADKAdapter()
		envSecretProvider := secrets.NewEnvSecretProvider()
		chatUC := usecases.NewChatUseCase(llmAdapter, envSecretProvider)
		chatHandler := httpTransport.NewChatHandler(chatUC)

		// templateHandler := httpTransport.NewTemplateHandler(templateUC)

		apiHandler := httpTransport.NewAPIHandler(chatHandler /*, templateHandler */)
		strictHandler := httpTransport.NewStrictHandler(apiHandler, nil)
		httpTransport.RegisterHandlers(e, strictHandler)

		// SWAGGER SERVER
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
	rootCmd.Flags().BoolVar(&debug, "debug", false, "Debug mode.")
	// rootCmd.Flags().StringVar(&environment, "environment", "dev", "Environment (dev or else).")
}

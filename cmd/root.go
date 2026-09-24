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
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	echoprometheus "github.com/labstack/echo-prometheus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	
	httpTransport "github.com/leomichalski/aigateway/internal/transport/http"
	"github.com/leomichalski/aigateway/internal/port/outbound"
	"github.com/leomichalski/aigateway/internal/adapter/adk_runner"
	"github.com/leomichalski/aigateway/internal/adapter/secrets"
	"github.com/leomichalski/aigateway/internal/adapter/casbin"
	"github.com/leomichalski/aigateway/internal/adapter/repository"
	"github.com/leomichalski/aigateway/internal/usecases"
	"github.com/leomichalski/aigateway/internal/transport/http/middleware/authn"
	"github.com/leomichalski/aigateway/internal/transport/http/middleware/authz"

	"github.com/spf13/cobra"
)

var (
	serverPort string
	prometheusPort string
	exportMetrics bool
	debug bool
	// environment string
)

func BuildServer(client *mongo.Client, dbConfig outbound.DatabaseConfig, envProvider *secrets.EnvProvider) *echo.Echo {
    e := echo.New()
	// =====================================================================
	// Middlewares
	// =====================================================================

	// Adds middleware to gather metrics
	if exportMetrics {
		e.Use(echoprometheus.NewMiddleware("aigateway"))
		// Make `/metrics` endpoint run at prometheusPort, not at serverPort
		go func() {
			metrics := echo.New()
			metrics.GET("/metrics", echoprometheus.NewHandler())
			if err := metrics.Start(":" + prometheusPort); err != nil {
				e.Logger.Error("failed to start metrics server", "error", err)
			}
		}()
	}

	// Logger
	e.Use(middleware.RequestLogger())

	// Recovers from panics anywhere in the chain
	e.Use(middleware.Recover())

	// Authentication (authn)
	db := client.Database(dbConfig.DBName)
	identityRepo := repository.NewMongoIdentityRepository(db)
	authNUseCase := usecases.NewAuthNUseCase(identityRepo)
	// Skip authentication for public routes
	authn_skipper := func(c *echo.Context) bool {
		path := c.Request().URL.Path
		// Skip exactly /health, and any path starting with /docs
		return path == "/healthz" || (debug && strings.HasPrefix(path, "/docs"))
	}
	e.Use(authn.Middleware(authNUseCase, authn_skipper))

	// Authorization (authz)
	casbinEnforcer, err := casbin.NewMongoEnforcer(client, dbConfig.DBName)
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

	// =====================================================================
	// Handlers
	// =====================================================================

	// Chat handler
	llmAdapter := adk_runner.NewGoogleADKAdapter()
	chatUC := usecases.NewChatUseCase(llmAdapter, envProvider)
	chatHandler := httpTransport.NewChatHandler(chatUC)

	// Template handler
	// templateHandler := httpTransport.NewTemplateHandler(templateUC)

	// Root handler
	apiHandler := httpTransport.NewAPIHandler(chatHandler /*, templateHandler */)
	strictHandler := httpTransport.NewStrictHandler(apiHandler, nil)
	httpTransport.RegisterHandlers(e, strictHandler)
    return e
}

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
		// Configure secrets/config
		// =====================================================================
		envProvider := secrets.NewEnvProvider()

		dbConfig, err := envProvider.GetDatabaseConfig()
		if err != nil {
			slog.Error("failed to load database config", "error", err)
			os.Exit(1)
		}

		// =====================================================================
		// MongoDB connection
		// =====================================================================

		// Create a short-lived context JUST for the initial connection
		startupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		client, err := mongo.Connect(startupCtx, options.Client().ApplyURI(dbConfig.URI))
		cancel() // instantly release resources after Connect finishes (success or fail)

		if err != nil {
			// Ping it to ensure we actually connected
			panic("failed to connect to mongodb: " + err.Error())
		}
		if err := client.Ping(context.Background(), nil); err != nil {
			panic("failed to ping mongodb: " + err.Error())
		}

		// =====================================================================
		// Server
		// =====================================================================
		e := BuildServer(client, dbConfig, envProvider)

		// =====================================================================
		// Swagger
		// =====================================================================
		if debug {
			httpTransport.ConfigureSwagger(e)
		}

		// =====================================================================
		// Graceful shutdown
		// =====================================================================

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
		serverErr := sc.Start(ctx, e)
		if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
			slog.Error("server encountered an error", "err", serverErr)
			// Do NOT os.Exit(1) right here, or we skip MongoDB cleanup!
		}
		slog.Info("disconnecting MongoDB...")

		// We MUST create a fresh context here.
		// The original 'ctx' is already canceled by the OS signal!
		mongoCtx, cancelMongo := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelMongo()

		if err := client.Disconnect(mongoCtx); err != nil {
			slog.Error("failed to disconnect MongoDB cleanly", "err", err)
		} else {
			slog.Info("MongoDB disconnected elegantly")
		}

		// Finally, exit with a non-zero code if the server crashed earlier
		if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
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
	rootCmd.Flags().StringVar(&prometheusPort, "prometheus-port", "9090", "Prometheus port. Can't be the same as the server port, as a measure to avoid public `/metrics` leakage.")
	rootCmd.Flags().BoolVar(&exportMetrics, "export-metrics", true, "Wether to export Prometheus metrics.")
	rootCmd.Flags().BoolVar(&debug, "debug", false, "Debug mode.")
	// rootCmd.Flags().StringVar(&environment, "environment", "dev", "Environment (dev or else).")
}

package secrets

import (
	"context"
	"fmt"
	"os"

	"github.com/leomichalski/aigateway/internal/port/outbound"
	"github.com/leomichalski/aigateway/internal/domain"
)

type EnvProvider struct{}

func NewEnvProvider() *EnvProvider {
	return &EnvProvider{}
}

// ---------------------------------------------------------
// Implements outbound.SecretProvider
// ---------------------------------------------------------
func (p *EnvProvider) GetProviderKey(ctx context.Context, name string) (string, error) {
	key := ""
    switch name {
	// case "openai":
		// key = os.Getenv("OPENAI_API_KEY")
	case "gemini":
		key = os.Getenv("GOOGLE_API_KEY")
	default:
		return "", domain.ErrSecretNotFound
    }
	if key == "" {
		return "", domain.ErrSecretNotSet
	}
	return key, nil
}

// ---------------------------------------------------------
// Implements outbound.DatabaseConfigProvider
// ---------------------------------------------------------
func (p *EnvProvider) GetDatabaseConfig() (outbound.DatabaseConfig, error) {
	host := os.Getenv("DATABASE_HOST")
	port := os.Getenv("DATABASE_PORT")
	user := os.Getenv("DATABASE_USERNAME")
	pass := os.Getenv("DATABASE_PASSWORD")
	dbName := os.Getenv("DATABASE_NAME")

	authSource := os.Getenv("DATABASE_AUTH_SOURCE")
	if authSource == "" {
		authSource = "admin"
	}

	if host == "" || dbName == "" {
		return outbound.DatabaseConfig{}, fmt.Errorf("missing required database environment variables")
	}

	uri := fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s",
		user, pass, host, port, dbName, authSource)

	return outbound.DatabaseConfig{
		URI:    uri,
		DBName: dbName,
	}, nil
}

package secrets

import (
	"context"
	"os"

	"github.com/leomichalski/aigateway/internal/port/outbound"
	"github.com/leomichalski/aigateway/internal/domain"
)

type EnvSecretProvider struct {}

func NewEnvSecretProvider() outbound.SecretProvider {
	// adkClient: adk.NewClient(os.Getenv("ADK_API_KEY")),
	return &EnvSecretProvider{}
}

func (p *EnvSecretProvider) GetProviderKey(ctx context.Context, name string) (string, error) {
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

package secrets

import (
	"context"
	"fmt"
	"os"

	"github.com/leomichalski/aigateway/internal/port/outbound"
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
		return "", fmt.Errorf("Provider Key not found for: %s", name)
    }
	if key == "" {
		return "", fmt.Errorf("Provider key is not set.")
	}
	return key, nil
}

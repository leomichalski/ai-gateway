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
    switch name {
	// case "openai":
	// 	return os.Getenv("OPENAI_API_KEY"), nil
	case "gemini":
		return os.Getenv("GOOGLE_API_KEY"), nil
    }
	return "", fmt.Errorf("Provider Key not found for: %s", name)
}
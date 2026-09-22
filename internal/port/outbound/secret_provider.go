package outbound

import "context"

type SecretProvider interface {
    GetProviderKey(ctx context.Context, providerName string) (string, error)
}

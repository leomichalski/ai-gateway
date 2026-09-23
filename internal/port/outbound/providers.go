package outbound

import "context"

type SecretProvider interface {
    GetProviderKey(ctx context.Context, providerName string) (string, error)
}

type DatabaseConfig struct {
	URI    string
	DBName string
}

// Used by the application bootstrapper (root.go)
type DatabaseConfigProvider interface {
	GetDatabaseConfig() (DatabaseConfig, error)
}

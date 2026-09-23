
package casbin

import (
	"context"
	"fmt"
	"os"

	casbinlib "github.com/casbin/casbin/v2"
	mongodbadapter "github.com/casbin/mongodb-adapter/v3"
	
	"github.com/leomichalski/aigateway/internal/port/outbound"
)

type MongoEnforcer struct {
	enforcer *casbinlib.Enforcer
}

// NewMongoEnforcer initializes the Casbin adapter using Env Vars
func NewMongoEnforcer(modelPath string) (outbound.PermissionEnforcer, error) {
	host := os.Getenv("DATABASE_HOST")
	port := os.Getenv("DATABASE_PORT")
	user := os.Getenv("DATABASE_USERNAME")
	pass := os.Getenv("DATABASE_PASSWORD")
	dbName := os.Getenv("DATABASE_NAME")

	authSource := os.Getenv("DATABASE_AUTH_SOURCE")
	if authSource == "" {
		authSource = "admin" 
	}

	// Append ?authSource= to the URI
	mongoURI := fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s", 
		user, pass, host, port, dbName, authSource)

	// Initialize MongoDB adapter for Casbin
	adapter, err := mongodbadapter.NewAdapter(mongoURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin mongo adapter: %w", err)
	}

	// Initialize Casbin Enforcer with the model definition and the Mongo adapter
	enforcer, err := casbinlib.NewEnforcer(modelPath, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	// Load policies from DB into memory
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	return &MongoEnforcer{enforcer: enforcer}, nil
}

func (m *MongoEnforcer) CheckAccess(ctx context.Context, subject, object, action string) (bool, error) {
	// Enforce returns true if the policy allows the request
	return m.enforcer.Enforce(subject, object, action)
}

package casbin

import (
	"context"
	"fmt"
	_ "embed"

	casbinlib "github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	mongodbadapter "github.com/casbin/mongodb-adapter/v3"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/leomichalski/aigateway/internal/port/outbound"
)

//go:embed casbin_model.conf
var casbinModelText string

type MongoEnforcer struct {
	enforcer *casbinlib.Enforcer
}

// NewMongoEnforcer initializes the Casbin adapter using Env Vars
func NewMongoEnforcer(client *mongo.Client, dbName string) (outbound.PermissionEnforcer, error) {
	// Initialize MongoDB adapter for Casbin
	adapterConfig := &mongodbadapter.AdapterConfig{
	    DatabaseName: dbName,
		// CollectionName string
		// Timeout        time.Duration
		// IsFiltered     bool
	}
	adapter, err := mongodbadapter.NewAdapterByDB(client, adapterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin mongo adapter: %w", err)
	}

	// Parse model directly from the embedded string (no disk I/O required)
	casbinModel, err := model.NewModelFromString(casbinModelText)
	if err != nil {
		return nil, err
	}
	// Initialize Casbin Enforcer with the parsed model and the Mongo adapter
	enforcer, err := casbinlib.NewEnforcer(casbinModel, adapter)
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

package casbin

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCasbinMongoEnforcer(t *testing.T) {
	ctx := context.Background()

	// 1. Spin up MongoDB Container
	req := testcontainers.ContainerRequest{
		Image:        "mongo:6.0",
		ExposedPorts: []string{"27017/tcp"},
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": "root",
			"MONGO_INITDB_ROOT_PASSWORD": "password",
		},
		WaitingFor: wait.ForLog("Waiting for connections"),
	}
	mongoC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer mongoC.Terminate(ctx)

	host, err := mongoC.Host(ctx)
	require.NoError(t, err)
	port, err := mongoC.MappedPort(ctx, "27017")
	require.NoError(t, err)

	dbName := "admin"
	
	// Construct the URI dynamically using the TestContainer's mapped host/port
	mongoURI := fmt.Sprintf("mongodb://root:password@%s:%s/%s?authSource=admin", host, port.Port(), dbName)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	require.NoError(t, err)
	
	// Ensure we cleanly disconnect the client after the test finishes
	defer client.Disconnect(ctx)
	
	// Verify the connection actually succeeded
	err = client.Ping(ctx, nil)
	require.NoError(t, err, "expected successful ping to mongo testcontainer")

	// Initialize Adapter
	// Assume casbin_model.conf is stored in a test fixtures directory
	adapter, err := NewMongoEnforcer(client, dbName, "../../../casbin_model.conf")
	require.NoError(t, err)

	// We cast to our concrete type to access the underlying Casbin Enforcer for seeding data
	impl := adapter.(*MongoEnforcer)

	// 3. Seed Acceptance Criteria: Tenant 1 allowed gemini-3.5-flash-lite and gemini-3.1-pro-preview
	_, err = impl.enforcer.AddPolicy("tenant-1", "gemini-3.5-flash-lite", "use")
	require.NoError(t, err)

	_, err = impl.enforcer.AddPolicy("tenant-1", "gemini-3.1-pro-preview", "use")
	require.NoError(t, err)

	// 4. Run Tests against the Port Interface
	t.Run("Given tenant allowed gemini-3.5-flash-lite, Then returns success", func(t *testing.T) {
		allowed, err := adapter.CheckAccess(ctx, "tenant-1", "gemini-3.5-flash-lite", "use")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("Given tenant allowed ONLY gemini-3.1-pro-preview, When requests claude-3-5-sonnet, Then returns forbidden", func(t *testing.T) {
		allowed, err := adapter.CheckAccess(ctx, "tenant-1", "claude-3-5-sonnet", "use")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}
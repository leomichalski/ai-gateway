package casbin

import (
	"context"
	// "fmt"
	"os"
	"testing"
	// "time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
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

	// Inject env vars to satisfy our Adapter's requirements
	os.Setenv("DATABASE_HOST", host)
	os.Setenv("DATABASE_PORT", port.Port())
	os.Setenv("DATABASE_USERNAME", "root")
	os.Setenv("DATABASE_PASSWORD", "password")
	os.Setenv("DATABASE_NAME", "admin")

	// 2. Initialize Adapter
	// Assume casbin_model.conf is stored in a test fixtures directory
	adapter, err := NewMongoEnforcer("../../../casbin_model.conf")
	require.NoError(t, err)

	// We cast to our concrete type to access the underlying Casbin Enforcer for seeding data
	impl := adapter.(*MongoEnforcer) 

	// 3. Seed Acceptance Criteria: Tenant 1 allowed gemini-1.5-pro and gpt-4o
	_, err = impl.enforcer.AddPolicy("tenant-1", "gemini-1.5-pro", "use")
	require.NoError(t, err)
	
	_, err = impl.enforcer.AddPolicy("tenant-1", "gpt-4o", "use")
	require.NoError(t, err)

	// 4. Run Tests against the Port Interface
	t.Run("Given tenant allowed gemini-1.5-pro, Then returns success", func(t *testing.T) {
		allowed, err := adapter.CheckAccess(ctx, "tenant-1", "gemini-1.5-pro", "use")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("Given tenant allowed ONLY gpt-4o, When requests claude-3-5-sonnet, Then returns forbidden", func(t *testing.T) {
		allowed, err := adapter.CheckAccess(ctx, "tenant-1", "claude-3-5-sonnet", "use")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}
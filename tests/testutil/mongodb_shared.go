package testutil

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/infrastructure/mongodb"
)

// pingRetryDelay is the delay between ping retries when connecting to MongoDB.
const pingRetryDelay = 500 * time.Millisecond

// containerStartupTimeout is the timeout for starting a new MongoDB container.
const containerStartupTimeout = 120 * time.Second

// sharedMongoContainer holds the singleton MongoDB container and client
var (
	sharedContainer   *SharedMongoContainer
	sharedContainerMu sync.Mutex
)

// SharedMongoContainer represents a reusable MongoDB container for tests
type SharedMongoContainer struct {
	Container testcontainers.Container
	URI       string
	client    *mongo.Client
	clientMu  sync.Mutex
}

// GetSharedMongoContainer returns a singleton MongoDB container.
// The container is started once and reused across all tests.
// It will be terminated when the test binary exits.
// If the container has crashed, it will be automatically recreated.
func GetSharedMongoContainer(ctx context.Context) (*SharedMongoContainer, error) {
	sharedContainerMu.Lock()
	defer sharedContainerMu.Unlock()

	// Check if we need to create or recreate the container
	needsCreation := sharedContainer == nil

	if !needsCreation {
		// Check if container is still running
		running, err := isContainerRunning(ctx, sharedContainer.Container)
		if err != nil || !running {
			// Container has crashed, need to recreate it
			// First, try to terminate the old container gracefully
			if sharedContainer.Container != nil {
				terminateCtx, cancel := context.WithTimeout(context.Background(), mongoContainerTerminateTimeout)
				_ = sharedContainer.Container.Terminate(terminateCtx)
				cancel()
			}
			// Disconnect old client if exists
			if sharedContainer.client != nil {
				disconnectCtx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
				_ = sharedContainer.client.Disconnect(disconnectCtx)
				cancel()
			}
			sharedContainer = nil
			needsCreation = true
		}
	}

	if needsCreation {
		// Use a longer timeout for container startup
		startupCtx, cancel := context.WithTimeout(context.Background(), containerStartupTimeout)
		defer cancel()

		cont, err := startMongoContainer(startupCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to start MongoDB container: %w", err)
		}
		sharedContainer = cont
	}

	return sharedContainer, nil
}

// isContainerRunning checks if the container is still running
func isContainerRunning(ctx context.Context, cont testcontainers.Container) (bool, error) {
	if cont == nil {
		return false, nil
	}

	state, err := cont.State(ctx)
	if err != nil {
		return false, err
	}

	return state.Running, nil
}

// startMongoContainer starts a new MongoDB container
func startMongoContainer(ctx context.Context) (*SharedMongoContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "mongo:6.0",
		ExposedPorts: []string{"27017/tcp"},
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": "admin",
			"MONGO_INITDB_ROOT_PASSWORD": "admin123",
		},
		// Limit memory to prevent OOM and crashes
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Memory = 512 * 1024 * 1024 // 512MB
			hc.MemorySwap = 512 * 1024 * 1024
		},
		// Use wiredTiger with limited cache size for stability
		Cmd: []string{"--wiredTigerCacheSizeGB=0.25"},
		// Use both log and port check for more reliable startup detection
		WaitingFor: wait.ForAll(
			wait.ForLog("Waiting for connections").WithStartupTimeout(containerStartupTimeout),
			wait.ForListeningPort("27017/tcp").WithStartupTimeout(containerStartupTimeout),
		),
	}

	cont, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            false, // Don't reuse to avoid stale containers
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start MongoDB container: %w", err)
	}

	host, err := cont.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := cont.MappedPort(ctx, "27017")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	uri := fmt.Sprintf("mongodb://admin:admin123@%s", net.JoinHostPort(host, port.Port()))

	return &SharedMongoContainer{
		Container: cont,
		URI:       uri,
	}, nil
}

// GetClient returns a shared MongoDB client, creating one if needed.
// The client is reused across all tests to avoid connection exhaustion.
func (c *SharedMongoContainer) GetClient(ctx context.Context) (*mongo.Client, error) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()

	// If client already exists and is connected, return it
	if c.client != nil {
		// Quick ping to verify connection
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := c.client.Ping(pingCtx, nil)
		cancel()
		if err == nil {
			return c.client, nil
		}
		// Client is dead, disconnect and recreate
		_ = c.client.Disconnect(ctx)
		c.client = nil
	}

	// Create new client with connection pool settings
	clientOpts := options.Client().
		ApplyURI(c.URI).
		SetMaxPoolSize(100).
		SetMinPoolSize(5)

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping with retries
	maxRetries := 5
	var pingErr error
	for i := range maxRetries {
		pingCtx, pingCancel := context.WithTimeout(context.Background(), mongoPingTimeout)
		pingErr = client.Ping(pingCtx, nil)
		pingCancel()
		if pingErr == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(pingRetryDelay)
		}
	}
	if pingErr != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to ping MongoDB after %d retries: %w", maxRetries, pingErr)
	}

	c.client = client
	return client, nil
}

// SetupSharedTestMongoDB creates a test database using the shared MongoDB container.
// Each test gets its own isolated database within the shared container.
// This is much faster than starting a new container for each test.
// Note: Index creation is disabled by default to prevent MongoDB container crashes.
// Use SetupSharedTestMongoDBWithOptions(t, true) if you need indexes.
func SetupSharedTestMongoDB(t *testing.T) *mongo.Database {
	t.Helper()
	return SetupSharedTestMongoDBWithOptions(t, false)
}

// SetupSharedTestMongoDBWithOptions creates a test database with optional index creation.
// Use createIndexes=false for tests that don't need indexes for faster execution.
func SetupSharedTestMongoDBWithOptions(t *testing.T, createIndexes bool) *mongo.Database {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
	defer cancel()

	cont, err := GetSharedMongoContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to get shared MongoDB container: %v", err)
	}

	// Get shared client
	client, err := cont.GetClient(ctx)
	if err != nil {
		t.Fatalf("Failed to get MongoDB client: %v", err)
	}

	// Create unique database name for test isolation
	dbName := generateTestDBName(t.Name())
	db := client.Database(dbName)

	// Create indexes for production-like testing environment (if requested)
	if createIndexes {
		indexCtx, indexCancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := mongodb.CreateAllIndexes(indexCtx, db); err != nil {
			indexCancel()
			t.Fatalf("Failed to create indexes: %v", err)
		}
		indexCancel()
	}

	// Cleanup: drop database after test (but don't disconnect the shared client)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
		defer cleanupCancel()
		_ = db.Drop(cleanupCtx)
	})

	return db
}

// SetupSharedTestMongoDBWithClient creates a test database and returns both client and database.
// Uses the shared MongoDB container for faster test execution.
// Note: Index creation is disabled by default to prevent MongoDB container crashes.
// Use SetupSharedTestMongoDBWithClientOptions(t, true) if you need indexes.
func SetupSharedTestMongoDBWithClient(t *testing.T) (*mongo.Client, *mongo.Database) {
	t.Helper()
	return SetupSharedTestMongoDBWithClientOptions(t, false)
}

// SetupSharedTestMongoDBWithClientOptions creates a test database with optional index creation.
func SetupSharedTestMongoDBWithClientOptions(t *testing.T, createIndexes bool) (*mongo.Client, *mongo.Database) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
	defer cancel()

	cont, err := GetSharedMongoContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to get shared MongoDB container: %v", err)
	}

	// Get shared client
	client, err := cont.GetClient(ctx)
	if err != nil {
		t.Fatalf("Failed to get MongoDB client: %v", err)
	}

	// Create unique database name for test isolation
	dbName := generateTestDBName(t.Name())
	db := client.Database(dbName)

	// Create indexes for production-like testing environment (if requested)
	if createIndexes {
		indexCtx, indexCancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := mongodb.CreateAllIndexes(indexCtx, db); err != nil {
			indexCancel()
			t.Fatalf("Failed to create indexes: %v", err)
		}
		indexCancel()
	}

	// Cleanup: drop database after test (but don't disconnect the shared client)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
		defer cleanupCancel()
		_ = db.Drop(cleanupCtx)
	})

	return client, db
}

// generateTestDBName creates a unique database name from test name
func generateTestDBName(testName string) string {
	if len(testName) > maxTestNameLength {
		// Use hash for long test names (MongoDB limit: 63 chars)
		hash := sha256.Sum256([]byte(testName))
		testName = testName[:20] + "_" + hex.EncodeToString(hash[:])[:12]
	}
	return "flowra_test_" + testName
}

// CleanupSharedContainer terminates the shared container.
// This is typically called from TestMain or when all tests are done.
func CleanupSharedContainer() {
	sharedContainerMu.Lock()
	defer sharedContainerMu.Unlock()

	if sharedContainer != nil {
		// Disconnect client first
		if sharedContainer.client != nil {
			ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
			_ = sharedContainer.client.Disconnect(ctx)
			cancel()
		}

		// Then terminate container
		if sharedContainer.Container != nil {
			ctx, cancel := context.WithTimeout(context.Background(), mongoContainerTerminateTimeout)
			defer cancel()
			_ = sharedContainer.Container.Terminate(ctx)
		}
		sharedContainer = nil
	}
}

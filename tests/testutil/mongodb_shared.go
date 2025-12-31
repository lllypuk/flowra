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

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// pingRetryDelay is the delay between ping retries when connecting to MongoDB.
const pingRetryDelay = 500 * time.Millisecond

// sharedMongoContainer holds the singleton MongoDB container
var (
	sharedContainer     *SharedMongoContainer
	sharedContainerOnce sync.Once
	errSharedContainer  error
)

// SharedMongoContainer represents a reusable MongoDB container for tests
type SharedMongoContainer struct {
	Container testcontainers.Container
	URI       string
	mu        sync.Mutex
	clients   []*mongo.Client
}

// GetSharedMongoContainer returns a singleton MongoDB container.
// The container is started once and reused across all tests.
// It will be terminated when the test binary exits.
func GetSharedMongoContainer(ctx context.Context) (*SharedMongoContainer, error) {
	sharedContainerOnce.Do(func() {
		container, err := startMongoContainer(ctx)
		if err != nil {
			errSharedContainer = err
			return
		}
		sharedContainer = container
	})

	return sharedContainer, errSharedContainer
}

// startMongoContainer starts a new MongoDB container
func startMongoContainer(ctx context.Context) (*SharedMongoContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "mongo:8",
		Name:         "flowra-test-mongodb", // Required for Reuse mode
		ExposedPorts: []string{"27017/tcp"},
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": "admin",
			"MONGO_INITDB_ROOT_PASSWORD": "admin123",
		},
		WaitingFor: wait.ForLog("Waiting for connections").WithStartupTimeout(mongoContainerStartupTimeout),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            true, // Enable container reuse across test runs
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start MongoDB container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "27017")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	uri := fmt.Sprintf("mongodb://admin:admin123@%s", net.JoinHostPort(host, port.Port()))

	return &SharedMongoContainer{
		Container: container,
		URI:       uri,
		clients:   make([]*mongo.Client, 0),
	}, nil
}

// trackClient adds a client to the list for cleanup
func (c *SharedMongoContainer) trackClient(client *mongo.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clients = append(c.clients, client)
}

// SetupSharedTestMongoDB creates a test database using the shared MongoDB container.
// Each test gets its own isolated database within the shared container.
// This is much faster than starting a new container for each test.
func SetupSharedTestMongoDB(t *testing.T) *mongo.Database {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
	defer cancel()

	container, err := GetSharedMongoContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to get shared MongoDB container: %v", err)
	}

	// Connect to MongoDB
	client, err := mongo.Connect(options.Client().ApplyURI(container.URI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Track client for potential cleanup
	container.trackClient(client)

	// Ping with retries
	maxRetries := 5
	for i := range maxRetries {
		pingCtx, pingCancel := context.WithTimeout(context.Background(), mongoPingTimeout)
		err = client.Ping(pingCtx, nil)
		pingCancel()
		if err == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(pingRetryDelay)
		}
	}
	if err != nil {
		t.Fatalf("Failed to ping MongoDB after %d retries: %v", maxRetries, err)
	}

	// Create unique database name for test isolation
	dbName := generateTestDBName(t.Name())
	db := client.Database(dbName)

	// Cleanup: drop database and disconnect after test
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
		defer cleanupCancel()
		_ = db.Drop(cleanupCtx)
		_ = client.Disconnect(cleanupCtx)
	})

	return db
}

// SetupSharedTestMongoDBWithClient creates a test database and returns both client and database.
// Uses the shared MongoDB container for faster test execution.
func SetupSharedTestMongoDBWithClient(t *testing.T) (*mongo.Client, *mongo.Database) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
	defer cancel()

	container, err := GetSharedMongoContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to get shared MongoDB container: %v", err)
	}

	// Connect to MongoDB
	client, err := mongo.Connect(options.Client().ApplyURI(container.URI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Track client for potential cleanup
	container.trackClient(client)

	// Ping with retries
	maxRetries := 5
	for i := range maxRetries {
		pingCtx, pingCancel := context.WithTimeout(context.Background(), mongoPingTimeout)
		err = client.Ping(pingCtx, nil)
		pingCancel()
		if err == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(pingRetryDelay)
		}
	}
	if err != nil {
		t.Fatalf("Failed to ping MongoDB after %d retries: %v", maxRetries, err)
	}

	// Create unique database name for test isolation
	dbName := generateTestDBName(t.Name())
	db := client.Database(dbName)

	// Cleanup: drop database and disconnect after test
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
		defer cleanupCancel()
		_ = db.Drop(cleanupCtx)
		_ = client.Disconnect(cleanupCtx)
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
// Note: With Reuse=true, the container may persist for faster subsequent runs.
func CleanupSharedContainer() {
	if sharedContainer != nil && sharedContainer.Container != nil {
		ctx, cancel := context.WithTimeout(context.Background(), mongoContainerTerminateTimeout)
		defer cancel()
		_ = sharedContainer.Container.Terminate(ctx)
	}
}

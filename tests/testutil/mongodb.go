package testutil

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoDB test configuration constants
const (
	mongoCtxTimeout                = 10 * time.Second
	mongoContainerStartupTimeout   = 120 * time.Second
	mongoContainerTerminateTimeout = 5 * time.Second
	mongoPingTimeout               = 2 * time.Second
	maxTestNameLength              = 40
)

// MongoContainer represents MongoDB container for tests
type MongoContainer struct {
	Container testcontainers.Container
	URI       string
}

// SetupMongoContainer running MongoDB 6 in testcontainer.
// This function creates a New container for each call which is slow.
//
// Deprecated: Use GetSharedMongoContainer for better performance.
func SetupMongoContainer(ctx context.Context, t *testing.T) *MongoContainer {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "mongo:6.0",
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
	})
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %v", err)
	}

	// Get host and container port
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "27017")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	uri := fmt.Sprintf("mongodb://admin:admin123@%s", net.JoinHostPort(host, port.Port()))

	// Cleanup: stop container after test
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), mongoContainerTerminateTimeout)
		defer cancel()
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	})

	return &MongoContainer{
		Container: container,
		URI:       uri,
	}
}

// SetupTestMongoDB creates connection to test MongoDB using shared container.
// each test receives its own isolated database for safe parallel execution.
func SetupTestMongoDB(t *testing.T) *mongo.Database {
	t.Helper()

	// Use shared container for much faster test execution
	return SetupSharedTestMongoDB(t)
}

// SetupTestMongoDBWithClient creates connection to test MongoDB and returns client and database.
// uses shared container for faster tests.
func SetupTestMongoDBWithClient(t *testing.T) (*mongo.Client, *mongo.Database) {
	t.Helper()

	// Use shared container for much faster test execution
	return SetupSharedTestMongoDBWithClient(t)
}

// SetupTestMongoDBIsolated creates new container MongoDB for full isolation.
// Use this function only when needed full isolation of container.
// in most cases SetupTestMongoDB is sufficient (isolation at the database).
func SetupTestMongoDBIsolated(t *testing.T) *mongo.Database {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
	defer cancel()

	// Running MongoDB in container
	mongoContainer := SetupMongoContainer(ctx, t)

	// Connect to MongoDB
	client, err := mongo.Connect(options.Client().ApplyURI(mongoContainer.URI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// check connection with retries
	maxRetries := 5
	for i := range maxRetries {
		ctx, cancel := context.WithTimeout(context.Background(), mongoPingTimeout)
		err = client.Ping(ctx, nil)
		cancel()
		if err == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(time.Second)
		}
	}
	if err != nil {
		t.Fatalf("Failed to ping MongoDB after %d retries: %v", maxRetries, err)
	}

	// Creating test database with unique name
	dbName := "flowra_test_" + t.Name()
	db := client.Database(dbName)

	// Cleanup: delete database and disconnect after test
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
		defer cancel()
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	return db
}

// SetupTestMongoDBWithClientIsolated creates new container and returns client and database.
// Use only when needed full isolation of container.
func SetupTestMongoDBWithClientIsolated(t *testing.T) (*mongo.Client, *mongo.Database) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
	defer cancel()

	// Running MongoDB in container
	mongoContainer := SetupMongoContainer(ctx, t)

	// Connect to MongoDB
	client, err := mongo.Connect(options.Client().ApplyURI(mongoContainer.URI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// check connection with retries
	maxRetries := 5
	for i := range maxRetries {
		ctx, cancel := context.WithTimeout(context.Background(), mongoPingTimeout)
		err = client.Ping(ctx, nil)
		cancel()
		if err == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(time.Second)
		}
	}
	if err != nil {
		t.Fatalf("Failed to ping MongoDB after %d retries: %v", maxRetries, err)
	}

	// Creating test database with unique name
	// Use hash for long imen tests (MongoDB limit: 63 chars)
	testName := t.Name()
	if len(testName) > maxTestNameLength {
		// Take first 20 characters + hash of the rest
		hash := sha256.Sum256([]byte(testName))
		testName = testName[:20] + "_" + hex.EncodeToString(hash[:])[:12]
	}
	dbName := "flowra_test_" + testName
	db := client.Database(dbName)

	// Cleanup: delete database and disconnect after test
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
		defer cancel()
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	return client, db
}

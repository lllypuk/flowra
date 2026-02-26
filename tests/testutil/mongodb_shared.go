package testutil

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/infrastructure/mongodb"
)

// pingRetryDelay is the delay between ping retries when connecting to MongoDB.
const pingRetryDelay = 500 * time.Millisecond

// containerStartupTimeout is the timeout for starting a new MongoDB container.
const containerStartupTimeout = 120 * time.Second

// MongoDB container resource limits
const (
	containerMemoryLimit   = 512 * 1024 * 1024 // 512MB
	clientPingTimeout      = 2 * time.Second
	maxPoolSize            = 100
	minPoolSize            = 5
	indexCreationTimeout   = 30 * time.Second
	mongoReplicaSetName    = "rs0"
	replicaInitTimeout     = 45 * time.Second
	replicaInitAttemptTO   = 5 * time.Second
	replicaInitRetryDelay  = 500 * time.Millisecond
	mongoDisconnectTimeout = 2 * time.Second
)

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

	if !needsCreation && needsContainerRecreation(ctx) {
		cleanupCrashedContainer()
		needsCreation = true
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

// needsContainerRecreation checks if the existing container needs to be recreated
func needsContainerRecreation(ctx context.Context) bool {
	running, err := isContainerRunning(ctx, sharedContainer.Container)
	return err != nil || !running
}

// cleanupCrashedContainer terminates a crashed container and disconnects the client
func cleanupCrashedContainer() {
	if sharedContainer.Container != nil {
		terminateCtx, cancel := context.WithTimeout(context.Background(), mongoContainerTerminateTimeout)
		_ = sharedContainer.Container.Terminate(terminateCtx)
		cancel()
	}
	if sharedContainer.client != nil {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
		_ = sharedContainer.client.Disconnect(disconnectCtx)
		cancel()
	}
	sharedContainer = nil
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
		// Limit memory to prevent OOM and crashes
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Memory = containerMemoryLimit
			hc.MemorySwap = containerMemoryLimit
		},
		// Enable single-node replica set so MongoDB transactions work in tests.
		Cmd: []string{"--replSet", mongoReplicaSetName, "--bind_ip_all", "--wiredTigerCacheSizeGB=0.25"},
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

	if err := initializeMongoReplicaSet(ctx, cont); err != nil {
		_ = cont.Terminate(ctx)
		return nil, fmt.Errorf("failed to initialize MongoDB replica set: %w", err)
	}

	uri := mongoTestURI(host, port.Port())

	return &SharedMongoContainer{
		Container: cont,
		URI:       uri,
	}, nil
}

func mongoTestURI(host, port string) string {
	host = normalizeMongoHost(host)
	return fmt.Sprintf(
		"mongodb://%s/admin?replicaSet=%s&directConnection=true",
		net.JoinHostPort(host, port),
		mongoReplicaSetName,
	)
}

func initializeMongoReplicaSet(ctx context.Context, cont testcontainers.Container) error {
	host, err := cont.Host(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve container host for replica set init: %w", err)
	}
	host = normalizeMongoHost(host)
	port, err := cont.MappedPort(ctx, "27017/tcp")
	if err != nil {
		return fmt.Errorf("failed to resolve container port for replica set init: %w", err)
	}

	bootstrapURI := fmt.Sprintf(
		"mongodb://%s/admin?directConnection=true",
		net.JoinHostPort(host, port.Port()),
	)

	deadline := time.Now().Add(replicaInitTimeout)
	var lastErr error

	for {
		if parentErr := ctx.Err(); parentErr != nil {
			return parentErr
		}

		attemptCtx, cancel := context.WithTimeout(ctx, replicaInitAttemptTO)
		client, connErr := mongo.Connect(options.Client().ApplyURI(bootstrapURI))
		if connErr != nil {
			cancel()
			lastErr = connErr
			if time.Now().After(deadline) {
				return lastErr
			}
			time.Sleep(replicaInitRetryDelay)
			continue
		}

		adminDB := client.Database("admin")
		initCmd := bson.D{
			{Key: "replSetInitiate", Value: bson.D{
				{Key: "_id", Value: mongoReplicaSetName},
				{Key: "members", Value: bson.A{
					bson.D{{Key: "_id", Value: 0}, {Key: "host", Value: "127.0.0.1:27017"}},
				}},
			}},
		}

		initErr := adminDB.RunCommand(attemptCtx, initCmd).Err()
		if initErr != nil && !isReplicaSetAlreadyInitialized(initErr) {
			lastErr = initErr
			disconnectMongoClient(client)
			cancel()
			if time.Now().After(deadline) {
				return fmt.Errorf("replSetInitiate failed: %w", lastErr)
			}
			time.Sleep(replicaInitRetryDelay)
			continue
		}

		var helloResp bson.M
		helloErr := adminDB.RunCommand(attemptCtx, bson.D{{Key: "hello", Value: 1}}).Decode(&helloResp)
		disconnectMongoClient(client)
		cancel()
		if helloErr != nil {
			lastErr = helloErr
			if time.Now().After(deadline) {
				return fmt.Errorf("hello command failed while waiting for primary: %w", lastErr)
			}
			time.Sleep(replicaInitRetryDelay)
			continue
		}

		if isWritablePrimary, _ := helloResp["isWritablePrimary"].(bool); isWritablePrimary {
			return nil
		}
		if isPrimary, _ := helloResp["ismaster"].(bool); isPrimary {
			return nil
		}

		lastErr = errors.New("replica set not primary yet")
		if time.Now().After(deadline) {
			return lastErr
		}
		time.Sleep(replicaInitRetryDelay)
	}
}

func isReplicaSetAlreadyInitialized(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "already initialized") || strings.Contains(msg, "AlreadyInitialized")
}

func normalizeMongoHost(host string) string {
	if host == "localhost" {
		return "127.0.0.1"
	}
	return host
}

func disconnectMongoClient(client *mongo.Client) {
	if client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), mongoDisconnectTimeout)
	defer cancel()
	_ = client.Disconnect(ctx)
}

// GetClient returns a shared MongoDB client, creating one if needed.
// The client is reused across all tests to avoid connection exhaustion.
func (c *SharedMongoContainer) GetClient(ctx context.Context) (*mongo.Client, error) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()

	// If client already exists and is connected, return it
	if c.client != nil {
		// Quick ping to verify connection
		pingCtx, cancel := context.WithTimeout(ctx, clientPingTimeout)
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
		SetMaxPoolSize(maxPoolSize).
		SetMinPoolSize(minPoolSize)

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
		indexCtx, indexCancel := context.WithTimeout(context.Background(), indexCreationTimeout)
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
		indexCtx, indexCancel := context.WithTimeout(context.Background(), indexCreationTimeout)
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

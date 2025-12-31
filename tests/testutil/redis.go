package testutil

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Redis test configuration constants
const (
	redisCtxTimeout                = 10 * time.Second
	redisContainerStartupTimeout   = 60 * time.Second
	redisContainerTerminateTimeout = 5 * time.Second
	redisPingTimeout               = 2 * time.Second
	redisPingRetryDelay            = 500 * time.Millisecond
	redisContainerMemoryLimit      = 128 * 1024 * 1024 // 128MB
	redisSharedPoolSize            = 50                // Pool size for shared container client
	redisTestPoolSize              = 10                // Pool size for individual test clients
)

// sharedRedisContainer holds the singleton Redis container and client
var (
	sharedRedisContainer   *SharedRedisContainer
	sharedRedisContainerMu sync.Mutex
)

// SharedRedisContainer represents a reusable Redis container for tests
type SharedRedisContainer struct {
	Container testcontainers.Container
	Addr      string
	client    *redis.Client
	clientMu  sync.Mutex
}

// GetSharedRedisContainer returns a singleton Redis container.
// The container is started once and reused across all tests.
func GetSharedRedisContainer(ctx context.Context) (*SharedRedisContainer, error) {
	sharedRedisContainerMu.Lock()
	defer sharedRedisContainerMu.Unlock()

	needsCreation := sharedRedisContainer == nil

	if !needsCreation && needsRedisContainerRecreation(ctx) {
		cleanupCrashedRedisContainer()
		needsCreation = true
	}

	if needsCreation {
		startupCtx, cancel := context.WithTimeout(context.Background(), redisContainerStartupTimeout)
		defer cancel()

		cont, err := startRedisContainer(startupCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to start Redis container: %w", err)
		}
		sharedRedisContainer = cont
	}

	return sharedRedisContainer, nil
}

// needsRedisContainerRecreation checks if the existing container needs to be recreated
func needsRedisContainerRecreation(ctx context.Context) bool {
	if sharedRedisContainer == nil || sharedRedisContainer.Container == nil {
		return true
	}
	running, err := isRedisContainerRunning(ctx, sharedRedisContainer.Container)
	return err != nil || !running
}

// cleanupCrashedRedisContainer terminates a crashed container and disconnects the client
func cleanupCrashedRedisContainer() {
	if sharedRedisContainer == nil {
		return
	}
	if sharedRedisContainer.Container != nil {
		terminateCtx, cancel := context.WithTimeout(context.Background(), redisContainerTerminateTimeout)
		_ = sharedRedisContainer.Container.Terminate(terminateCtx)
		cancel()
	}
	if sharedRedisContainer.client != nil {
		_ = sharedRedisContainer.client.Close()
	}
	sharedRedisContainer = nil
}

// isRedisContainerRunning checks if the container is still running
func isRedisContainerRunning(ctx context.Context, cont testcontainers.Container) (bool, error) {
	if cont == nil {
		return false, nil
	}
	state, err := cont.State(ctx)
	if err != nil {
		return false, err
	}
	return state.Running, nil
}

// startRedisContainer starts a new Redis container
func startRedisContainer(ctx context.Context) (*SharedRedisContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Memory = redisContainerMemoryLimit
			hc.MemorySwap = redisContainerMemoryLimit
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("Ready to accept connections").WithStartupTimeout(redisContainerStartupTimeout),
			wait.ForListeningPort("6379/tcp").WithStartupTimeout(redisContainerStartupTimeout),
		),
	}

	cont, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start Redis container: %w", err)
	}

	host, err := cont.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := cont.MappedPort(ctx, "6379")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	addr := net.JoinHostPort(host, port.Port())

	return &SharedRedisContainer{
		Container: cont,
		Addr:      addr,
	}, nil
}

// GetClient returns a shared Redis client, creating one if needed.
func (c *SharedRedisContainer) GetClient(ctx context.Context) (*redis.Client, error) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()

	if c.client != nil {
		pingCtx, cancel := context.WithTimeout(ctx, redisPingTimeout)
		err := c.client.Ping(pingCtx).Err()
		cancel()
		if err == nil {
			return c.client, nil
		}
		_ = c.client.Close()
		c.client = nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     c.Addr,
		PoolSize: redisSharedPoolSize,
	})

	maxRetries := 5
	var pingErr error
	for i := range maxRetries {
		pingCtx, pingCancel := context.WithTimeout(context.Background(), redisPingTimeout)
		pingErr = client.Ping(pingCtx).Err()
		pingCancel()
		if pingErr == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(redisPingRetryDelay)
		}
	}
	if pingErr != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to ping Redis after %d retries: %w", maxRetries, pingErr)
	}

	c.client = client
	return client, nil
}

// SetupTestRedis creates a Redis client using the shared container.
// This is the recommended way to get a Redis client for tests.
func SetupTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeout)
	defer cancel()

	cont, err := GetSharedRedisContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to get shared Redis container: %v", err)
	}

	// Verify container is healthy by getting a client (this also warms up the connection)
	if _, err := cont.GetClient(ctx); err != nil {
		t.Fatalf("Failed to get Redis client: %v", err)
	}

	// Create a new client for this test to allow independent cleanup
	testClient := redis.NewClient(&redis.Options{
		Addr:     cont.Addr,
		PoolSize: redisTestPoolSize,
	})

	// Verify connection
	if err := testClient.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to ping Redis: %v", err)
	}

	// Cleanup: flush DB and close client after test
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), redisCtxTimeout)
		defer cleanupCancel()
		_ = testClient.FlushDB(cleanupCtx).Err()
		_ = testClient.Close()
	})

	return testClient
}

// SetupTestRedisWithPrefix creates a Redis client and returns a unique key prefix for test isolation.
// This allows multiple tests to run in parallel without key conflicts.
func SetupTestRedisWithPrefix(t *testing.T) (*redis.Client, string) {
	t.Helper()

	client := SetupTestRedis(t)
	prefix := fmt.Sprintf("test:%s:", t.Name())

	return client, prefix
}

// SetupTestRedisIsolated creates a new Redis container for complete isolation.
// Use this only when you need a completely clean Redis instance.
func SetupTestRedisIsolated(t *testing.T) *redis.Client {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), redisContainerStartupTimeout)
	defer cancel()

	cont, err := startRedisContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start Redis container: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: cont.Addr,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to ping Redis: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
		terminateCtx, terminateCancel := context.WithTimeout(context.Background(), redisContainerTerminateTimeout)
		defer terminateCancel()
		_ = cont.Container.Terminate(terminateCtx)
	})

	return client
}

// CleanupSharedRedisContainer terminates the shared container.
// This is typically called from TestMain or when all tests are done.
func CleanupSharedRedisContainer() {
	sharedRedisContainerMu.Lock()
	defer sharedRedisContainerMu.Unlock()

	if sharedRedisContainer != nil {
		if sharedRedisContainer.client != nil {
			_ = sharedRedisContainer.client.Close()
		}
		if sharedRedisContainer.Container != nil {
			ctx, cancel := context.WithTimeout(context.Background(), redisContainerTerminateTimeout)
			defer cancel()
			_ = sharedRedisContainer.Container.Terminate(ctx)
		}
		sharedRedisContainer = nil
	}
}

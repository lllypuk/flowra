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

// MongoContainer represents контейнер MongoDB for tests
type MongoContainer struct {
	Container testcontainers.Container
	URI       string
}

// SetupMongoContainer runningает MongoDB 6 in testcontainer.
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

	// Получаем хост and порт контейнера
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "27017")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	uri := fmt.Sprintf("mongodb://admin:admin123@%s", net.JoinHostPort(host, port.Port()))

	// Cleanup: останавливаем контейнер after теста
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

// SetupTestMongoDB creates подключение to тестовой MongoDB с использованием shared контейнера.
// each test receivает свою изолированную database for безопасного pairллельного выполнения.
func SetupTestMongoDB(t *testing.T) *mongo.Database {
	t.Helper()

	// Use shared container for much faster test execution
	return SetupSharedTestMongoDB(t)
}

// SetupTestMongoDBWithClient creates подключение to тестовой MongoDB and returns клиент and database.
// uses shared контейнер for ускорения tests.
func SetupTestMongoDBWithClient(t *testing.T) (*mongo.Client, *mongo.Database) {
	t.Helper()

	// Use shared container for much faster test execution
	return SetupSharedTestMongoDBWithClient(t)
}

// SetupTestMongoDBIsolated creates New контейнер MongoDB for полной изоляции.
// Используйте эту функцию only when нужна полная изоляция контейнера.
// in большинстве случаев SetupTestMongoDB достаточно (изоляция on уровне database).
func SetupTestMongoDBIsolated(t *testing.T) *mongo.Database {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
	defer cancel()

	// Runningаем MongoDB in контейнере
	mongoContainer := SetupMongoContainer(ctx, t)

	// Подключаемся to MongoDB
	client, err := mongo.Connect(options.Client().ApplyURI(mongoContainer.URI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// check соединения с ретраями
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

	// Creating тестовую database с uniqueым именем
	dbName := "flowra_test_" + t.Name()
	db := client.Database(dbName)

	// Cleanup: удаляем database and отключаемся after теста
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
		defer cancel()
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	return db
}

// SetupTestMongoDBWithClientIsolated creates New контейнер and returns клиент and database.
// Используйте only when нужна полная изоляция контейнера.
func SetupTestMongoDBWithClientIsolated(t *testing.T) (*mongo.Client, *mongo.Database) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
	defer cancel()

	// Runningаем MongoDB in контейнере
	mongoContainer := SetupMongoContainer(ctx, t)

	// Подключаемся to MongoDB
	client, err := mongo.Connect(options.Client().ApplyURI(mongoContainer.URI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// check соединения с ретраями
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

	// Creating тестовую database с uniqueым именем
	// Используем hash for длинных имен tests (MongoDB limit: 63 chars)
	testName := t.Name()
	if len(testName) > maxTestNameLength {
		// Берем первые 20 символов + hash остального
		hash := sha256.Sum256([]byte(testName))
		testName = testName[:20] + "_" + hex.EncodeToString(hash[:])[:12]
	}
	dbName := "flowra_test_" + testName
	db := client.Database(dbName)

	// Cleanup: удаляем database and отключаемся after теста
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
		defer cancel()
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	return client, db
}

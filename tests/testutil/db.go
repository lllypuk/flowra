//go:build integration

package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/infrastructure/mongodb"
)

// SetupTestDatabase creates тестовое подключение to MongoDB
func SetupTestDatabase(t *testing.T) *mongo.Database {
	t.Helper()

	mongoURI := os.Getenv("TEST_MONGODB_URI")
	if mongoURI == "" {
		t.Skip("TEST_MONGODB_URI not set, skipping integration test")
	}

	ctx := context.Background()
	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Checking подключение
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Creating uniqueое имя базы данных for изоляции tests
	dbName := fmt.Sprintf("test_%s", sanitizeDatabaseName(t.Name()))
	db := client.Database(dbName)

	// Creating all индексы for production-like testing
	if err := mongodb.CreateAllIndexes(ctx, db); err != nil {
		t.Fatalf("Failed to create indexes: %v", err)
	}

	return db
}

// TeardownTestDatabase удаляет тестовую базу данных
func TeardownTestDatabase(t *testing.T, db *mongo.Database) {
	t.Helper()

	ctx := context.Background()
	if err := db.Drop(ctx); err != nil {
		t.Logf("Warning: Failed to drop test database: %v", err)
	}

	if err := db.Client().Disconnect(ctx); err != nil {
		t.Logf("Warning: Failed to disconnect from MongoDB: %v", err)
	}
}

// sanitizeDatabaseName clears имя теста for использования in качестве имени базы данных
func sanitizeDatabaseName(name string) string {
	// Заменяем недопустимые символы on подчеркивания
	result := ""
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			result += string(ch)
		} else {
			result += "_"
		}
	}
	return result
}

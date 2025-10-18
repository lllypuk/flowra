//go:build integration

package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// SetupTestDatabase создает тестовое подключение к MongoDB
func SetupTestDatabase(t *testing.T) *mongo.Database {
	t.Helper()

	mongoURI := os.Getenv("TEST_MONGODB_URI")
	if mongoURI == "" {
		t.Skip("TEST_MONGODB_URI not set, skipping integration test")
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Проверяем подключение
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Создаем уникальное имя базы данных для изоляции тестов
	dbName := fmt.Sprintf("test_%s", sanitizeDatabaseName(t.Name()))
	db := client.Database(dbName)

	// Создаем индексы для event store
	if err := createIndexes(ctx, db); err != nil {
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

// sanitizeDatabaseName очищает имя теста для использования в качестве имени базы данных
func sanitizeDatabaseName(name string) string {
	// Заменяем недопустимые символы на подчеркивания
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

// createIndexes создает необходимые индексы для event store
func createIndexes(ctx context.Context, db *mongo.Database) error {
	eventsCollection := db.Collection("events")

	// Уникальный индекс для aggregate_id + version
	_, err := eventsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    map[string]interface{}{"aggregate_id": 1, "version": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create unique index: %w", err)
	}

	// Индекс для aggregate_id
	_, err = eventsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"aggregate_id": 1},
	})
	if err != nil {
		return fmt.Errorf("failed to create aggregate_id index: %w", err)
	}

	// Индекс для aggregate_type
	_, err = eventsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"aggregate_type": 1},
	})
	if err != nil {
		return fmt.Errorf("failed to create aggregate_type index: %w", err)
	}

	// Индекс для occurred_at
	_, err = eventsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"occurred_at": 1},
	})
	if err != nil {
		return fmt.Errorf("failed to create occurred_at index: %w", err)
	}

	return nil
}

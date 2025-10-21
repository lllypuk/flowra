package testutil

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const mongoCtxTimeout = 5 * time.Second

// SetupTestMongoDB создает подключение к тестовой MongoDB
// Использует testcontainers или docker-compose
func SetupTestMongoDB(t *testing.T) *mongo.Database {
	ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
	defer cancel()

	// Для интеграционных тестов используем отдельную БД
	uri := "mongodb://admin:admin123@localhost:27017" //nolint:gosec // Пример URI, заменить на реальный
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Проверка соединения
	err = client.Ping(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to ping MongoDB: %v", err)
	}

	// Создаем тестовую БД с уникальным именем
	dbName := "flowra_test_" + t.Name()
	db := client.Database(dbName)

	// Cleanup: удаляем БД после теста
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), mongoCtxTimeout)
		defer cancel()
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	return db
}

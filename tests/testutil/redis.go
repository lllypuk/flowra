package testutil

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
)

// SetupTestRedis создает подключение к тестовому Redis
func SetupTestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // используем отдельную БД для тестов
	})

	ctx := context.Background()
	err := client.Ping(ctx).Err()
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Cleanup: очищаем БД после теста
	t.Cleanup(func() {
		_ = client.FlushDB(ctx).Err()
		_ = client.Close()
	})

	return client
}

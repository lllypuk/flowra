# Task 1.1.3: Redis Repositories

**Приоритет:** 🟡 MEDIUM
**Статус:** Ready
**Время:** 2 дня
**Зависимости:** Нет (может выполняться параллельно с 1.1.2)

---

## Проблема

Sessions, idempotency tracking, и caching требуют быстрого key-value store. Redis идеален для этих use cases.

---

## Цель

Реализовать Redis repositories для:
1. Session management (user sessions, JWT tokens)
2. Idempotency tracking (prevent duplicate event processing)
3. Cache (опционально для MVP)

---

## Файлы для создания

```
internal/infrastructure/repository/redis/
├── session_repository.go
├── session_repository_test.go
├── idempotency_repository.go
├── idempotency_repository_test.go
├── cache_repository.go      (optional для MVP)
└── cache_repository_test.go
```

---

## 1. SessionRepository

### Interface

```go
package redis

type SessionData struct {
    UserID        uuid.UUID
    Username      string
    AccessToken   string
    RefreshToken  string
    ExpiresAt     time.Time
}

type SessionRepository interface {
    Save(ctx context.Context, sessionID string, data *SessionData, ttl time.Duration) error
    Load(ctx context.Context, sessionID string) (*SessionData, error)
    Delete(ctx context.Context, sessionID string) error
    Extend(ctx context.Context, sessionID string, ttl time.Duration) error
}
```

### Implementation

```go
type RedisSessionRepository struct {
    client *redis.Client
}

func NewRedisSessionRepository(client *redis.Client) *RedisSessionRepository {
    return &RedisSessionRepository{client: client}
}

func (r *RedisSessionRepository) Save(ctx context.Context, sessionID string, data *SessionData, ttl time.Duration) error {
    key := fmt.Sprintf("session:%s", sessionID)

    // Serialize session data
    value, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("failed to marshal session: %w", err)
    }

    // Save with TTL
    return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisSessionRepository) Load(ctx context.Context, sessionID string) (*SessionData, error) {
    key := fmt.Sprintf("session:%s", sessionID)

    value, err := r.client.Get(ctx, key).Result()
    if err == redis.Nil {
        return nil, &shared.NotFoundError{Resource: "Session", ID: sessionID}
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get session: %w", err)
    }

    var data SessionData
    if err := json.Unmarshal([]byte(value), &data); err != nil {
        return nil, fmt.Errorf("failed to unmarshal session: %w", err)
    }

    return &data, nil
}

func (r *RedisSessionRepository) Delete(ctx context.Context, sessionID string) error {
    key := fmt.Sprintf("session:%s", sessionID)
    return r.client.Del(ctx, key).Err()
}

func (r *RedisSessionRepository) Extend(ctx context.Context, sessionID string, ttl time.Duration) error {
    key := fmt.Sprintf("session:%s", sessionID)
    return r.client.Expire(ctx, key, ttl).Err()
}
```

### Usage

```go
// Save session after login
sessionID := uuid.New().String()
sessionData := &SessionData{
    UserID:       userID,
    Username:     "john.doe",
    AccessToken:  "jwt-token",
    RefreshToken: "refresh-token",
    ExpiresAt:    time.Now().Add(24 * time.Hour),
}

sessionRepo.Save(ctx, sessionID, sessionData, 24*time.Hour)

// Load session
session, err := sessionRepo.Load(ctx, sessionID)

// Extend session
sessionRepo.Extend(ctx, sessionID, 24*time.Hour)

// Delete session (logout)
sessionRepo.Delete(ctx, sessionID)
```

---

## 2. IdempotencyRepository

### Purpose

Prevent duplicate event processing when events are retried or replayed.

### Interface

```go
type IdempotencyRepository interface {
    IsProcessed(ctx context.Context, eventID uuid.UUID) (bool, error)
    MarkAsProcessed(ctx context.Context, eventID uuid.UUID, ttl time.Duration) error
}
```

### Implementation

```go
type RedisIdempotencyRepository struct {
    client *redis.Client
}

func (r *RedisIdempotencyRepository) IsProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
    key := fmt.Sprintf("idempotency:%s", eventID.String())

    exists, err := r.client.Exists(ctx, key).Result()
    if err != nil {
        return false, fmt.Errorf("failed to check idempotency: %w", err)
    }

    return exists > 0, nil
}

func (r *RedisIdempotencyRepository) MarkAsProcessed(ctx context.Context, eventID uuid.UUID, ttl time.Duration) error {
    key := fmt.Sprintf("idempotency:%s", eventID.String())

    // Store simple marker with TTL
    return r.client.Set(ctx, key, "processed", ttl).Err()
}
```

### Usage in Event Handler

```go
func (h *EventHandler) Handle(ctx context.Context, event DomainEvent) error {
    // 1. Check if already processed
    processed, err := h.idempotencyRepo.IsProcessed(ctx, event.EventID())
    if err != nil {
        return err
    }

    if processed {
        // Already handled, skip
        return nil
    }

    // 2. Process event
    err = h.processEvent(ctx, event)
    if err != nil {
        return err
    }

    // 3. Mark as processed (TTL = 7 days)
    return h.idempotencyRepo.MarkAsProcessed(ctx, event.EventID(), 7*24*time.Hour)
}
```

---

## 3. CacheRepository (Optional)

### Interface

```go
type CacheRepository interface {
    Get(ctx context.Context, key string) (interface{}, error)
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    DeletePattern(ctx context.Context, pattern string) error
}
```

### Usage

```go
// Cache user data
cacheKey := fmt.Sprintf("user:%s", userID)
cacheRepo.Set(ctx, cacheKey, user, 1*time.Hour)

// Get from cache
cachedUser, err := cacheRepo.Get(ctx, cacheKey)

// Invalidate cache
cacheRepo.DeletePattern(ctx, "user:*")
```

---

## Тестирование

### Unit Tests

```go
func TestSessionRepository_SaveAndLoad(t *testing.T) {
    // Setup Redis (testcontainers or miniredis)
    client := testutil.SetupRedis(t)
    repo := redis.NewRedisSessionRepository(client)

    sessionID := uuid.New().String()
    sessionData := &redis.SessionData{
        UserID:   uuid.New(),
        Username: "test",
    }

    // Save
    err := repo.Save(ctx, sessionID, sessionData, 10*time.Second)
    require.NoError(t, err)

    // Load
    loaded, err := repo.Load(ctx, sessionID)
    require.NoError(t, err)
    assert.Equal(t, sessionData.UserID, loaded.UserID)

    // Wait for TTL expiration
    time.Sleep(11 * time.Second)

    // Should be expired
    _, err = repo.Load(ctx, sessionID)
    assert.Error(t, err)
}

func TestIdempotencyRepository_PreventDuplicates(t *testing.T) {
    // ...
}
```

---

## Критерии успеха

- ✅ **SessionRepository works** (CRUD + TTL)
- ✅ **IdempotencyRepository prevents duplicates**
- ✅ **TTL expiration verified**
- ✅ **Test coverage >80%**
- ✅ **Integration tests with Redis**

---

## Следующий шаг

→ **Task 1.2.1: Redis Event Bus**

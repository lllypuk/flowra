# 05: User Sync

**Приоритет:** 🟢 Medium
**Статус:** ✅ Выполнено
**Зависит от:** [04-group-management.md](04-group-management.md)

---

## Описание

Реализовать периодическую синхронизацию пользователей между Keycloak и локальной базой данных. Это обеспечит актуальность данных пользователей и обработку удалений в Keycloak.

---

## Текущее состояние

Синхронизация происходит только при login:

```go
// AuthService.Login
user, err := s.findOrCreateUser(ctx, userInfo)
s.updateExistingUserIfNeeded(ctx, user, userInfo)
```

**Проблемы:**
- Изменения профиля видны только после re-login
- Удалённые в Keycloak пользователи остаются активными
- Нет синхронизации ролей/групп
- Нет batch-обновлений

---

## Решение

### Sync Worker

```
┌─────────────────────────────────────────────────────────────┐
│                    User Sync Worker                          │
│                    (runs every 15 min)                       │
└──────────────────────────┬──────────────────────────────────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           v               v               v
    ┌────────────┐  ┌────────────┐  ┌────────────┐
    │  Fetch     │  │  Compare   │  │  Update    │
    │  Keycloak  │  │  Changes   │  │  MongoDB   │
    │  Users     │  │            │  │            │
    └────────────┘  └────────────┘  └────────────┘
```

---

## Файлы

```
internal/worker/
├── user_sync.go           # Sync worker
├── user_sync_test.go      # Tests
└── scheduler.go           # Cron scheduler

internal/infrastructure/keycloak/
├── user_client.go         # User admin API client
└── user_client_test.go    # Tests
```

---

## Реализация

### User Client

```go
// internal/infrastructure/keycloak/user_client.go

package keycloak

type UserClient struct {
    config       UserClientConfig
    tokenManager *AdminTokenManager
    httpClient   *http.Client
}

type UserClientConfig struct {
    KeycloakURL string
    Realm       string
}

// KeycloakUser represents user from Keycloak
type KeycloakUser struct {
    ID              string            `json:"id"`
    Username        string            `json:"username"`
    Email           string            `json:"email"`
    EmailVerified   bool              `json:"emailVerified"`
    FirstName       string            `json:"firstName"`
    LastName        string            `json:"lastName"`
    Enabled         bool              `json:"enabled"`
    CreatedTimestamp int64            `json:"createdTimestamp"`
    Attributes      map[string][]string `json:"attributes"`
}

func NewUserClient(config UserClientConfig, tokenManager *AdminTokenManager) *UserClient {
    return &UserClient{
        config:       config,
        tokenManager: tokenManager,
        httpClient:   &http.Client{Timeout: 60 * time.Second},
    }
}

// ListUsers returns all users from Keycloak with pagination
func (c *UserClient) ListUsers(ctx context.Context, first, max int) ([]KeycloakUser, error) {
    token, err := c.tokenManager.GetToken(ctx)
    if err != nil {
        return nil, err
    }

    url := fmt.Sprintf("%s/admin/realms/%s/users?first=%d&max=%d",
        c.config.KeycloakURL, c.config.Realm, first, max)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("list users failed: %s", body)
    }

    var users []KeycloakUser
    if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
        return nil, err
    }

    return users, nil
}

// GetUser returns single user by ID
func (c *UserClient) GetUser(ctx context.Context, userID string) (*KeycloakUser, error) {
    token, err := c.tokenManager.GetToken(ctx)
    if err != nil {
        return nil, err
    }

    url := fmt.Sprintf("%s/admin/realms/%s/users/%s",
        c.config.KeycloakURL, c.config.Realm, userID)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusNotFound {
        return nil, ErrUserNotFound
    }

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("get user failed: %s", body)
    }

    var user KeycloakUser
    if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
        return nil, err
    }

    return &user, nil
}

// CountUsers returns total user count
func (c *UserClient) CountUsers(ctx context.Context) (int, error) {
    token, err := c.tokenManager.GetToken(ctx)
    if err != nil {
        return 0, err
    }

    url := fmt.Sprintf("%s/admin/realms/%s/users/count",
        c.config.KeycloakURL, c.config.Realm)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return 0, err
    }
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    var count int
    if err := json.NewDecoder(resp.Body).Decode(&count); err != nil {
        return 0, err
    }

    return count, nil
}

var ErrUserNotFound = errors.New("user not found")
```

### Sync Worker

```go
// internal/worker/user_sync.go

package worker

import (
    "context"
    "time"

    "go.uber.org/zap"
)

type UserSyncWorker struct {
    keycloakClient KeycloakUserClient
    userRepo       UserRepository
    logger         *zap.Logger
    config         UserSyncConfig
}

type UserSyncConfig struct {
    Interval  time.Duration // Sync interval
    BatchSize int           // Users per batch
    Enabled   bool          // Feature flag
}

type KeycloakUserClient interface {
    ListUsers(ctx context.Context, first, max int) ([]keycloak.KeycloakUser, error)
    CountUsers(ctx context.Context) (int, error)
}

type UserRepository interface {
    FindByExternalID(ctx context.Context, externalID string) (*domain.User, error)
    Save(ctx context.Context, user *domain.User) error
    Deactivate(ctx context.Context, userID uuid.UUID) error
    ListExternalIDs(ctx context.Context) ([]string, error)
}

func NewUserSyncWorker(
    keycloakClient KeycloakUserClient,
    userRepo UserRepository,
    logger *zap.Logger,
    config UserSyncConfig,
) *UserSyncWorker {
    return &UserSyncWorker{
        keycloakClient: keycloakClient,
        userRepo:       userRepo,
        logger:         logger,
        config:         config,
    }
}

func (w *UserSyncWorker) Run(ctx context.Context) error {
    if !w.config.Enabled {
        w.logger.Info("User sync disabled")
        return nil
    }

    ticker := time.NewTicker(w.config.Interval)
    defer ticker.Stop()

    // Run immediately on start
    if err := w.sync(ctx); err != nil {
        w.logger.Error("Initial user sync failed", zap.Error(err))
    }

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := w.sync(ctx); err != nil {
                w.logger.Error("User sync failed", zap.Error(err))
            }
        }
    }
}

func (w *UserSyncWorker) sync(ctx context.Context) error {
    start := time.Now()
    w.logger.Info("Starting user sync")

    // Get total count
    totalCount, err := w.keycloakClient.CountUsers(ctx)
    if err != nil {
        return fmt.Errorf("failed to count users: %w", err)
    }

    // Track seen users
    seenExternalIDs := make(map[string]bool)

    // Fetch and sync in batches
    var synced, created, updated int
    for offset := 0; offset < totalCount; offset += w.config.BatchSize {
        users, err := w.keycloakClient.ListUsers(ctx, offset, w.config.BatchSize)
        if err != nil {
            return fmt.Errorf("failed to list users at offset %d: %w", offset, err)
        }

        for _, kcUser := range users {
            seenExternalIDs[kcUser.ID] = true

            result, err := w.syncUser(ctx, kcUser)
            if err != nil {
                w.logger.Warn("Failed to sync user",
                    zap.String("keycloak_id", kcUser.ID),
                    zap.Error(err))
                continue
            }

            synced++
            if result == syncResultCreated {
                created++
            } else if result == syncResultUpdated {
                updated++
            }
        }
    }

    // Deactivate users not in Keycloak
    deactivated, err := w.deactivateMissingUsers(ctx, seenExternalIDs)
    if err != nil {
        w.logger.Warn("Failed to deactivate missing users", zap.Error(err))
    }

    w.logger.Info("User sync completed",
        zap.Int("synced", synced),
        zap.Int("created", created),
        zap.Int("updated", updated),
        zap.Int("deactivated", deactivated),
        zap.Duration("duration", time.Since(start)))

    return nil
}

type syncResult int

const (
    syncResultNoChange syncResult = iota
    syncResultCreated
    syncResultUpdated
)

func (w *UserSyncWorker) syncUser(ctx context.Context, kcUser keycloak.KeycloakUser) (syncResult, error) {
    existing, err := w.userRepo.FindByExternalID(ctx, kcUser.ID)
    if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
        return syncResultNoChange, err
    }

    if existing == nil {
        // Create new user
        user := &domain.User{
            ID:          uuid.New(),
            ExternalID:  kcUser.ID,
            Username:    kcUser.Username,
            Email:       kcUser.Email,
            DisplayName: fmt.Sprintf("%s %s", kcUser.FirstName, kcUser.LastName),
            AvatarURL:   "", // Not available from Keycloak
            IsActive:    kcUser.Enabled,
            CreatedAt:   time.Now(),
            UpdatedAt:   time.Now(),
        }
        if err := w.userRepo.Save(ctx, user); err != nil {
            return syncResultNoChange, err
        }
        return syncResultCreated, nil
    }

    // Check if update needed
    needsUpdate := false
    if existing.Username != kcUser.Username {
        existing.Username = kcUser.Username
        needsUpdate = true
    }
    if existing.Email != kcUser.Email {
        existing.Email = kcUser.Email
        needsUpdate = true
    }
    displayName := fmt.Sprintf("%s %s", kcUser.FirstName, kcUser.LastName)
    if existing.DisplayName != displayName {
        existing.DisplayName = displayName
        needsUpdate = true
    }
    if existing.IsActive != kcUser.Enabled {
        existing.IsActive = kcUser.Enabled
        needsUpdate = true
    }

    if needsUpdate {
        existing.UpdatedAt = time.Now()
        if err := w.userRepo.Save(ctx, existing); err != nil {
            return syncResultNoChange, err
        }
        return syncResultUpdated, nil
    }

    return syncResultNoChange, nil
}

func (w *UserSyncWorker) deactivateMissingUsers(ctx context.Context, seenExternalIDs map[string]bool) (int, error) {
    localExternalIDs, err := w.userRepo.ListExternalIDs(ctx)
    if err != nil {
        return 0, err
    }

    var deactivated int
    for _, externalID := range localExternalIDs {
        if !seenExternalIDs[externalID] {
            user, err := w.userRepo.FindByExternalID(ctx, externalID)
            if err != nil {
                continue
            }
            if user.IsActive {
                if err := w.userRepo.Deactivate(ctx, user.ID); err != nil {
                    w.logger.Warn("Failed to deactivate user",
                        zap.String("user_id", user.ID.String()),
                        zap.Error(err))
                    continue
                }
                deactivated++
            }
        }
    }

    return deactivated, nil
}
```

---

## Конфигурация

```yaml
# config.yaml
sync:
  users:
    enabled: true
    interval: "15m"
    batch_size: 100
```

---

## Чеклист

### User Client
- [x] `ListUsers` с пагинацией
- [x] `GetUser` по ID
- [x] `CountUsers`
- [x] Error handling

### Sync Worker
- [x] Периодический запуск
- [x] Batch processing
- [x] Create new users
- [x] Update changed users
- [x] Deactivate missing users
- [x] Logging и metrics

### Testing
- [x] Unit tests для worker
- [ ] Integration test с Keycloak

### Integration
- [x] Worker запускается в cmd/worker
- [x] Graceful shutdown
- [x] Feature flag для включения/выключения

---

## Критерии приёмки

- [x] Новые пользователи из Keycloak создаются в MongoDB
- [x] Изменения профиля синхронизируются
- [x] Удалённые/disabled пользователи деактивируются
- [x] Sync запускается каждые 15 минут
- [x] Логи показывают прогресс sync
- [x] Sync не блокирует приложение

---

## Зависимости

### Входящие
- [04-group-management.md](04-group-management.md) — Admin token manager

### Исходящие
- Нет

---

*Обновлено: 2026-01-06*

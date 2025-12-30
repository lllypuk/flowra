# Task 03: Завершение User Repository

## Цель

Завершить реализацию MongoDB репозитория для User, добавив недостающий метод `documentToUser` и функцию `Restore` в domain layer.

## Контекст

Репозиторий `MongoUserRepository` уже создан в `internal/infrastructure/repository/mongodb/user_repository.go`, но метод `documentToUser` возвращает ошибку:

```go
func (r *MongoUserRepository) documentToUser(doc *userDocument) (*userdomain.User, error) {
    // TODO: Полная реализация требует наличия constructor или setter методов в domain/user
    return nil, errors.New("documentToUser requires domain setter methods - not yet implemented")
}
```

Для корректной работы необходимо:
1. Добавить функцию `Restore` в domain/user для восстановления User из полей
2. Реализовать `documentToUser` с использованием этой функции
3. Добавить метод `Exists` для проверки существования пользователя

## Зависимости

### Уже реализовано

- `internal/infrastructure/repository/mongodb/user_repository.go` — частичная реализация
- `internal/domain/user/user.go` — domain model User
- `internal/application/user/repository.go` — интерфейсы репозитория
- `internal/application/shared/user_repository.go` — shared интерфейс с `Exists`

### Требуется изменить

1. `internal/domain/user/user.go` — добавить функцию `Restore`
2. `internal/infrastructure/repository/mongodb/user_repository.go` — реализовать `documentToUser` и `Exists`

## Детальное описание

### 1. Добавить Restore функцию в domain

Изменить `internal/domain/user/user.go`:

```go
// Restore восстанавливает User из сохраненных полей (для persistence layer)
// Эта функция должна использоваться ТОЛЬКО репозиторием для восстановления
// сущности из хранилища. Для создания нового пользователя используйте NewUser.
func Restore(
    id uuid.UUID,
    externalID *string,
    username string,
    email string,
    displayName string,
    isSystemAdmin bool,
    createdAt time.Time,
    updatedAt time.Time,
) *User {
    var extID string
    if externalID != nil {
        extID = *externalID
    }

    return &User{
        id:            id,
        externalID:    extID,
        username:      username,
        email:         email,
        displayName:   displayName,
        isSystemAdmin: isSystemAdmin,
        createdAt:     createdAt,
        updatedAt:     updatedAt,
    }
}
```

### 2. Реализовать documentToUser

Изменить `internal/infrastructure/repository/mongodb/user_repository.go`:

```go
// documentToUser преобразует Document в User
func (r *MongoUserRepository) documentToUser(doc *userDocument) (*userdomain.User, error) {
    if doc == nil {
        return nil, errs.ErrInvalidInput
    }

    // Парсим UUID
    userID := uuid.UUID(doc.UserID)
    if userID.IsZero() {
        return nil, fmt.Errorf("invalid user_id: %s", doc.UserID)
    }

    // Восстанавливаем User из полей
    user := userdomain.Restore(
        userID,
        doc.KeycloakID,
        doc.Username,
        doc.Email,
        doc.DisplayName,
        doc.IsSystemAdmin,
        doc.CreatedAt,
        doc.UpdatedAt,
    )

    return user, nil
}
```

### 3. Добавить метод Exists

Метод `Exists` требуется интерфейсом `shared.UserRepository`:

```go
// Exists проверяет, существует ли пользователь с заданным ID
func (r *MongoUserRepository) Exists(ctx context.Context, userID uuid.UUID) (bool, error) {
    if userID.IsZero() {
        return false, errs.ErrInvalidInput
    }

    filter := bson.M{"user_id": userID.String()}
    count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
    if err != nil {
        return false, HandleMongoError(err, "user")
    }

    return count > 0, nil
}
```

### 4. Добавить метод ExistsByUsername

Полезный дополнительный метод:

```go
// ExistsByUsername проверяет, существует ли пользователь с заданным username
func (r *MongoUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
    if username == "" {
        return false, errs.ErrInvalidInput
    }

    filter := bson.M{"username": username}
    count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
    if err != nil {
        return false, HandleMongoError(err, "user")
    }

    return count > 0, nil
}
```

### 5. Добавить метод ExistsByEmail

```go
// ExistsByEmail проверяет, существует ли пользователь с заданным email
func (r *MongoUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
    if email == "" {
        return false, errs.ErrInvalidInput
    }

    filter := bson.M{"email": email}
    count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
    if err != nil {
        return false, HandleMongoError(err, "user")
    }

    return count > 0, nil
}
```

## Полный обновленный код user_repository.go

```go
package mongodb

import (
    "context"
    "fmt"

    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"

    "github.com/lllypuk/flowra/internal/domain/errs"
    userdomain "github.com/lllypuk/flowra/internal/domain/user"
    "github.com/lllypuk/flowra/internal/domain/uuid"
)

// MongoUserRepository реализует userapp.Repository (application layer interface)
type MongoUserRepository struct {
    collection *mongo.Collection
}

// NewMongoUserRepository создает новый MongoDB User Repository
func NewMongoUserRepository(collection *mongo.Collection) *MongoUserRepository {
    return &MongoUserRepository{
        collection: collection,
    }
}

// FindByID находит пользователя по ID
func (r *MongoUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*userdomain.User, error) {
    if id.IsZero() {
        return nil, errs.ErrInvalidInput
    }

    filter := bson.M{"user_id": id.String()}
    var doc userDocument
    err := r.collection.FindOne(ctx, filter).Decode(&doc)
    if err != nil {
        return nil, HandleMongoError(err, "user")
    }

    return r.documentToUser(&doc)
}

// FindByExternalID находит пользователя по ID из внешней системы аутентификации
func (r *MongoUserRepository) FindByExternalID(ctx context.Context, externalID string) (*userdomain.User, error) {
    if externalID == "" {
        return nil, errs.ErrInvalidInput
    }

    filter := bson.M{"keycloak_id": externalID}
    var doc userDocument
    err := r.collection.FindOne(ctx, filter).Decode(&doc)
    if err != nil {
        return nil, HandleMongoError(err, "user")
    }

    return r.documentToUser(&doc)
}

// FindByEmail находит пользователя по email
func (r *MongoUserRepository) FindByEmail(ctx context.Context, email string) (*userdomain.User, error) {
    if email == "" {
        return nil, errs.ErrInvalidInput
    }

    filter := bson.M{"email": email}
    var doc userDocument
    err := r.collection.FindOne(ctx, filter).Decode(&doc)
    if err != nil {
        return nil, HandleMongoError(err, "user")
    }

    return r.documentToUser(&doc)
}

// FindByUsername находит пользователя по username
func (r *MongoUserRepository) FindByUsername(ctx context.Context, username string) (*userdomain.User, error) {
    if username == "" {
        return nil, errs.ErrInvalidInput
    }

    filter := bson.M{"username": username}
    var doc userDocument
    err := r.collection.FindOne(ctx, filter).Decode(&doc)
    if err != nil {
        return nil, HandleMongoError(err, "user")
    }

    return r.documentToUser(&doc)
}

// Exists проверяет, существует ли пользователь с заданным ID
func (r *MongoUserRepository) Exists(ctx context.Context, userID uuid.UUID) (bool, error) {
    if userID.IsZero() {
        return false, errs.ErrInvalidInput
    }

    filter := bson.M{"user_id": userID.String()}
    count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
    if err != nil {
        return false, HandleMongoError(err, "user")
    }

    return count > 0, nil
}

// ExistsByUsername проверяет, существует ли пользователь с заданным username
func (r *MongoUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
    if username == "" {
        return false, errs.ErrInvalidInput
    }

    filter := bson.M{"username": username}
    count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
    if err != nil {
        return false, HandleMongoError(err, "user")
    }

    return count > 0, nil
}

// ExistsByEmail проверяет, существует ли пользователь с заданным email
func (r *MongoUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
    if email == "" {
        return false, errs.ErrInvalidInput
    }

    filter := bson.M{"email": email}
    count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
    if err != nil {
        return false, HandleMongoError(err, "user")
    }

    return count > 0, nil
}

// Save сохраняет пользователя
func (r *MongoUserRepository) Save(ctx context.Context, user *userdomain.User) error {
    if user == nil {
        return errs.ErrInvalidInput
    }

    if user.ID().IsZero() {
        return errs.ErrInvalidInput
    }

    doc := r.userToDocument(user)
    filter := bson.M{"user_id": user.ID().String()}
    update := bson.M{"$set": doc}
    opts := options.UpdateOne().SetUpsert(true)

    _, err := r.collection.UpdateOne(ctx, filter, update, opts)
    if err != nil {
        if mongo.IsDuplicateKeyError(err) {
            return errs.ErrAlreadyExists
        }
        return HandleMongoError(err, "user")
    }

    return nil
}

// Delete удаляет пользователя
func (r *MongoUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
    if id.IsZero() {
        return errs.ErrInvalidInput
    }

    filter := bson.M{"user_id": id.String()}
    result, err := r.collection.DeleteOne(ctx, filter)
    if err != nil {
        return HandleMongoError(err, "user")
    }

    if result.DeletedCount == 0 {
        return errs.ErrNotFound
    }

    return nil
}

// List возвращает список пользователей с пагинацией
func (r *MongoUserRepository) List(ctx context.Context, offset, limit int) ([]*userdomain.User, error) {
    return listDocuments(ctx, r.collection, offset, limit, r.documentToUser, "users")
}

// Count возвращает общее количество пользователей
func (r *MongoUserRepository) Count(ctx context.Context) (int, error) {
    count, err := r.collection.CountDocuments(ctx, bson.M{})
    if err != nil {
        return 0, HandleMongoError(err, "users")
    }

    return int(count), nil
}

// userDocument представляет структуру документа в MongoDB
type userDocument struct {
    UserID        string    `bson:"user_id"`
    KeycloakID    *string   `bson:"keycloak_id,omitempty"`
    Username      string    `bson:"username"`
    Email         string    `bson:"email"`
    DisplayName   string    `bson:"display_name"`
    IsSystemAdmin bool      `bson:"is_system_admin"`
    CreatedAt     time.Time `bson:"created_at"`
    UpdatedAt     time.Time `bson:"updated_at"`
}

// userToDocument преобразует User в Document
func (r *MongoUserRepository) userToDocument(user *userdomain.User) userDocument {
    doc := userDocument{
        UserID:        user.ID().String(),
        Username:      user.Username(),
        Email:         user.Email(),
        DisplayName:   user.DisplayName(),
        IsSystemAdmin: user.IsSystemAdmin(),
        CreatedAt:     user.CreatedAt(),
        UpdatedAt:     user.UpdatedAt(),
    }

    if externalID := user.ExternalID(); externalID != "" {
        doc.KeycloakID = &externalID
    }

    return doc
}

// documentToUser преобразует Document в User
func (r *MongoUserRepository) documentToUser(doc *userDocument) (*userdomain.User, error) {
    if doc == nil {
        return nil, errs.ErrInvalidInput
    }

    // Парсим UUID
    userID := uuid.UUID(doc.UserID)
    if userID.IsZero() {
        return nil, fmt.Errorf("invalid user_id: %s", doc.UserID)
    }

    // Восстанавливаем User из полей
    user := userdomain.Restore(
        userID,
        doc.KeycloakID,
        doc.Username,
        doc.Email,
        doc.DisplayName,
        doc.IsSystemAdmin,
        doc.CreatedAt,
        doc.UpdatedAt,
    )

    return user, nil
}
```

## Тестирование

### Дополнительные тесты для Exists

```go
func TestMongoUserRepository_Exists(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    coll := client.Database("test_db").Collection("users")
    repo := mongodb.NewMongoUserRepository(coll)

    // Create user
    userID := uuid.NewUUID()
    user := userdomain.NewUser(userID, "testuser", "test@example.com", "Test User")
    err := repo.Save(ctx, user)
    require.NoError(t, err)

    // Test Exists - should return true
    exists, err := repo.Exists(ctx, userID)
    require.NoError(t, err)
    assert.True(t, exists)

    // Test Exists - should return false for non-existent user
    exists, err = repo.Exists(ctx, uuid.NewUUID())
    require.NoError(t, err)
    assert.False(t, exists)
}

func TestMongoUserRepository_ExistsByUsername(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    coll := client.Database("test_db").Collection("users")
    repo := mongodb.NewMongoUserRepository(coll)

    // Create user
    userID := uuid.NewUUID()
    user := userdomain.NewUser(userID, "testuser", "test@example.com", "Test User")
    err := repo.Save(ctx, user)
    require.NoError(t, err)

    // Test ExistsByUsername - should return true
    exists, err := repo.ExistsByUsername(ctx, "testuser")
    require.NoError(t, err)
    assert.True(t, exists)

    // Test ExistsByUsername - should return false
    exists, err = repo.ExistsByUsername(ctx, "nonexistent")
    require.NoError(t, err)
    assert.False(t, exists)
}

func TestMongoUserRepository_FindByID_And_DocumentToUser(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    coll := client.Database("test_db").Collection("users")
    repo := mongodb.NewMongoUserRepository(coll)

    // Create and save user
    userID := uuid.NewUUID()
    originalUser := userdomain.NewUser(userID, "testuser", "test@example.com", "Test User")
    err := repo.Save(ctx, originalUser)
    require.NoError(t, err)

    // Load user
    loaded, err := repo.FindByID(ctx, userID)
    require.NoError(t, err)

    // Verify all fields
    assert.Equal(t, userID, loaded.ID())
    assert.Equal(t, "testuser", loaded.Username())
    assert.Equal(t, "test@example.com", loaded.Email())
    assert.Equal(t, "Test User", loaded.DisplayName())
    assert.False(t, loaded.IsSystemAdmin())
}
```

## Checklist

### Phase 1: Domain layer

- [ ] Добавить функцию `Restore` в `internal/domain/user/user.go`
- [ ] Убедиться, что `Restore` принимает все необходимые поля
- [ ] Добавить комментарий о назначении функции

### Phase 2: Repository layer

- [ ] Реализовать `documentToUser` с использованием `Restore`
- [ ] Добавить метод `Exists`
- [ ] Добавить метод `ExistsByUsername`
- [ ] Добавить метод `ExistsByEmail`
- [ ] Добавить необходимые импорты (`fmt`, `time`)

### Phase 3: Тестирование

- [ ] Обновить существующие тесты для работы с `documentToUser`
- [ ] Добавить тесты для `Exists`
- [ ] Добавить тесты для `ExistsByUsername`
- [ ] Добавить тесты для `ExistsByEmail`
- [ ] Проверить, что все тесты проходят

### Phase 4: Проверка интерфейсов

- [ ] Убедиться, что `MongoUserRepository` реализует `userapp.Repository`
- [ ] Убедиться, что `MongoUserRepository` реализует `shared.UserRepository`

## Следующие шаги

После завершения этой задачи:

1. **Task 04** — завершение WorkspaceRepository
2. **Task 05** — проверка и доработка MessageRepository

## Референсы

- Существующий код: `internal/infrastructure/repository/mongodb/user_repository.go`
- Domain model: `internal/domain/user/user.go`
- Интерфейсы: `internal/application/user/repository.go`
- Shared интерфейс: `internal/application/shared/user_repository.go`

# Task 04: Завершение Workspace Repository

## Цель

Завершить реализацию MongoDB репозитория для Workspace, добавив недостающий метод `documentToWorkspace`, функцию `Restore` в domain layer и методы для работы с членами workspace.

## Контекст

Репозиторий `MongoWorkspaceRepository` уже создан в `internal/infrastructure/repository/mongodb/workspace_repository.go`, но имеет несколько проблем:

1. Метод `documentToWorkspace` возвращает ошибку:
```go
func (r *MongoWorkspaceRepository) documentToWorkspace(doc *workspaceDocument) (*workspacedomain.Workspace, error) {
    return nil, errors.New("documentToWorkspace requires domain setter methods - not yet implemented")
}
```

2. Метод `FindInviteByToken` не полностью реализован
3. Отсутствуют методы для работы с членами workspace

## Зависимости

### Уже реализовано

- `internal/infrastructure/repository/mongodb/workspace_repository.go` — частичная реализация
- `internal/domain/workspace/workspace.go` — domain model Workspace
- `internal/application/workspace/repository.go` — интерфейсы репозитория

### Требуется изменить

1. `internal/domain/workspace/workspace.go` — добавить функцию `Restore`
2. `internal/infrastructure/repository/mongodb/workspace_repository.go` — реализовать `documentToWorkspace` и методы для членов

## Детальное описание

### 1. Анализ текущей структуры Workspace

Сначала нужно проанализировать `internal/domain/workspace/workspace.go` для понимания структуры:

```go
type Workspace struct {
    id              uuid.UUID
    name            string
    keycloakGroupID *string
    createdBy       uuid.UUID
    createdAt       time.Time
    updatedAt       time.Time
    members         []Member
    invites         []Invite
}
```

### 2. Добавить Restore функцию в domain

Изменить `internal/domain/workspace/workspace.go`:

```go
// Restore восстанавливает Workspace из сохраненных полей (для persistence layer)
// Эта функция должна использоваться ТОЛЬКО репозиторием для восстановления
// сущности из хранилища. Для создания нового workspace используйте NewWorkspace.
func Restore(
    id uuid.UUID,
    name string,
    keycloakGroupID *string,
    createdBy uuid.UUID,
    createdAt time.Time,
    updatedAt time.Time,
    members []Member,
    invites []Invite,
) *Workspace {
    return &Workspace{
        id:              id,
        name:            name,
        keycloakGroupID: keycloakGroupID,
        createdBy:       createdBy,
        createdAt:       createdAt,
        updatedAt:       updatedAt,
        members:         members,
        invites:         invites,
    }
}

// RestoreMember восстанавливает Member из сохраненных полей
func RestoreMember(
    userID uuid.UUID,
    role Role,
    joinedAt time.Time,
) Member {
    return Member{
        userID:   userID,
        role:     role,
        joinedAt: joinedAt,
    }
}

// RestoreInvite восстанавливает Invite из сохраненных полей
func RestoreInvite(
    token string,
    email string,
    expiresAt time.Time,
    createdAt time.Time,
) Invite {
    return Invite{
        token:     token,
        email:     email,
        expiresAt: expiresAt,
        createdAt: createdAt,
    }
}
```

### 3. Обновить структуры документов

Расширить структуры в `workspace_repository.go`:

```go
// workspaceDocument представляет структуру документа в MongoDB
type workspaceDocument struct {
    WorkspaceID     string           `bson:"workspace_id"`
    Name            string           `bson:"name"`
    KeycloakGroupID *string          `bson:"keycloak_group_id,omitempty"`
    CreatedBy       string           `bson:"created_by"`
    CreatedAt       time.Time        `bson:"created_at"`
    UpdatedAt       time.Time        `bson:"updated_at"`
    Members         []memberDocument `bson:"members"`
    Invites         []inviteDocument `bson:"invites"`
}

// memberDocument представляет члена workspace в документе
type memberDocument struct {
    UserID   string    `bson:"user_id"`
    Role     string    `bson:"role"`
    JoinedAt time.Time `bson:"joined_at"`
}

// inviteDocument представляет приглашение в документе
type inviteDocument struct {
    Token     string    `bson:"token"`
    Email     string    `bson:"email"`
    ExpiresAt time.Time `bson:"expires_at"`
    CreatedAt time.Time `bson:"created_at"`
}
```

### 4. Реализовать documentToWorkspace

```go
// documentToWorkspace преобразует Document в Workspace
func (r *MongoWorkspaceRepository) documentToWorkspace(doc *workspaceDocument) (*workspacedomain.Workspace, error) {
    if doc == nil {
        return nil, errs.ErrInvalidInput
    }

    // Парсим UUID
    workspaceID := uuid.UUID(doc.WorkspaceID)
    if workspaceID.IsZero() {
        return nil, fmt.Errorf("invalid workspace_id: %s", doc.WorkspaceID)
    }

    createdBy := uuid.UUID(doc.CreatedBy)
    if createdBy.IsZero() {
        return nil, fmt.Errorf("invalid created_by: %s", doc.CreatedBy)
    }

    // Восстанавливаем members
    members := make([]workspacedomain.Member, 0, len(doc.Members))
    for _, m := range doc.Members {
        userID := uuid.UUID(m.UserID)
        if userID.IsZero() {
            continue // Пропускаем невалидных членов
        }

        member := workspacedomain.RestoreMember(
            userID,
            workspacedomain.Role(m.Role),
            m.JoinedAt,
        )
        members = append(members, member)
    }

    // Восстанавливаем invites
    invites := make([]workspacedomain.Invite, 0, len(doc.Invites))
    for _, inv := range doc.Invites {
        invite := workspacedomain.RestoreInvite(
            inv.Token,
            inv.Email,
            inv.ExpiresAt,
            inv.CreatedAt,
        )
        invites = append(invites, invite)
    }

    // Восстанавливаем Workspace
    workspace := workspacedomain.Restore(
        workspaceID,
        doc.Name,
        doc.KeycloakGroupID,
        createdBy,
        doc.CreatedAt,
        doc.UpdatedAt,
        members,
        invites,
    )

    return workspace, nil
}
```

### 5. Обновить workspaceToDocument

```go
// workspaceToDocument преобразует Workspace в Document
func (r *MongoWorkspaceRepository) workspaceToDocument(ws *workspacedomain.Workspace) workspaceDocument {
    // Преобразуем members
    members := make([]memberDocument, 0, len(ws.Members()))
    for _, m := range ws.Members() {
        members = append(members, memberDocument{
            UserID:   m.UserID().String(),
            Role:     string(m.Role()),
            JoinedAt: m.JoinedAt(),
        })
    }

    // Преобразуем invites
    invites := make([]inviteDocument, 0, len(ws.Invites()))
    for _, inv := range ws.Invites() {
        invites = append(invites, inviteDocument{
            Token:     inv.Token(),
            Email:     inv.Email(),
            ExpiresAt: inv.ExpiresAt(),
            CreatedAt: inv.CreatedAt(),
        })
    }

    doc := workspaceDocument{
        WorkspaceID:     ws.ID().String(),
        Name:            ws.Name(),
        KeycloakGroupID: ws.KeycloakGroupID(),
        CreatedBy:       ws.CreatedBy().String(),
        CreatedAt:       ws.CreatedAt(),
        UpdatedAt:       ws.UpdatedAt(),
        Members:         members,
        Invites:         invites,
    }

    return doc
}
```

### 6. Реализовать FindInviteByToken

```go
// FindInviteByToken находит приглашение по токену
func (r *MongoWorkspaceRepository) FindInviteByToken(
    ctx context.Context,
    token string,
) (*workspacedomain.Invite, error) {
    if token == "" {
        return nil, errs.ErrInvalidInput
    }

    // Ищем workspace с этим приглашением
    filter := bson.M{"invites.token": token}

    var doc workspaceDocument
    err := r.collection.FindOne(ctx, filter).Decode(&doc)
    if err != nil {
        return nil, HandleMongoError(err, "invite")
    }

    // Находим конкретное приглашение в массиве
    for _, inv := range doc.Invites {
        if inv.Token == token {
            invite := workspacedomain.RestoreInvite(
                inv.Token,
                inv.Email,
                inv.ExpiresAt,
                inv.CreatedAt,
            )
            return &invite, nil
        }
    }

    return nil, errs.ErrNotFound
}
```

### 7. Добавить методы для работы с членами

```go
// GetMember возвращает члена workspace по userID
func (r *MongoWorkspaceRepository) GetMember(
    ctx context.Context,
    workspaceID uuid.UUID,
    userID uuid.UUID,
) (*workspacedomain.Member, error) {
    if workspaceID.IsZero() || userID.IsZero() {
        return nil, errs.ErrInvalidInput
    }

    filter := bson.M{
        "workspace_id":    workspaceID.String(),
        "members.user_id": userID.String(),
    }

    var doc workspaceDocument
    err := r.collection.FindOne(ctx, filter).Decode(&doc)
    if err != nil {
        return nil, HandleMongoError(err, "workspace")
    }

    for _, m := range doc.Members {
        if m.UserID == userID.String() {
            member := workspacedomain.RestoreMember(
                uuid.UUID(m.UserID),
                workspacedomain.Role(m.Role),
                m.JoinedAt,
            )
            return &member, nil
        }
    }

    return nil, errs.ErrNotFound
}

// IsMember проверяет, является ли пользователь членом workspace
func (r *MongoWorkspaceRepository) IsMember(
    ctx context.Context,
    workspaceID uuid.UUID,
    userID uuid.UUID,
) (bool, error) {
    if workspaceID.IsZero() || userID.IsZero() {
        return false, errs.ErrInvalidInput
    }

    filter := bson.M{
        "workspace_id":    workspaceID.String(),
        "members.user_id": userID.String(),
    }

    count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
    if err != nil {
        return false, HandleMongoError(err, "workspace")
    }

    return count > 0, nil
}

// ListWorkspacesByUser возвращает workspaces, в которых пользователь является членом
func (r *MongoWorkspaceRepository) ListWorkspacesByUser(
    ctx context.Context,
    userID uuid.UUID,
    offset, limit int,
) ([]*workspacedomain.Workspace, error) {
    if userID.IsZero() {
        return nil, errs.ErrInvalidInput
    }

    filter := bson.M{"members.user_id": userID.String()}
    opts := options.Find().
        SetSort(bson.D{{Key: "name", Value: 1}}).
        SetLimit(int64(limit)).
        SetSkip(int64(offset))

    cursor, err := r.collection.Find(ctx, filter, opts)
    if err != nil {
        return nil, HandleMongoError(err, "workspaces")
    }
    defer cursor.Close(ctx)

    var workspaces []*workspacedomain.Workspace
    for cursor.Next(ctx) {
        var doc workspaceDocument
        if decodeErr := cursor.Decode(&doc); decodeErr != nil {
            continue
        }

        ws, docErr := r.documentToWorkspace(&doc)
        if docErr != nil {
            continue
        }

        workspaces = append(workspaces, ws)
    }

    if workspaces == nil {
        workspaces = make([]*workspacedomain.Workspace, 0)
    }

    return workspaces, nil
}

// CountWorkspacesByUser возвращает количество workspaces пользователя
func (r *MongoWorkspaceRepository) CountWorkspacesByUser(ctx context.Context, userID uuid.UUID) (int, error) {
    if userID.IsZero() {
        return 0, errs.ErrInvalidInput
    }

    filter := bson.M{"members.user_id": userID.String()}
    count, err := r.collection.CountDocuments(ctx, filter)
    if err != nil {
        return 0, HandleMongoError(err, "workspaces")
    }

    return int(count), nil
}
```

### 8. Обновить интерфейс репозитория

Добавить в `internal/application/workspace/repository.go`:

```go
// QueryRepository предоставляет методы для чтения данных Workspace
type QueryRepository interface {
    // FindByID находит рабочее пространство по ID
    FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)

    // FindByKeycloakGroup находит рабочее пространство по ID группы Keycloak
    FindByKeycloakGroup(ctx context.Context, keycloakGroupID string) (*workspace.Workspace, error)

    // List возвращает список рабочих пространств с пагинацией
    List(ctx context.Context, offset, limit int) ([]*workspace.Workspace, error)

    // Count возвращает общее количество рабочих пространств
    Count(ctx context.Context) (int, error)

    // FindInviteByToken находит приглашение по токену
    FindInviteByToken(ctx context.Context, token string) (*workspace.Invite, error)

    // GetMember возвращает члена workspace
    GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)

    // IsMember проверяет, является ли пользователь членом workspace
    IsMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error)

    // ListWorkspacesByUser возвращает workspaces пользователя
    ListWorkspacesByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*workspace.Workspace, error)

    // CountWorkspacesByUser возвращает количество workspaces пользователя
    CountWorkspacesByUser(ctx context.Context, userID uuid.UUID) (int, error)
}
```

## Тестирование

### Тесты для documentToWorkspace

```go
func TestMongoWorkspaceRepository_FindByID_And_DocumentToWorkspace(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    db := client.Database("test_db")
    coll := db.Collection("workspaces")
    membersColl := db.Collection("workspace_members")
    repo := mongodb.NewMongoWorkspaceRepository(coll, membersColl)

    // Create workspace with members
    wsID := uuid.NewUUID()
    createdBy := uuid.NewUUID()
    memberID := uuid.NewUUID()

    ws := workspacedomain.NewWorkspace(wsID, "Test Workspace", createdBy)
    ws.AddMember(memberID, workspacedomain.RoleMember)

    err := repo.Save(ctx, ws)
    require.NoError(t, err)

    // Load workspace
    loaded, err := repo.FindByID(ctx, wsID)
    require.NoError(t, err)

    // Verify fields
    assert.Equal(t, wsID, loaded.ID())
    assert.Equal(t, "Test Workspace", loaded.Name())
    assert.Equal(t, createdBy, loaded.CreatedBy())
    assert.Len(t, loaded.Members(), 2) // createdBy + memberID
}

func TestMongoWorkspaceRepository_IsMember(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    db := client.Database("test_db")
    coll := db.Collection("workspaces")
    membersColl := db.Collection("workspace_members")
    repo := mongodb.NewMongoWorkspaceRepository(coll, membersColl)

    // Create workspace
    wsID := uuid.NewUUID()
    createdBy := uuid.NewUUID()
    memberID := uuid.NewUUID()
    nonMemberID := uuid.NewUUID()

    ws := workspacedomain.NewWorkspace(wsID, "Test Workspace", createdBy)
    ws.AddMember(memberID, workspacedomain.RoleMember)
    _ = repo.Save(ctx, ws)

    // Test IsMember - should return true for member
    isMember, err := repo.IsMember(ctx, wsID, memberID)
    require.NoError(t, err)
    assert.True(t, isMember)

    // Test IsMember - should return false for non-member
    isMember, err = repo.IsMember(ctx, wsID, nonMemberID)
    require.NoError(t, err)
    assert.False(t, isMember)
}

func TestMongoWorkspaceRepository_FindInviteByToken(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    db := client.Database("test_db")
    coll := db.Collection("workspaces")
    membersColl := db.Collection("workspace_members")
    repo := mongodb.NewMongoWorkspaceRepository(coll, membersColl)

    // Create workspace with invite
    wsID := uuid.NewUUID()
    createdBy := uuid.NewUUID()
    ws := workspacedomain.NewWorkspace(wsID, "Test Workspace", createdBy)

    token := "test-invite-token-123"
    ws.CreateInvite(token, "invite@example.com", time.Now().Add(24*time.Hour))
    _ = repo.Save(ctx, ws)

    // Find invite by token
    invite, err := repo.FindInviteByToken(ctx, token)
    require.NoError(t, err)

    assert.Equal(t, token, invite.Token())
    assert.Equal(t, "invite@example.com", invite.Email())
}

func TestMongoWorkspaceRepository_ListWorkspacesByUser(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    db := client.Database("test_db")
    coll := db.Collection("workspaces")
    membersColl := db.Collection("workspace_members")
    repo := mongodb.NewMongoWorkspaceRepository(coll, membersColl)

    // Create workspaces
    userID := uuid.NewUUID()

    for i := 0; i < 3; i++ {
        wsID := uuid.NewUUID()
        ws := workspacedomain.NewWorkspace(wsID, fmt.Sprintf("Workspace %d", i), userID)
        _ = repo.Save(ctx, ws)
    }

    // List workspaces by user
    workspaces, err := repo.ListWorkspacesByUser(ctx, userID, 0, 10)
    require.NoError(t, err)

    assert.Len(t, workspaces, 3)
}
```

## Индексы для Workspace

Добавить в `07-mongodb-indexes.md`:

```javascript
// Workspaces Collection
db.workspaces.createIndex({ "workspace_id": 1 }, { unique: true })
db.workspaces.createIndex({ "keycloak_group_id": 1 }, { unique: true, sparse: true })
db.workspaces.createIndex({ "name": 1 })
db.workspaces.createIndex({ "created_by": 1 })
db.workspaces.createIndex({ "members.user_id": 1 })
db.workspaces.createIndex({ "invites.token": 1 })
db.workspaces.createIndex({ "invites.email": 1 })
```

## Checklist

### Phase 1: Domain layer

- [x] Добавить функцию `Restore` в `internal/domain/workspace/workspace.go` (уже есть как `Reconstruct`)
- [x] Добавить функцию `RestoreMember` (создан `internal/domain/workspace/member.go` с `ReconstructMember`)
- [x] Добавить функцию `RestoreInvite` (уже есть как `ReconstructInvite`)
- [x] Убедиться, что Workspace имеет getter для invites (`Invites()`)

### Phase 2: Document structures

- [x] Обновить `memberDocument` структуру (добавлена в workspace_repository.go)
- [x] Обновить `inviteDocument` структуру (уже была реализована)
- [x] Обновить `workspaceDocument` с members и invites (члены хранятся в отдельной коллекции)

### Phase 3: Core methods

- [x] Реализовать `documentToWorkspace` с использованием `Restore` (использует `Reconstruct`)
- [x] Обновить `workspaceToDocument` для сохранения members и invites
- [x] Реализовать `FindInviteByToken` полностью

### Phase 4: Member methods

- [x] Добавить метод `GetMember`
- [x] Добавить метод `IsMember`
- [x] Добавить метод `ListWorkspacesByUser`
- [x] Добавить метод `CountWorkspacesByUser`
- [x] Добавить метод `AddMember`
- [x] Добавить метод `RemoveMember`
- [x] Добавить метод `ListMembers`
- [x] Добавить метод `CountMembers`

### Phase 5: Interface update

- [x] Обновить `QueryRepository` интерфейс
- [x] Обновить `CommandRepository` интерфейс (добавлены AddMember, RemoveMember)
- [x] Убедиться, что `MongoWorkspaceRepository` реализует все методы

### Phase 6: Тестирование

- [x] Добавить тест `FindByID_And_DocumentToWorkspace` (как `DocToWorkspace`)
- [x] Добавить тест `IsMember`
- [x] Добавить тест `FindInviteByToken`
- [x] Добавить тест `ListWorkspacesByUser` (как `ListByUser`)
- [x] Добавить тесты для Member domain model
- [x] Проверить, что все тесты проходят

## Следующие шаги

После завершения этой задачи:

1. **Task 05** — проверка и доработка MessageRepository
2. **Task 06** — проверка и доработка NotificationRepository

## Референсы

- Существующий код: `internal/infrastructure/repository/mongodb/workspace_repository.go`
- Domain model: `internal/domain/workspace/workspace.go`
- Интерфейсы: `internal/application/workspace/repository.go`
- Аналогичная реализация: `user_repository.go`

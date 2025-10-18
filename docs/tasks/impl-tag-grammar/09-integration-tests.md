# Task 09: Integration Tests

**Статус:** Pending
**Приоритет:** High
**Зависимости:** All previous tasks (01-08)
**Оценка:** 3-4 дня

## Описание

Создать end-to-end интеграционные тесты для системы тегов, тестирующие полный flow от сообщения с тегами до изменения состояния в БД и генерации bot response.

## Цели

1. Протестировать полный pipeline обработки тегов
2. Проверить взаимодействие всех компонентов
3. Протестировать реальные сценарии пользователей
4. Проверить event sourcing и проекции
5. Убедиться в корректности bot responses

## Технические требования

### Структура интеграционных тестов

```
tests/integration/
├── tag/
│   ├── setup_test.go           # Test setup и helpers
│   ├── create_entity_test.go   # Создание сущностей
│   ├── manage_entity_test.go   # Управление сущностями
│   ├── error_handling_test.go  # Обработка ошибок
│   ├── scenarios_test.go       # Полные сценарии
│   └── fixtures/
│       ├── users.json
│       └── chats.json
```

### Setup для интеграционных тестов

```go
// tests/integration/tag/setup_test.go
package tag_test

import (
    "testing"
    "github.com/google/uuid"
    "github.com/stretchr/testify/suite"
)

type TagIntegrationTestSuite struct {
    suite.Suite

    // Repos
    chatRepo    repository.ChatRepository
    userRepo    repository.UserRepository
    messageRepo repository.MessageRepository
    eventStore  event.EventStore

    // Services
    tagHandler  *tag.TagHandler

    // Test data
    testUser1   *domain.User
    testUser2   *domain.User
    testChat    *domain.Chat

    // MongoDB
    mongoClient *mongo.Client
    database    *mongo.Database
}

func (suite *TagIntegrationTestSuite) SetupTest() {
    // Подключение к test MongoDB
    suite.mongoClient = setupTestMongo()
    suite.database = suite.mongoClient.Database("tag_integration_test")

    // Создание репозиториев
    suite.chatRepo = repository.NewMongoDBChatRepository(suite.database)
    suite.userRepo = repository.NewMongoDBUserRepository(suite.database)
    suite.messageRepo = repository.NewMongoDBMessageRepository(suite.database)
    suite.eventStore = event.NewMongoDBEventStore(suite.database)

    // Создание tag handler
    parser := tag.NewTagParser()
    validator := tag.NewTagValidationSystem(suite.userRepo)
    processor := tag.NewTagProcessor()
    executor := tag.NewCommandExecutor(suite.chatRepo, suite.userRepo, suite.eventStore)
    suite.tagHandler = tag.NewTagHandler(parser, validator, processor, executor, suite.messageRepo)

    // Создание тестовых пользователей
    suite.testUser1 = &domain.User{
        ID:       uuid.New(),
        Username: "alex",
        Email:    "alex@example.com",
    }
    suite.userRepo.Save(suite.testUser1)

    suite.testUser2 = &domain.User{
        ID:       uuid.New(),
        Username: "bob",
        Email:    "bob@example.com",
    }
    suite.userRepo.Save(suite.testUser2)

    // Создание тестового чата
    suite.testChat = &domain.Chat{
        ID:   uuid.New(),
        Type: domain.TypeDiscussion,
    }
    suite.chatRepo.Save(suite.testChat)
}

func (suite *TagIntegrationTestSuite) TearDownTest() {
    // Очистка БД
    suite.database.Drop(context.Background())
    suite.mongoClient.Disconnect(context.Background())
}

func TestTagIntegrationTestSuite(t *testing.T) {
    suite.Run(t, new(TagIntegrationTestSuite))
}
```

### Тесты создания сущностей

```go
// tests/integration/tag/create_entity_test.go

func (suite *TagIntegrationTestSuite) TestCreateTask() {
    content := "#task Реализовать OAuth авторизацию"

    err := suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        content,
    )

    suite.NoError(err)

    // Проверяем сообщение сохранено
    messages := suite.messageRepo.FindByChatID(suite.testChat.ID)
    suite.Len(messages, 2) // 1 user message + 1 bot response
    suite.Equal(content, messages[0].Content)

    // Проверяем чат превращён в Task
    chat := suite.chatRepo.FindByID(suite.testChat.ID)
    suite.Equal(domain.TypeTask, chat.Type)
    suite.Equal("Реализовать OAuth авторизацию", chat.Title)
    suite.Equal("To Do", chat.Status) // default status

    // Проверяем события
    events := suite.eventStore.GetEventsByChatID(suite.testChat.ID)
    suite.NotEmpty(events)

    // Последнее событие - ChatConvertedToTaskEvent
    lastEvent := events[len(events)-1]
    suite.IsType(&domain.ChatConvertedToTaskEvent{}, lastEvent)

    // Проверяем bot response
    botMessage := messages[1]
    suite.Contains(botMessage.Content, "✅ Task created")
    suite.Contains(botMessage.Content, "Реализовать OAuth авторизацию")
}

func (suite *TagIntegrationTestSuite) TestCreateTaskWithAttributes() {
    content := "#task Реализовать OAuth #priority High #assignee @alex"

    err := suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        content,
    )

    suite.NoError(err)

    // Проверяем чат
    chat := suite.chatRepo.FindByID(suite.testChat.ID)
    suite.Equal(domain.TypeTask, chat.Type)
    suite.Equal("Реализовать OAuth", chat.Title)
    suite.Equal("High", chat.Priority)
    suite.Equal(&suite.testUser1.ID, chat.AssigneeID)

    // Проверяем bot response
    messages := suite.messageRepo.FindByChatID(suite.testChat.ID)
    botMessage := messages[1]
    suite.Contains(botMessage.Content, "✅ Task created")
    suite.Contains(botMessage.Content, "✅ Priority changed to High")
    suite.Contains(botMessage.Content, "✅ Assigned to: @alex")
}

func (suite *TagIntegrationTestSuite) TestCreateBugWithSeverity() {
    content := "#bug Ошибка при логине #severity Critical"

    err := suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        content,
    )

    suite.NoError(err)

    chat := suite.chatRepo.FindByID(suite.testChat.ID)
    suite.Equal(domain.TypeBug, chat.Type)
    suite.Equal("Ошибка при логине", chat.Title)
    suite.Equal("Critical", chat.Severity)
    suite.Equal("New", chat.Status) // default Bug status
}
```

### Тесты управления сущностями

```go
// tests/integration/tag/manage_entity_test.go

func (suite *TagIntegrationTestSuite) TestChangeStatus() {
    // Подготовка: создаём Task
    suite.testChat.Type = domain.TypeTask
    suite.testChat.Title = "Test Task"
    suite.testChat.Status = "To Do"
    suite.chatRepo.Save(suite.testChat)

    content := "#status In Progress"

    err := suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        content,
    )

    suite.NoError(err)

    // Проверяем статус изменился
    chat := suite.chatRepo.FindByID(suite.testChat.ID)
    suite.Equal("In Progress", chat.Status)

    // Проверяем событие
    events := suite.eventStore.GetEventsByChatID(suite.testChat.ID)
    lastEvent := events[len(events)-1].(*domain.StatusChangedEvent)
    suite.Equal("To Do", lastEvent.OldStatus)
    suite.Equal("In Progress", lastEvent.NewStatus)

    // Проверяем bot response
    messages := suite.messageRepo.FindByChatID(suite.testChat.ID)
    botMessage := messages[len(messages)-1]
    suite.Contains(botMessage.Content, "✅ Status changed to In Progress")
}

func (suite *TagIntegrationTestSuite) TestAssignUser() {
    suite.testChat.Type = domain.TypeTask
    suite.testChat.Title = "Test Task"
    suite.chatRepo.Save(suite.testChat)

    content := "#assignee @bob"

    err := suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        content,
    )

    suite.NoError(err)

    chat := suite.chatRepo.FindByID(suite.testChat.ID)
    suite.NotNil(chat.AssigneeID)
    suite.Equal(suite.testUser2.ID, *chat.AssigneeID)

    messages := suite.messageRepo.FindByChatID(suite.testChat.ID)
    botMessage := messages[len(messages)-1]
    suite.Contains(botMessage.Content, "✅ Assigned to: @bob")
}

func (suite *TagIntegrationTestSuite) TestRemoveAssignee() {
    suite.testChat.Type = domain.TypeTask
    suite.testChat.AssigneeID = &suite.testUser2.ID
    suite.chatRepo.Save(suite.testChat)

    content := "#assignee @none"

    err := suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        content,
    )

    suite.NoError(err)

    chat := suite.chatRepo.FindByID(suite.testChat.ID)
    suite.Nil(chat.AssigneeID)

    messages := suite.messageRepo.FindByChatID(suite.testChat.ID)
    botMessage := messages[len(messages)-1]
    suite.Contains(botMessage.Content, "✅ Assignee removed")
}

func (suite *TagIntegrationTestSuite) TestSetDueDate() {
    suite.testChat.Type = domain.TypeTask
    suite.chatRepo.Save(suite.testChat)

    content := "#due 2025-10-20"

    err := suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        content,
    )

    suite.NoError(err)

    chat := suite.chatRepo.FindByID(suite.testChat.ID)
    suite.NotNil(chat.DueDate)
    suite.Equal(2025, chat.DueDate.Year())
    suite.Equal(time.October, chat.DueDate.Month())
    suite.Equal(20, chat.DueDate.Day())
}
```

### Тесты обработки ошибок

```go
// tests/integration/tag/error_handling_test.go

func (suite *TagIntegrationTestSuite) TestPartialApplication() {
    suite.testChat.Type = domain.TypeTask
    suite.chatRepo.Save(suite.testChat)

    content := "#status Done #assignee @nonexistent #priority High"

    err := suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        content,
    )

    suite.NoError(err)

    // Проверяем частичное применение
    chat := suite.chatRepo.FindByID(suite.testChat.ID)
    suite.Equal("Done", chat.Status)          // ✅ применено
    suite.Nil(chat.AssigneeID)                // ❌ не применено
    suite.Equal("High", chat.Priority)        // ✅ применено

    // Проверяем bot response
    messages := suite.messageRepo.FindByChatID(suite.testChat.ID)
    botMessage := messages[len(messages)-1]
    suite.Contains(botMessage.Content, "✅ Status changed to Done")
    suite.Contains(botMessage.Content, "✅ Priority changed to High")
    suite.Contains(botMessage.Content, "❌ User @nonexistent not found")

    // Проверяем, что сообщение пользователя сохранено
    userMessage := messages[len(messages)-2]
    suite.Equal(content, userMessage.Content)
}

func (suite *TagIntegrationTestSuite) TestAllTagsInvalid() {
    suite.testChat.Type = domain.TypeTask
    suite.chatRepo.Save(suite.testChat)

    content := "#status InvalidStatus #assignee @nobody"

    err := suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        content,
    )

    suite.NoError(err)

    // Проверяем, что сообщение сохранено
    messages := suite.messageRepo.FindByChatID(suite.testChat.ID)
    suite.Len(messages, 2) // user message + bot response

    userMessage := messages[0]
    suite.Equal(content, userMessage.Content)

    // Проверяем bot response содержит обе ошибки
    botMessage := messages[1]
    suite.Contains(botMessage.Content, "❌ Invalid status")
    suite.Contains(botMessage.Content, "❌ User @nobody not found")
}

func (suite *TagIntegrationTestSuite) TestInvalidStatusForEntityType() {
    suite.testChat.Type = domain.TypeTask
    suite.chatRepo.Save(suite.testChat)

    content := "#status Fixed" // Bug status в Task

    err := suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        content,
    )

    suite.NoError(err)

    messages := suite.messageRepo.FindByChatID(suite.testChat.ID)
    botMessage := messages[len(messages)-1]
    suite.Contains(botMessage.Content, "❌ Invalid status 'Fixed' for Task")
    suite.Contains(botMessage.Content, "Available: To Do, In Progress, Done")
}
```

### Тесты полных сценариев

```go
// tests/integration/tag/scenarios_test.go

func (suite *TagIntegrationTestSuite) TestFullTaskLifecycle() {
    // Сценарий: Создание задачи → изменение статуса → назначение → завершение

    // 1. Создание Task
    err := suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        "#task Реализовать OAuth",
    )
    suite.NoError(err)

    chat := suite.chatRepo.FindByID(suite.testChat.ID)
    suite.Equal(domain.TypeTask, chat.Type)
    suite.Equal("To Do", chat.Status)

    // 2. Назначение и начало работы
    err = suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        "Начинаю работу\n#status In Progress\n#assignee @alex",
    )
    suite.NoError(err)

    chat = suite.chatRepo.FindByID(suite.testChat.ID)
    suite.Equal("In Progress", chat.Status)
    suite.Equal(&suite.testUser1.ID, chat.AssigneeID)

    // 3. Установка приоритета и дедлайна
    err = suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        "#priority High\n#due 2025-10-25",
    )
    suite.NoError(err)

    chat = suite.chatRepo.FindByID(suite.testChat.ID)
    suite.Equal("High", chat.Priority)
    suite.NotNil(chat.DueDate)

    // 4. Завершение
    err = suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        "Закончил работу\n#status Done",
    )
    suite.NoError(err)

    chat = suite.chatRepo.FindByID(suite.testChat.ID)
    suite.Equal("Done", chat.Status)

    // Проверяем историю событий
    events := suite.eventStore.GetEventsByChatID(suite.testChat.ID)
    suite.NotEmpty(events)

    // Должны быть события: ConvertToTask, ChangeStatus (x2), AssignUser, SetPriority, SetDueDate
    eventTypes := make(map[string]int)
    for _, evt := range events {
        eventTypes[fmt.Sprintf("%T", evt)]++
    }

    suite.Greater(eventTypes["*domain.ChatConvertedToTaskEvent"], 0)
    suite.Greater(eventTypes["*domain.StatusChangedEvent"], 0)
    suite.Greater(eventTypes["*domain.UserAssignedEvent"], 0)
}

func (suite *TagIntegrationTestSuite) TestConvertDiscussionToTask() {
    // Сценарий из спецификации: превращение обычного чата в задачу

    // 1. Создаём несколько сообщений в обычном чате
    suite.messageRepo.Save(domain.Message{
        ID:       uuid.New(),
        ChatID:   suite.testChat.ID,
        AuthorID: suite.testUser1.ID,
        Content:  "У нас проблема с производительностью",
    })

    suite.messageRepo.Save(domain.Message{
        ID:       uuid.New(),
        ChatID:   suite.testChat.ID,
        AuthorID: suite.testUser2.ID,
        Content:  "Да, нужно разобраться",
    })

    // 2. Превращаем в задачу
    err := suite.tagHandler.HandleMessageWithTags(
        suite.testChat.ID,
        suite.testUser1.ID,
        "Давайте сделаем это задачей\n#task Разобраться с проблемой производительности",
    )
    suite.NoError(err)

    // 3. Проверяем, что чат превратился в Task
    chat := suite.chatRepo.FindByID(suite.testChat.ID)
    suite.Equal(domain.TypeTask, chat.Type)
    suite.Equal("Разобраться с проблемой производительности", chat.Title)

    // 4. Проверяем, что все предыдущие сообщения остались
    messages := suite.messageRepo.FindByChatID(suite.testChat.ID)
    suite.GreaterOrEqual(len(messages), 4) // 2 старых + 1 новое + 1 bot response

    // Проверяем bot response
    botMessage := messages[len(messages)-1]
    suite.Contains(botMessage.Content, "✅ Task created")
    suite.Contains(botMessage.Content, "Разобраться с проблемой производительности")
}
```

## Acceptance Criteria

- [ ] Созданы интеграционные тесты для всех основных сценариев
- [ ] Протестировано создание всех типов сущностей (Task, Bug, Epic)
- [ ] Протестированы все операции управления (status, assignee, priority, due, title, severity)
- [ ] Протестирована обработка ошибок и частичное применение
- [ ] Протестированы полные жизненные циклы задач
- [ ] Протестировано превращение обычного чата в typed
- [ ] Все тесты проходят
- [ ] Используется test MongoDB (изолированные тесты)
- [ ] Тесты независимы друг от друга (Setup/TearDown)

## Запуск интеграционных тестов

```bash
# Запуск всех интеграционных тестов
go test -tags=integration ./tests/integration/tag/...

# С verbose output
go test -tags=integration -v ./tests/integration/tag/...

# Конкретный тест
go test -tags=integration -v -run TestTagIntegrationTestSuite/TestCreateTask ./tests/integration/tag/...
```

## Требования к окружению

- MongoDB должен быть запущен (можно через Docker)
- Используется отдельная тестовая БД (автоматически создаётся и удаляется)

```bash
# Запуск MongoDB для тестов
docker-compose up -d mongodb
```

## Ссылки

- Сценарии из спецификации: `docs/03-tag-grammar.md` (строки 755-863)
- Integration tests примеры: `docs/03-tag-grammar.md` (строки 987-1011)

# UseCase Layer Completion Plan

**Дата создания:** 2025-10-22
**Текущий прогресс:** 82%
**Оставшееся время:** 4-6 часов

## Статус проекта

### ✅ Завершено (5/8 фаз)

1. **Phase 1: Architecture** - 100% ✅
2. **Phase 3: Message UseCases** - 100% ✅
3. **Phase 4: User UseCases** - 100% ✅
4. **Phase 5: Workspace UseCases** - 100% ✅
5. **Phase 8: Tag Integration** - 100% ✅

### 🟡 В процессе (3/8 фаз)

6. **Phase 2: Chat UseCases** - 80%
7. **Phase 6: Notification UseCases** - 78%
8. **Phase 7: Integration & Testing** - 75%

## Критические задачи (Must Complete)

### 🔴 Task 09: Chat UseCases Testing

**Приоритет:** КРИТИЧЕСКИЙ
**Оценка:** 3-4 часа
**Блокирует:** Переход к infrastructure layer

**Проблема:**
- Chat UseCases имеют 0% test coverage
- 12 Command UseCases реализованы, но не протестированы
- Это самый большой риск проекта

**Решение:**
- Создать 60+ unit тестов для всех Chat UseCases
- Достичь coverage >85%
- Использовать существующую тестовую инфраструктуру

**Детали:** См. [09-chat-tests.md](./09-chat-tests.md)

**Декомпозиция:**
1. Подготовка (15 мин)
   - Создать test_setup.go
   - Настроить mocks
2. Реализация (3 часа)
   - CreateChatUseCase: 8 тестов
   - AddParticipantUseCase: 7 тестов
   - RemoveParticipantUseCase: 5 тестов
   - ConvertToTaskUseCase: 5 тестов
   - ConvertToBugUseCase: 4 теста
   - ConvertToEpicUseCase: 3 теста
   - ChangeStatusUseCase: 6 тестов
   - AssignUserUseCase: 4 теста
   - SetPriorityUseCase: 6 тестов
   - SetDueDateUseCase: 5 тестов
   - RenameChatUseCase: 4 теста
   - SetSeverityUseCase: 6 тестов
3. Проверка (15 мин)

**Результат:**
- ✅ Coverage увеличится с 0% до >85%
- ✅ Application layer coverage: 64.7% → ~75%
- ✅ Confidence в корректности Chat UseCases

---

### 🟡 Task 10: Chat Query UseCases

**Приоритет:** ВЫСОКИЙ
**Оценка:** 1-2 часа
**Блокирует:** Полную функциональность Chat агрегата

**Проблема:**
- Query UseCases не реализованы
- Невозможно получить данные чата для UI
- Отсутствует пагинация для списков

**Решение:**
- Реализовать 3 Query UseCases:
  1. GetChatUseCase - получение по ID
  2. ListChatsUseCase - список с фильтрацией и пагинацией
  3. ListParticipantsUseCase - список участников
- Полное тестовое покрытие (15 тестов)

**Детали:** См. [10-chat-queries.md](./10-chat-queries.md)

**Декомпозиция:**
1. Подготовка (10 мин)
   - Создать queries.go
   - Обновить results.go
2. GetChatUseCase (30 мин)
   - Реализация + 4 теста
3. ListChatsUseCase (40 мин)
   - Реализация + 6 тестов
   - Фильтрация по типу
   - Pagination
4. ListParticipantsUseCase (30 мин)
   - Реализация + 5 тестов
5. Проверка (10 мин)

**Результат:**
- ✅ Chat агрегат полностью функционален
- ✅ Phase 2 полностью завершена
- ✅ Можно использовать в HTTP handlers

---

## Дополнительные задачи (Should Complete)

### 📝 Task 11: Documentation Update

**Приоритет:** СРЕДНИЙ
**Оценка:** 1 час

**Что обновить:**

1. **README.md**
   - Обновить статус реализации
   - Добавить примеры использования UseCases
   - Обновить архитектурную диаграмму

2. **ARCHITECTURE_DIAGRAM.md**
   - Добавить UseCase layer
   - Показать зависимости между слоями
   - Обновить flow диаграммы

3. **Создать API_EXAMPLES.md**
   ```go
   // Пример использования Chat UseCases

   // 1. Создание чата
   createCmd := chat.CreateChatCommand{
       WorkspaceID: workspaceID,
       Type:        chat.TypeTask,
       Title:       "Implement feature X",
       IsPublic:    true,
       CreatedBy:   userID,
   }
   result, err := createChatUseCase.Execute(ctx, createCmd)

   // 2. Получение чата
   query := chat.GetChatQuery{ChatID: chatID}
   chatResult, err := getChatUseCase.Execute(ctx, query)

   // 3. Список чатов
   listQuery := chat.ListChatsQuery{
       WorkspaceID: workspaceID,
       Type:        &taskType,
       Limit:       20,
   }
   chats, err := listChatsUseCase.Execute(ctx, listQuery)
   ```

**Результат:**
- ✅ Документация актуальна
- ✅ Новые разработчики понимают архитектуру
- ✅ Примеры использования готовы

---

### 🧪 Task 12: E2E Tests

**Приоритет:** СРЕДНИЙ
**Оценка:** 2-3 часа

**Что протестировать:**

1. **Complete Task Workflow**
   ```
   CreateChat (Discussion)
   → SendMessage with tag "#createTask Test Task"
   → Tag Processor parses tag
   → CommandExecutor converts to Task
   → Verify chat type changed
   → Verify events published
   ```

2. **Messaging Workflow**
   ```
   CreateChat
   → AddParticipant
   → SendMessage
   → AddReaction
   → EditMessage
   → Verify all events
   ```

3. **Workspace Invitation Workflow**
   ```
   CreateWorkspace
   → CreateInvite
   → AcceptInvite
   → Verify Keycloak integration
   → Verify user added to workspace
   ```

**Файлы:**
- `tests/e2e/task_workflow_test.go`
- `tests/e2e/messaging_workflow_test.go`
- `tests/e2e/workspace_workflow_test.go`

**Результат:**
- ✅ Confidence в интеграции между доменами
- ✅ Проверка end-to-end сценариев
- ✅ Regression protection

---

## Опциональные задачи (Nice to Have)

### 🔔 Task 13: Notification Event Handlers

**Приоритет:** НИЗКИЙ (можно в infrastructure phase)
**Оценка:** 2 часа

**Реализовать:**
1. NotificationEventHandler
2. HandleChatCreated
3. HandleUserAssigned
4. HandleStatusChanged
5. HandleMessageSent
6. Event bus subscription setup

**Примечание:** Требует Event Bus implementation, поэтому логичнее сделать в infrastructure phase.

---

### 📊 Task 14: CI/CD Setup

**Приоритет:** НИЗКИЙ
**Оценка:** 1-2 часа

**Создать:**

1. **GitHub Actions Workflow**
   ```yaml
   # .github/workflows/test.yml
   name: Tests
   on: [push, pull_request]
   jobs:
     test:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v2
         - uses: actions/setup-go@v2
           with:
             go-version: 1.25
         - run: go test -v -coverprofile=coverage.out ./...
         - run: go tool cover -func=coverage.out
         - name: Check coverage
           run: |
             coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
             echo "Coverage: $coverage"
             # Fail if coverage < 80%
   ```

2. **Pre-commit hooks**
   - golangci-lint
   - go test
   - go fmt check

**Результат:**
- ✅ Automated testing на каждый commit
- ✅ Coverage tracking
- ✅ Code quality enforcement

---

## План выполнения

### Фаза 1: Критические задачи (4-6 часов)

**Приоритет:** Завершить до перехода к infrastructure layer

```
День 1 (4 часа):
├─ Task 09: Chat Tests (3.5 часа)
│  ├─ Подготовка: 15 мин
│  ├─ Реализация тестов: 3 часа
│  │  ├─ CreateChat, AddParticipant, RemoveParticipant: 1 час
│  │  ├─ Convert* UseCases: 50 мин
│  │  ├─ ChangeStatus, AssignUser, SetPriority: 1 час
│  │  └─ SetDueDate, Rename, SetSeverity: 50 мин
│  └─ Проверка: 15 мин
└─ Task 10: Query UseCases (2 часа)
   ├─ Подготовка: 10 мин
   ├─ GetChatUseCase: 30 мин
   ├─ ListChatsUseCase: 40 мин
   ├─ ListParticipantsUseCase: 30 мин
   └─ Проверка: 10 мин

Результат:
✅ Chat UseCases 100% complete
✅ Coverage >85% для всех UseCases
✅ Phase 2 завершена
```

### Фаза 2: Документация (1 час)

**Опционально, но рекомендуется**

```
День 2 (1 час):
└─ Task 11: Documentation
   ├─ README.md update: 20 мин
   ├─ ARCHITECTURE_DIAGRAM.md: 20 мин
   └─ API_EXAMPLES.md: 20 мин
```

### Фаза 3: E2E Tests (2-3 часа)

**Опционально**

```
День 3 (2-3 часа):
└─ Task 12: E2E Tests
   ├─ Task workflow: 1 час
   ├─ Messaging workflow: 45 мин
   └─ Workspace workflow: 45 мин
```

### Фаза 4: Infrastructure (отложено)

```
Следующая итерация:
├─ Task 13: Notification Event Handlers
│  (в рамках Event Bus implementation)
└─ Task 14: CI/CD Setup
```

---

## Метрики успеха

### Минимальные требования (Must have)

- [x] Architecture Phase - 100%
- [ ] **Chat UseCases - 100%** ⚠️ КРИТИЧНО
  - [ ] All tests written
  - [ ] Query UseCases implemented
  - [ ] Coverage >85%
- [x] Message UseCases - 100%
- [x] User UseCases - 100%
- [x] Workspace UseCases - 100%
- [x] Notification UseCases - 100% (UseCases only)
- [x] Tag Integration - 100%

### Coverage goals

```
Current:
  Domain Layer:          ~90%+ ✅
  Application Layer:     64.7%
    - chat:              0.0%  ❌ БЛОКЕР!
    - message:          78.7%  ✅
    - user:             85.7%  ✅
    - workspace:        85.9%  ✅
    - notification:     84.8%  ✅
    - task:             84.9%  ✅
    - shared:           72.8%  🟡

Target:
  Domain Layer:          >90%  ✅
  Application Layer:     >85%
    - chat:             >85%   ⚠️ After Task 09+10
```

### Дополнительные метрики

- [ ] E2E tests: >70% coverage
- [ ] CI/CD: automated testing
- [ ] Documentation: up-to-date

---

## Следующая фаза: Infrastructure Layer

После завершения UseCase layer:

### Готовые компоненты для infrastructure:

1. **EventStore implementation** (MongoDB)
   - SaveEvents
   - LoadEvents
   - Snapshots support

2. **Repository implementations**
   - ChatRepository (EventStore based)
   - MessageRepository (MongoDB)
   - UserRepository (MongoDB)
   - WorkspaceRepository (MongoDB)
   - NotificationRepository (MongoDB)

3. **Event Bus** (Redis pub/sub)
   - Publish
   - Subscribe
   - Event handlers registration

4. **HTTP Handlers** (Echo framework)
   - REST API endpoints
   - Request/Response DTOs
   - Middleware (auth, logging, errors)

5. **WebSocket Handlers**
   - Real-time messaging
   - Presence tracking
   - Event broadcasting

6. **Keycloak Integration**
   - OAuth2 client
   - User sync
   - Group management

### Оценка infrastructure phase:

**Время:** 2-3 недели
**Сложность:** Высокая
**Зависимости:** Docker, MongoDB, Redis, Keycloak

---

## Команды для быстрого старта

### Запуск критических задач

```bash
# Task 09: Chat Tests
cd internal/application/chat
# Создать test файлы согласно плану в 09-chat-tests.md
go test -v ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Task 10: Query UseCases
# Реализовать согласно плану в 10-chat-queries.md
go test -v -run Query ./...
```

### Проверка прогресса

```bash
# Coverage по всему application layer
go test -coverprofile=/tmp/coverage.out ./internal/application/...
go tool cover -func=/tmp/coverage.out | tail -1

# Запуск всех тестов
go test ./... -v

# Линтер
golangci-lint run ./internal/application/...
```

### Обновление PROGRESS_TRACKER

```bash
# После каждой завершённой задачи
vim docs/tasks/04-impl-usecase/PROGRESS_TRACKER.md
# Обновить прогресс Phase 2
```

---

## Решение

**Рекомендуемый порядок выполнения:**

1. ✅ **СЕГОДНЯ: Task 09 - Chat Tests** (3.5 часа)
   - Максимальный приоритет
   - Блокирует всё остальное

2. ✅ **СЕГОДНЯ/ЗАВТРА: Task 10 - Query UseCases** (2 часа)
   - Высокий приоритет
   - Завершает Phase 2

3. 🟡 **ОПЦИОНАЛЬНО: Task 11 - Documentation** (1 час)
   - Полезно, но не критично

4. 🟡 **ОПЦИОНАЛЬНО: Task 12 - E2E Tests** (2-3 часа)
   - Можно отложить

5. ⏸️ **ОТЛОЖИТЬ: Task 13 - Event Handlers**
   - Сделать в infrastructure phase

6. ⏸️ **ОТЛОЖИТЬ: Task 14 - CI/CD**
   - Можно сделать параллельно с infrastructure

**Итого критический путь: 5.5-6 часов**

После завершения Task 09 и Task 10:
- ✅ UseCase layer готов на 100%
- ✅ Можно переходить к infrastructure layer
- ✅ Все бизнес-логика протестирована и безопасна

---

## Контакты и ресурсы

**Документация:**
- [PROGRESS_TRACKER.md](./PROGRESS_TRACKER.md) - текущий статус
- [09-chat-tests.md](./09-chat-tests.md) - детальный план тестов
- [10-chat-queries.md](./10-chat-queries.md) - детальный план Query UseCases
- [README.md](./README.md) - общая информация

**Референсы:**
- Message UseCases: `internal/application/message/`
- Test examples: `internal/application/message/*_test.go`
- Mocks: `tests/mocks/`
- Fixtures: `tests/fixtures/`

**Полезные команды:**
```bash
# Текущий coverage
make test-coverage

# Только Chat tests
go test ./internal/application/chat/... -v

# Benchmark
go test -bench=. ./internal/application/chat/...
```

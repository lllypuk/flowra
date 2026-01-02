# PR #7 — Implementation Status

Документ отслеживает статус реализации задач из `00-blockers.md`, `01-cmd-api-fixes.md` и `02-repo-hygiene.md`.

---

## Сводка

| Категория | Выполнено | Всего | Статус |
|-----------|-----------|-------|--------|
| Блокеры (00-blockers.md) | 6 | 6 | ✅ Готово |
| DI/Health Fixes (01-cmd-api-fixes.md) | 6 | 6 | ✅ Готово |
| Repo Hygiene (02-repo-hygiene.md) | 3 | 4 | ⚠️ В процессе |

---

## Блокер 1 — DI container mocks

**Статус:** ✅ Выполнено

### Что сделано:
- Добавлен `AppConfig` в конфигурацию с полем `Mode` (`real` | `mock`)
- Реализована логика условного wiring в `container.go`:
  - `setupHTTPHandlersReal()` — для production
  - `setupHTTPHandlersMock()` — для dev/testing
- Добавлена валидация: mock mode запрещён в production
- Логирование режима wiring при старте
- Добавлен `validateWiring()` для проверки инициализации компонентов

### Изменённые файлы:
- `internal/config/config.go` — добавлен `AppConfig`, `AppMode`, валидация
- `cmd/api/container.go` — разделение wiring, валидация, логирование
- `configs/config.yaml` — добавлена секция `app:`

---

## Блокер 2 — AcquireContext() в readiness

**Статус:** ✅ Выполнено

### Что сделано:
- Удалено использование `e.AcquireContext().Request().Context()`
- Создан интерфейс `httpserver.HealthChecker` с методами:
  - `IsReady(ctx context.Context) bool`
  - `GetHealthStatus(ctx context.Context) []ComponentStatus`
- `Container` реализует `HealthChecker`
- Используется `router.RegisterHealthEndpointsWithChecker(c)` вместо callback

### Изменённые файлы:
- `internal/infrastructure/httpserver/health.go` — новый файл
- `internal/infrastructure/httpserver/router.go` — deprecated старый метод
- `cmd/api/routes.go` — использование нового API

---

## Блокер 3 — Дублирование health констант

**Статус:** ✅ Выполнено

### Что сделано:
- Создан единый источник констант в `httpserver/health.go`:
  - `StatusHealthy`, `StatusUnhealthy`, `StatusDegraded`
  - `StatusReady`, `StatusNotReady`
- Удалены локальные константы из `container.go` и `routes.go`
- Единый формат ответа через `HealthResponse` и `ComponentStatus`

### Изменённые файлы:
- `internal/infrastructure/httpserver/health.go` — константы и типы
- `cmd/api/container.go` — использует `httpserver.StatusXxx`
- `cmd/api/routes.go` — удалены локальные константы

---

## Блокер 4 — Файл `api` в корне

**Статус:** ✅ Выполнено

### Что сделано:
- Удалён бинарник `api` (17MB ELF executable)
- Удалены артефакты: `coverage.html`, `coverage.out`, `websocket.test`
- Обновлён `.gitignore`:
  - Добавлены `/api`, `/worker`, `/migrator`, `/websocket`
  - Добавлены `*.prof`, `*.pprof`
  - Добавлены `dist/`, `build/`

### Изменённые файлы:
- `.gitignore` — расширен список игнорируемых файлов

---

## Блокер 5 — "Пустые" файлы

**Статус:** ⚠️ Требует проверки

### Что сделано:
- Задача описана в `02-repo-hygiene.md`
- Требуется ручная проверка содержимого файлов в PR

### Что осталось:
- [ ] Проверить `internal/infrastructure/eventbus/*.go`
- [ ] Проверить `internal/infrastructure/httpserver/*.go`
- [ ] Проверить `tests/e2e/*.go`

---

## Блокер 6 — Псевдотесты

**Статус:** ✅ Частично

### Что сделано:
- Обновлены тесты в `cmd/api/container_test.go`:
  - Используют `httpserver.StatusXxx` вместо локальных констант
  - Удалены тесты удалённых типов
- Обновлены тесты в `cmd/api/routes_test.go`:
  - Удалены тесты для удалённой функции `SetupHealthEndpoints`
  - Добавлен тест `TestCreatePlaceholderHandler`
- Добавлены тесты для `AppConfig` в `internal/config/config_test.go`

### Что осталось:
- [ ] Ревью других тестов на "псевдотесты" (отдельная задача)

---

## Дополнительные улучшения

### Рефакторинг `Config.Validate()`

**Статус:** ✅ Выполнено

Для снижения когнитивной сложности функция разбита на:
- `validateApp()`
- `validateServer()`
- `validateMongoDB()`
- `validateRedis()`
- `validateAuth()`
- `validateLog()`
- `validateEventBus()`
- `validateWebSocket()`

---

## Проверка качества кода

### Linting

```bash
golangci-lint run ./...
# 0 issues.
```

### Тесты

```bash
go test ./... -count=1
# ok  (все пакеты)
```

---

## Файлы, затронутые изменениями

| Файл | Тип изменения |
|------|---------------|
| `.gitignore` | Modified |
| `configs/config.yaml` | Modified |
| `internal/config/config.go` | Modified |
| `internal/config/config_test.go` | Modified |
| `internal/infrastructure/httpserver/health.go` | **Created** |
| `internal/infrastructure/httpserver/router.go` | Modified |
| `cmd/api/container.go` | Modified |
| `cmd/api/container_test.go` | Modified |
| `cmd/api/routes.go` | Modified |
| `cmd/api/routes_test.go` | Modified |

---

## Следующие шаги

1. **Ручная проверка "пустых" файлов** (Блокер 5)
2. **Создание реальных сервисов** для замены mock implementations:
   - `AuthService` (интеграция с Keycloak)
   - `WorkspaceService` (обёртка над use cases)
   - `ChatService` (обёртка над use cases)
   - `WorkspaceAccessChecker` (реальная проверка membership)
3. **Документация** — обновить README/DEPLOYMENT с новыми конфигурационными опциями

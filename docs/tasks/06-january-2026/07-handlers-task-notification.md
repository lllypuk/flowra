# 07: Task & Notification Handlers

**Приоритет:** 🟡 High  
**Статус:** ⏳ Не начато  
**Дни:** 15-17 января  
**Зависит от:** [04-middleware.md](04-middleware.md)

---

## Описание

Реализовать HTTP handlers для управления задачами и уведомлениями. Task handler обеспечивает полный CRUD для задач с Event Sourcing, Notification handler — управление уведомлениями пользователя.

---

## Файлы для создания

```
internal/handler/http/
├── task_handler.go         (~400 LOC)
├── task_handler_test.go    (~300 LOC)
├── notification_handler.go (~250 LOC)
├── notification_handler_test.go (~200 LOC)
├── user_handler.go         (~200 LOC)
└── user_handler_test.go    (~150 LOC)
```

---

## Task Handler

### Endpoints

| Method | Path | Описание |
|--------|------|----------|
| `POST` | `/api/v1/workspaces/:workspace_id/tasks` | Создать задачу |
| `GET` | `/api/v1/workspaces/:workspace_id/tasks` | Список задач workspace |
| `GET` | `/api/v1/tasks/:id` | Получить задачу |
| `PUT` | `/api/v1/tasks/:id/status` | Изменить статус |
| `PUT` | `/api/v1/tasks/:id/assign` | Назначить исполнителя |
| `PUT` | `/api/v1/tasks/:id/priority` | Изменить приоритет |
| `PUT` | `/api/v1/tasks/:id/due-date` | Установить срок |
| `DELETE` | `/api/v1/tasks/:id` | Удалить задачу |

### Структура handler

```go
type TaskHandler struct {
    createTaskUC       *task.CreateTaskUseCase
    updateTaskUC       *task.UpdateTaskUseCase
    changeStatusUC     *task.ChangeStatusUseCase
    assignTaskUC       *task.AssignTaskUseCase
    setDueDateUC       *task.SetDueDateUseCase
    taskRepo           TaskRepository
}

func NewTaskHandler(
    createTaskUC *task.CreateTaskUseCase,
    updateTaskUC *task.UpdateTaskUseCase,
    changeStatusUC *task.ChangeStatusUseCase,
    assignTaskUC *task.AssignTaskUseCase,
    setDueDateUC *task.SetDueDateUseCase,
    taskRepo TaskRepository,
) *TaskHandler
```

### Request/Response DTOs

```go
// CreateTaskRequest
type CreateTaskRequest struct {
    Title       string    `json:"title" validate:"required,min=1,max=200"`
    Description string    `json:"description" validate:"max=5000"`
    Priority    string    `json:"priority" validate:"oneof=low medium high urgent"`
    AssigneeID  *string   `json:"assignee_id" validate:"omitempty,uuid"`
    DueDate     *string   `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
    ChatID      *string   `json:"chat_id" validate:"omitempty,uuid"`
}

// TaskResponse
type TaskResponse struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Status      string    `json:"status"`
    Priority    string    `json:"priority"`
    AssigneeID  *string   `json:"assignee_id,omitempty"`
    ReporterID  string    `json:"reporter_id"`
    DueDate     *string   `json:"due_date,omitempty"`
    CreatedAt   string    `json:"created_at"`
    UpdatedAt   string    `json:"updated_at"`
}

// ChangeStatusRequest
type ChangeStatusRequest struct {
    Status string `json:"status" validate:"required,oneof=open in_progress review done cancelled"`
}

// AssignTaskRequest
type AssignTaskRequest struct {
    AssigneeID string `json:"assignee_id" validate:"required,uuid"`
}
```

### Фильтрация и пагинация

```go
// ListTasksQuery
type ListTasksQuery struct {
    Status     string `query:"status"`
    AssigneeID string `query:"assignee_id"`
    Priority   string `query:"priority"`
    ChatID     string `query:"chat_id"`
    Page       int    `query:"page" validate:"min=1"`
    PerPage    int    `query:"per_page" validate:"min=1,max=100"`
    SortBy     string `query:"sort_by" validate:"oneof=created_at updated_at due_date priority"`
    SortOrder  string `query:"sort_order" validate:"oneof=asc desc"`
}
```

---

## Notification Handler

### Endpoints

| Method | Path | Описание |
|--------|------|----------|
| `GET` | `/api/v1/notifications` | Список уведомлений |
| `GET` | `/api/v1/notifications/unread/count` | Количество непрочитанных |
| `PUT` | `/api/v1/notifications/:id/read` | Пометить как прочитанное |
| `PUT` | `/api/v1/notifications/mark-all-read` | Прочитать все |
| `DELETE` | `/api/v1/notifications/:id` | Удалить уведомление |

### Структура handler

```go
type NotificationHandler struct {
    listNotificationsUC *notification.ListNotificationsUseCase
    markReadUC          *notification.MarkReadUseCase
    deleteNotifUC       *notification.DeleteNotificationUseCase
    notifRepo           NotificationRepository
}
```

### Response DTOs

```go
// NotificationResponse
type NotificationResponse struct {
    ID        string `json:"id"`
    Type      string `json:"type"`
    Title     string `json:"title"`
    Body      string `json:"body"`
    IsRead    bool   `json:"is_read"`
    Link      string `json:"link,omitempty"`
    CreatedAt string `json:"created_at"`
}

// UnreadCountResponse
type UnreadCountResponse struct {
    Count int `json:"count"`
}
```

---

## User Handler

### Endpoints

| Method | Path | Описание |
|--------|------|----------|
| `GET` | `/api/v1/users/me` | Текущий пользователь |
| `PUT` | `/api/v1/users/me` | Обновить профиль |
| `GET` | `/api/v1/users/:id` | Получить пользователя |

### Структура handler

```go
type UserHandler struct {
    userRepo UserRepository
}

// UserResponse
type UserResponse struct {
    ID          string `json:"id"`
    Email       string `json:"email"`
    DisplayName string `json:"display_name"`
    AvatarURL   string `json:"avatar_url,omitempty"`
    Status      string `json:"status"`
    CreatedAt   string `json:"created_at"`
}

// UpdateProfileRequest
type UpdateProfileRequest struct {
    DisplayName string `json:"display_name" validate:"required,min=1,max=100"`
    AvatarURL   string `json:"avatar_url" validate:"omitempty,url"`
}
```

---

## Критерии приёмки

### Task Handler
- [ ] POST `/workspaces/:workspace_id/tasks` создаёт задачу
- [ ] GET `/workspaces/:workspace_id/tasks` возвращает список с фильтрацией
- [ ] GET `/tasks/:id` возвращает задачу
- [ ] PUT `/tasks/:id/status` меняет статус
- [ ] PUT `/tasks/:id/assign` назначает исполнителя
- [ ] PUT `/tasks/:id/priority` меняет приоритет
- [ ] PUT `/tasks/:id/due-date` устанавливает срок
- [ ] DELETE `/tasks/:id` удаляет задачу
- [ ] Валидация входных данных работает
- [ ] Пагинация и сортировка работают
- [ ] Authorization проверяется

### Notification Handler
- [ ] GET `/notifications` возвращает список
- [ ] GET `/notifications/unread/count` возвращает количество
- [ ] PUT `/notifications/:id/read` помечает как прочитанное
- [ ] PUT `/notifications/mark-all-read` помечает все
- [ ] DELETE `/notifications/:id` удаляет уведомление

### User Handler
- [ ] GET `/users/me` возвращает текущего пользователя
- [ ] PUT `/users/me` обновляет профиль
- [ ] GET `/users/:id` возвращает пользователя

### Общее
- [ ] Unit tests для всех handlers
- [ ] Integration tests с mock use cases
- [ ] Error handling корректен
- [ ] HTTP статусы соответствуют REST conventions

---

## Зависимости

### Входящие
- [04-middleware.md](04-middleware.md) — middleware инфраструктура
- [06-handlers-chat-message.md](06-handlers-chat-message.md) — паттерны из предыдущих handlers

### Исходящие
- [08-websocket.md](08-websocket.md) — real-time updates для задач
- [09-entry-points.md](09-entry-points.md) — регистрация handlers
- [10-e2e-tests.md](10-e2e-tests.md) — E2E тесты для задач

---

## Заметки

- Task handler работает с Event Sourcing через use cases
- Notification handler должен возвращать только уведомления текущего пользователя
- Notifications могут группироваться по типу или источнику (опционально)
- При изменении статуса задачи публикуется событие через Event Bus
- User handler `/users/me` использует UserID из auth context

---

*Создано: 2026-01-01*
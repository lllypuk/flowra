# Step 3: Wire `TaskActionHandler` in Container

## Status: Complete

## Goal

Initialize `TaskActionHandler` in `cmd/api/container.go` by reusing the already-wired
`ActionService` and the existing `boardTaskServiceAdapter` (which already satisfies the
`TaskActionTaskService` interface via its `GetTask` method).

## File to Change

**`cmd/api/container.go`** — only the `setupHTTPHandlers` method

## Context: What Already Exists

At the end of `setupHTTPHandlers` (around line 906–908):

```go
// === 14. Action Service ===
c.ActionService = service.NewActionService(c.SendMessageUC, c.UserRepo)
c.ChatActionHandler = httphandler.NewChatActionHandler(c.ActionService)
c.Logger.Debug("action service and chat action handler initialized")

// Initialize TaskHandler with full service
c.TaskHandler = httphandler.NewTaskHandler(c.createFullTaskService())
c.Logger.Debug("task handler initialized (real)")
```

Both `c.ActionService` and the task service (via `createFullTaskService` which wraps
`boardTaskServiceAdapter`) are already available at this point.

## Change

Add `TaskActionHandler` initialization **immediately after** `TaskHandler` initialization:

### Current Code (lines ~900–908)

```go
// === 14. Action Service ===
c.ActionService = service.NewActionService(c.SendMessageUC, c.UserRepo)
c.ChatActionHandler = httphandler.NewChatActionHandler(c.ActionService)
c.Logger.Debug("action service and chat action handler initialized")
c.Logger.Debug("action service initialized")

// Initialize TaskHandler with full service
c.TaskHandler = httphandler.NewTaskHandler(c.createFullTaskService())
c.Logger.Debug("task handler initialized (real)")
```

### Updated Code

```go
// === 14. Action Service ===
c.ActionService = service.NewActionService(c.SendMessageUC, c.UserRepo)
c.ChatActionHandler = httphandler.NewChatActionHandler(c.ActionService)
c.Logger.Debug("action service and chat action handler initialized")

// Initialize TaskHandler with full service
c.TaskHandler = httphandler.NewTaskHandler(c.createFullTaskService())
c.Logger.Debug("task handler initialized (real)")

// Initialize TaskActionHandler — routes sidebar changes through chat message system
c.TaskActionHandler = httphandler.NewTaskActionHandler(
    c.createTaskActionService(),
    c.ActionService,
)
c.Logger.Debug("task action handler initialized")
```

## New Helper Method

Add a new private method `createTaskActionService` to `container.go`. This returns the
`boardTaskServiceAdapter` — which already implements `GetTask` — wrapped as the
`TaskActionTaskService` interface:

```go
// createTaskActionService creates a service implementing TaskActionTaskService.
// Reuses boardTaskServiceAdapter since it already provides GetTask by task_id.
func (c *Container) createTaskActionService() httphandler.TaskActionTaskService {
	return &boardTaskServiceAdapter{
		collection: c.MongoDB.Database(c.MongoDBName).Collection("tasks_read_model"),
	}
}
```

### Why Reuse `boardTaskServiceAdapter`?

`boardTaskServiceAdapter` (already defined in `container.go`) has a `GetTask` method:

```go
func (a *boardTaskServiceAdapter) GetTask(ctx context.Context, taskID uuid.UUID) (*taskapp.ReadModel, error) {
    filter := map[string]any{"task_id": taskID.String()}
    var result taskReadModelDoc
    if err := a.collection.FindOne(ctx, filter).Decode(&result); err != nil {
        return nil, taskapp.ErrTaskNotFound
    }
    return result.toReadModel(), nil
}
```

The `TaskActionTaskService` interface (declared in `task_action_handler.go`) requires:

```go
type TaskActionTaskService interface {
    GetTask(ctx context.Context, taskID uuid.UUID) (*taskapp.ReadModel, error)
}
```

`boardTaskServiceAdapter` satisfies this interface already — no new code needed in the
adapter itself.

## Verification

After this change, the wiring sequence in `setupHTTPHandlers` is:

1. `ActionService` created (needs `SendMessageUC`, `UserRepo`) ✅
2. `ChatActionHandler` created (needs `ActionService`) ✅
3. `TaskHandler` created (needs full task service) ✅
4. `TaskActionHandler` created (needs task service + `ActionService`) ✅ ← **new**

All dependencies are available before `TaskActionHandler` is initialized.

## No Changes to `validateHandlers`

`validateHandlers` checks only `AuthHandler`, `WorkspaceHandler`, `ChatHandler`, and
`WSHandler` for real mode. `TaskActionHandler` does not need to be added to validation —
the nil guard in `registerTaskRoutes` handles the case where it is not initialized.

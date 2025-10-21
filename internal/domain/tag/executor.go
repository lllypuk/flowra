package tag

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/user"
	domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

const (
	noneUsername = "@none"
)

// CommandExecutor выполняет tag команды на Chat aggregate
type CommandExecutor struct {
	chatRepo ChatRepository
	userRepo UserRepository
	eventBus event.Bus
}

// NewCommandExecutor создает новый CommandExecutor
func NewCommandExecutor(
	chatRepo ChatRepository,
	userRepo UserRepository,
	eventBus event.Bus,
) *CommandExecutor {
	return &CommandExecutor{
		chatRepo: chatRepo,
		userRepo: userRepo,
		eventBus: eventBus,
	}
}

// Execute выполняет команду
func (e *CommandExecutor) Execute(ctx context.Context, cmd Command, actorID uuid.UUID) error {
	switch c := cmd.(type) {
	case CreateTaskCommand:
		return e.executeCreateTask(ctx, c, actorID)
	case CreateBugCommand:
		return e.executeCreateBug(ctx, c, actorID)
	case CreateEpicCommand:
		return e.executeCreateEpic(ctx, c, actorID)
	case ChangeStatusCommand:
		return e.executeChangeStatus(ctx, c, actorID)
	case AssignUserCommand:
		return e.executeAssignUser(ctx, c, actorID)
	case ChangePriorityCommand:
		return e.executeChangePriority(ctx, c, actorID)
	case SetDueDateCommand:
		return e.executeSetDueDate(ctx, c, actorID)
	case ChangeTitleCommand:
		return e.executeChangeTitle(ctx, c, actorID)
	case SetSeverityCommand:
		return e.executeSetSeverity(ctx, c, actorID)
	default:
		return fmt.Errorf("unknown command type: %T", cmd)
	}
}

// executeCreateTask выполняет команду создания Task
func (e *CommandExecutor) executeCreateTask(ctx context.Context, cmd CreateTaskCommand, actorID uuid.UUID) error {
	// Конвертация UUID
	chatID := domainUUID.FromGoogleUUID(cmd.ChatID)
	userID := domainUUID.FromGoogleUUID(actorID)

	// Загрузка чата
	c, err := e.chatRepo.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}

	// Выполнение команды на aggregate
	if err = c.ConvertToTask(cmd.Title, userID); err != nil {
		return fmt.Errorf("failed to convert to task: %w", err)
	}

	// Публикация событий и сохранение
	return e.publishAndSave(ctx, c)
}

// executeCreateBug выполняет команду создания Bug
func (e *CommandExecutor) executeCreateBug(ctx context.Context, cmd CreateBugCommand, actorID uuid.UUID) error {
	chatID := domainUUID.FromGoogleUUID(cmd.ChatID)
	userID := domainUUID.FromGoogleUUID(actorID)

	c, err := e.chatRepo.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}

	if err = c.ConvertToBug(cmd.Title, userID); err != nil {
		return fmt.Errorf("failed to convert to bug: %w", err)
	}

	return e.publishAndSave(ctx, c)
}

// executeCreateEpic выполняет команду создания Epic
func (e *CommandExecutor) executeCreateEpic(ctx context.Context, cmd CreateEpicCommand, actorID uuid.UUID) error {
	chatID := domainUUID.FromGoogleUUID(cmd.ChatID)
	userID := domainUUID.FromGoogleUUID(actorID)

	c, err := e.chatRepo.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}

	if err = c.ConvertToEpic(cmd.Title, userID); err != nil {
		return fmt.Errorf("failed to convert to epic: %w", err)
	}

	return e.publishAndSave(ctx, c)
}

// executeChangeStatus выполняет команду изменения статуса
func (e *CommandExecutor) executeChangeStatus(ctx context.Context, cmd ChangeStatusCommand, actorID uuid.UUID) error {
	chatID := domainUUID.FromGoogleUUID(cmd.ChatID)
	userID := domainUUID.FromGoogleUUID(actorID)

	c, err := e.chatRepo.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}

	if err = c.ChangeStatus(cmd.Status, userID); err != nil {
		return fmt.Errorf("failed to change status: %w", err)
	}

	return e.publishAndSave(ctx, c)
}

// executeAssignUser выполняет команду назначения пользователя
func (e *CommandExecutor) executeAssignUser(ctx context.Context, cmd AssignUserCommand, actorID uuid.UUID) error {
	chatID := domainUUID.FromGoogleUUID(cmd.ChatID)
	userID := domainUUID.FromGoogleUUID(actorID)

	c, err := e.chatRepo.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}

	// Резолвинг пользователя
	var assigneeID *domainUUID.UUID
	if cmd.Username != "" && cmd.Username != noneUsername {
		username := strings.TrimPrefix(cmd.Username, "@")
		var u *user.User
		u, err = e.userRepo.FindByUsername(ctx, username)
		if err != nil {
			return fmt.Errorf("user %s not found: %w", cmd.Username, err)
		}
		uid := u.ID()
		assigneeID = &uid
	}

	// Выполнение команды
	if err = c.AssignUser(assigneeID, userID); err != nil {
		return fmt.Errorf("failed to assign user: %w", err)
	}

	return e.publishAndSave(ctx, c)
}

// executeChangePriority выполняет команду изменения приоритета
func (e *CommandExecutor) executeChangePriority(
	ctx context.Context,
	cmd ChangePriorityCommand,
	actorID uuid.UUID,
) error {
	chatID := domainUUID.FromGoogleUUID(cmd.ChatID)
	userID := domainUUID.FromGoogleUUID(actorID)

	c, err := e.chatRepo.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}

	if err = c.SetPriority(cmd.Priority, userID); err != nil {
		return fmt.Errorf("failed to set priority: %w", err)
	}

	return e.publishAndSave(ctx, c)
}

// executeSetDueDate выполняет команду установки дедлайна
func (e *CommandExecutor) executeSetDueDate(ctx context.Context, cmd SetDueDateCommand, actorID uuid.UUID) error {
	chatID := domainUUID.FromGoogleUUID(cmd.ChatID)
	userID := domainUUID.FromGoogleUUID(actorID)

	c, err := e.chatRepo.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}

	if err = c.SetDueDate(cmd.DueDate, userID); err != nil {
		return fmt.Errorf("failed to set due date: %w", err)
	}

	return e.publishAndSave(ctx, c)
}

// executeChangeTitle выполняет команду изменения названия
func (e *CommandExecutor) executeChangeTitle(ctx context.Context, cmd ChangeTitleCommand, actorID uuid.UUID) error {
	chatID := domainUUID.FromGoogleUUID(cmd.ChatID)
	userID := domainUUID.FromGoogleUUID(actorID)

	c, err := e.chatRepo.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}

	if err = c.Rename(cmd.Title, userID); err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}

	return e.publishAndSave(ctx, c)
}

// executeSetSeverity выполняет команду установки severity
func (e *CommandExecutor) executeSetSeverity(ctx context.Context, cmd SetSeverityCommand, actorID uuid.UUID) error {
	chatID := domainUUID.FromGoogleUUID(cmd.ChatID)
	userID := domainUUID.FromGoogleUUID(actorID)

	c, err := e.chatRepo.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}

	if err = c.SetSeverity(cmd.Severity, userID); err != nil {
		return fmt.Errorf("failed to set severity: %w", err)
	}

	return e.publishAndSave(ctx, c)
}

// publishAndSave публикует события и сохраняет aggregate
func (e *CommandExecutor) publishAndSave(ctx context.Context, c *chat.Chat) error {
	// Получение несохраненных событий
	events := c.GetUncommittedEvents()

	// Публикация событий
	for _, evt := range events {
		if err := e.eventBus.Publish(ctx, evt); err != nil {
			return fmt.Errorf("failed to publish event: %w", err)
		}
	}

	// Сохранение aggregate
	if err := e.chatRepo.Save(ctx, c); err != nil {
		return fmt.Errorf("failed to save chat: %w", err)
	}

	// Пометка событий как зафиксированных
	c.MarkEventsAsCommitted()

	return nil
}

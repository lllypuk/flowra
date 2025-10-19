package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/chat"
)

// CreateChatUseCase обрабатывает создание нового чата
type CreateChatUseCase struct {
	eventStore shared.EventStore
}

// NewCreateChatUseCase создает новый CreateChatUseCase
func NewCreateChatUseCase(eventStore shared.EventStore) *CreateChatUseCase {
	return &CreateChatUseCase{
		eventStore: eventStore,
	}
}

// Execute выполняет создание чата
func (uc *CreateChatUseCase) Execute(ctx context.Context, cmd CreateChatCommand) (Result, error) {
	// Валидация
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Создание агрегата
	chatAggregate, err := chat.NewChat(cmd.WorkspaceID, cmd.Type, cmd.IsPublic, cmd.CreatedBy)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create chat: %w", err)
	}

	// Для typed чатов (Task/Bug/Epic) конвертируем и устанавливаем title
	if cmd.Type != chat.TypeDiscussion && cmd.Title != "" {
		switch cmd.Type {
		case chat.TypeTask:
			if convertErr := chatAggregate.ConvertToTask(cmd.Title, cmd.CreatedBy); convertErr != nil {
				return Result{}, fmt.Errorf("failed to convert to task: %w", convertErr)
			}
		case chat.TypeBug:
			if convertErr := chatAggregate.ConvertToBug(cmd.Title, cmd.CreatedBy); convertErr != nil {
				return Result{}, fmt.Errorf("failed to convert to bug: %w", convertErr)
			}
		case chat.TypeEpic:
			if convertErr := chatAggregate.ConvertToEpic(cmd.Title, cmd.CreatedBy); convertErr != nil {
				return Result{}, fmt.Errorf("failed to convert to epic: %w", convertErr)
			}
		case chat.TypeDiscussion:
			// TypeDiscussion не требует конвертации
		}
	}

	// Сохранение событий
	return saveAggregate(ctx, uc.eventStore, chatAggregate, chatAggregate.ID().String())
}

func (uc *CreateChatUseCase) validate(cmd CreateChatCommand) error {
	if err := shared.ValidateUUID("workspaceID", cmd.WorkspaceID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("createdBy", cmd.CreatedBy); err != nil {
		return err
	}
	if err := shared.ValidateEnum("type", string(cmd.Type), []string{
		string(chat.TypeDiscussion),
		string(chat.TypeTask),
		string(chat.TypeBug),
		string(chat.TypeEpic),
	}); err != nil {
		return err
	}

	// Для typed чатов title обязателен
	if cmd.Type != chat.TypeDiscussion {
		if err := shared.ValidateRequired("title", cmd.Title); err != nil {
			return ErrTitleRequired
		}
		if err := shared.ValidateMaxLength("title", cmd.Title, shared.MaxTitleLength); err != nil {
			return err
		}
	}

	return nil
}

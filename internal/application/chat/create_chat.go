package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// CreateChatUseCase обрабатывает создание нового чата
type CreateChatUseCase struct {
	eventStore appcore.EventStore
}

// NewCreateChatUseCase создает новый CreateChatUseCase
func NewCreateChatUseCase(eventStore appcore.EventStore) *CreateChatUseCase {
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

	// Создание агрегата как Discussion для сохранения трейла событий конверсии
	// NewChat() автоматически генерирует события ChatCreated и ParticipantAdded
	chatAggregate, err := chat.NewChat(cmd.WorkspaceID, chat.TypeDiscussion, cmd.IsPublic, cmd.CreatedBy)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create chat: %w", err)
	}

	// Для typed чатов (Task/Bug/Epic) конвертируем и устанавливаем title
	if cmd.Type != chat.TypeDiscussion {
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
			// Discussion type is already handled above, no conversion needed
		}
	}

	// Сохранение событий
	return saveAggregate(ctx, uc.eventStore, chatAggregate, chatAggregate.ID().String())
}

func (uc *CreateChatUseCase) validate(cmd CreateChatCommand) error {
	if err := appcore.ValidateUUID("workspaceID", cmd.WorkspaceID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("createdBy", cmd.CreatedBy); err != nil {
		return err
	}
	if err := appcore.ValidateEnum("type", string(cmd.Type), []string{
		string(chat.TypeDiscussion),
		string(chat.TypeTask),
		string(chat.TypeBug),
		string(chat.TypeEpic),
	}); err != nil {
		return err
	}

	// Для typed чатов title обязателен
	if cmd.Type != chat.TypeDiscussion {
		if err := appcore.ValidateRequired("title", cmd.Title); err != nil {
			return ErrTitleRequired
		}
		if err := appcore.ValidateMaxLength("title", cmd.Title, appcore.MaxTitleLength); err != nil {
			return err
		}
	}

	return nil
}

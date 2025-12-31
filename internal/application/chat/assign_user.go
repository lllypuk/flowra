package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// AssignUserUseCase обрабатывает назначение пользователя на чат
type AssignUserUseCase struct {
	eventStore appcore.EventStore
}

// NewAssignUserUseCase создает новый AssignUserUseCase
func NewAssignUserUseCase(eventStore appcore.EventStore) *AssignUserUseCase {
	return &AssignUserUseCase{
		eventStore: eventStore,
	}
}

// Execute выполняет назначение пользователя
func (uc *AssignUserUseCase) Execute(ctx context.Context, cmd AssignUserCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := loadAggregate(ctx, uc.eventStore, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	if assignErr := chatAggregate.AssignUser(cmd.AssigneeID, cmd.AssignedBy); assignErr != nil {
		return Result{}, fmt.Errorf("failed to assign user: %w", assignErr)
	}

	return saveAggregate(ctx, uc.eventStore, chatAggregate, cmd.ChatID.String())
}

func (uc *AssignUserUseCase) validate(cmd AssignUserCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if cmd.AssigneeID != nil {
		if err := appcore.ValidateUUID("assigneeID", *cmd.AssigneeID); err != nil {
			return err
		}
	}
	if err := appcore.ValidateUUID("assignedBy", cmd.AssignedBy); err != nil {
		return err
	}
	return nil
}

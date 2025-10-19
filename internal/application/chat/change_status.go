//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/teams-up/internal/application/shared"
)

// ChangeStatusUseCase обрабатывает изменение статуса чата
type ChangeStatusUseCase struct {
	eventStore shared.EventStore
}

// NewChangeStatusUseCase создает новый ChangeStatusUseCase
func NewChangeStatusUseCase(eventStore shared.EventStore) *ChangeStatusUseCase {
	return &ChangeStatusUseCase{
		eventStore: eventStore,
	}
}

// Execute выполняет изменение статуса
func (uc *ChangeStatusUseCase) Execute(ctx context.Context, cmd ChangeStatusCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := loadAggregate(ctx, uc.eventStore, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	if statusErr := chatAggregate.ChangeStatus(cmd.Status, cmd.ChangedBy); statusErr != nil {
		return Result{}, fmt.Errorf("failed to change status: %w", statusErr)
	}

	return saveAggregate(ctx, uc.eventStore, chatAggregate, cmd.ChatID.String())
}

func (uc *ChangeStatusUseCase) validate(cmd ChangeStatusCommand) error {
	if err := shared.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := shared.ValidateRequired("status", cmd.Status); err != nil {
		return err
	}
	if err := shared.ValidateUUID("changedBy", cmd.ChangedBy); err != nil {
		return err
	}
	return nil
}

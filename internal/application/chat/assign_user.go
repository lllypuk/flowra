package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// AssignUserUseCase handles assigning a user to a chat
type AssignUserUseCase struct {
	chatRepo CommandRepository
	userRepo appcore.UserRepository
}

// NewAssignUserUseCase creates a new AssignUserUseCase
func NewAssignUserUseCase(chatRepo CommandRepository, userRepo appcore.UserRepository) *AssignUserUseCase {
	if userRepo == nil {
		panic("userRepo is required")
	}

	return &AssignUserUseCase{
		chatRepo: chatRepo,
		userRepo: userRepo,
	}
}

// Execute performs assigning a user
func (uc *AssignUserUseCase) Execute(ctx context.Context, cmd AssignUserCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}
	if validationErr := uc.validateAssigneeExists(ctx, cmd.AssigneeID); validationErr != nil {
		return Result{}, validationErr
	}

	chatAggregate, err := uc.chatRepo.Load(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load chat: %w", err)
	}

	if assignErr := chatAggregate.AssignUser(cmd.AssigneeID, cmd.AssignedBy); assignErr != nil {
		return Result{}, fmt.Errorf("failed to assign user: %w", assignErr)
	}

	// Save via repository (updates both event store and read model)
	if err = uc.chatRepo.Save(ctx, chatAggregate); err != nil {
		return Result{}, fmt.Errorf("failed to save chat: %w", err)
	}

	return Result{
		Result: appcore.Result[*chat.Chat]{
			Value:   chatAggregate,
			Version: chatAggregate.Version(),
		},
	}, nil
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

func (uc *AssignUserUseCase) validateAssigneeExists(ctx context.Context, assigneeID *uuid.UUID) error {
	if assigneeID == nil {
		return nil
	}

	exists, err := uc.userRepo.Exists(ctx, *assigneeID)
	if err != nil {
		return fmt.Errorf("failed to validate assignee existence: %w", err)
	}
	if !exists {
		return ErrAssigneeNotFound
	}
	return nil
}

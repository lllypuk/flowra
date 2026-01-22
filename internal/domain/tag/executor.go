package tag

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	chatApp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/domain/errs"
	domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

const (
	noneUsername = "@none"

	// Retry configuration for handling concurrency conflicts
	// Increased to handle high-contention scenarios where chat is being modified rapidly
	maxRetries         = 5
	initialRetryDelay  = 100 * time.Millisecond
	maxRetryDelay      = 2000 * time.Millisecond
	retryDelayMultiple = 2
)

// CommandExecutor performs tag commands via Chat UseCases
type CommandExecutor struct {
	chatUseCases *ChatUseCases
	userRepo     UserRepository
}

// NewCommandExecutor creates New CommandExecutor
func NewCommandExecutor(
	chatUseCases *ChatUseCases,
	userRepo UserRepository,
) *CommandExecutor {
	return &CommandExecutor{
		chatUseCases: chatUseCases,
		userRepo:     userRepo,
	}
}

// retryOnConcurrentModification retries an operation with exponential backoff when concurrent modification occurs.
// This is used by all executor methods to handle optimistic locking conflicts during chat operations.
func (e *CommandExecutor) retryOnConcurrentModification(
	ctx context.Context,
	operation func(context.Context) error,
	operationName string,
) error {
	var lastErr error
	delay := initialRetryDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := operation(ctx)
		if err == nil {
			return nil // Success
		}

		// Check if this is a concurrency conflict
		if !errors.Is(err, errs.ErrConcurrentModification) {
			return fmt.Errorf("%s: %w", operationName, err)
		}

		lastErr = err

		// If this was the last attempt, return the error
		if attempt == maxRetries {
			break
		}

		// Wait before retrying with exponential backoff
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay *= retryDelayMultiple
			if delay > maxRetryDelay {
				delay = maxRetryDelay
			}
		}
	}

	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// Execute performs command by dispatching to appropriate executor method.
// Each executor method handles its own retry logic for concurrency conflicts.
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
	case InviteUserCommand:
		return e.executeInviteUser(ctx, c, actorID)
	case RemoveUserCommand:
		return e.executeRemoveUser(ctx, c, actorID)
	case CloseChatCommand:
		return e.executeCloseChat(ctx, c, actorID)
	case ReopenChatCommand:
		return e.executeReopenChat(ctx, c, actorID)
	case DeleteChatCommand:
		return e.executeDeleteChat(ctx, c, actorID)
	default:
		return fmt.Errorf("unknown command type: %T", cmd)
	}
}

// executeCreateTask performs komandu creating Task via UseCase
func (e *CommandExecutor) executeCreateTask(ctx context.Context, cmd CreateTaskCommand, actorID uuid.UUID) error {
	usecaseCmd := chatApp.ConvertToTaskCommand{
		ChatID:      domainUUID.FromGoogleUUID(cmd.ChatID),
		Title:       cmd.Title,
		ConvertedBy: domainUUID.FromGoogleUUID(actorID),
	}

	_, err := e.chatUseCases.ConvertToTask.Execute(ctx, usecaseCmd)
	if err != nil {
		return fmt.Errorf("failed to convert to task: %w", err)
	}

	return nil
}

// executeCreateBug performs komandu creating Bug via UseCase
func (e *CommandExecutor) executeCreateBug(ctx context.Context, cmd CreateBugCommand, actorID uuid.UUID) error {
	usecaseCmd := chatApp.ConvertToBugCommand{
		ChatID:      domainUUID.FromGoogleUUID(cmd.ChatID),
		Title:       cmd.Title,
		ConvertedBy: domainUUID.FromGoogleUUID(actorID),
	}

	_, err := e.chatUseCases.ConvertToBug.Execute(ctx, usecaseCmd)
	if err != nil {
		return fmt.Errorf("failed to convert to bug: %w", err)
	}

	return nil
}

// executeCreateEpic performs komandu creating Epic via UseCase
func (e *CommandExecutor) executeCreateEpic(ctx context.Context, cmd CreateEpicCommand, actorID uuid.UUID) error {
	usecaseCmd := chatApp.ConvertToEpicCommand{
		ChatID:      domainUUID.FromGoogleUUID(cmd.ChatID),
		Title:       cmd.Title,
		ConvertedBy: domainUUID.FromGoogleUUID(actorID),
	}

	_, err := e.chatUseCases.ConvertToEpic.Execute(ctx, usecaseCmd)
	if err != nil {
		return fmt.Errorf("failed to convert to epic: %w", err)
	}

	return nil
}

// executeChangeStatus performs komandu changing status via UseCase
func (e *CommandExecutor) executeChangeStatus(ctx context.Context, cmd ChangeStatusCommand, actorID uuid.UUID) error {
	usecaseCmd := chatApp.ChangeStatusCommand{
		ChatID:    domainUUID.FromGoogleUUID(cmd.ChatID),
		Status:    cmd.Status,
		ChangedBy: domainUUID.FromGoogleUUID(actorID),
	}

	return e.retryOnConcurrentModification(ctx, func(ctx context.Context) error {
		_, err := e.chatUseCases.ChangeStatus.Execute(ctx, usecaseCmd)
		return err
	}, "failed to change status")
}

// executeAssignUser performs komandu assigning user via UseCase
func (e *CommandExecutor) executeAssignUser(ctx context.Context, cmd AssignUserCommand, actorID uuid.UUID) error {
	// rezolving user po username
	var assigneeID *domainUUID.UUID
	if cmd.Username != "" && cmd.Username != noneUsername {
		username := strings.TrimPrefix(cmd.Username, "@")
		u, err := e.userRepo.FindByUsername(ctx, username)
		if err != nil {
			return fmt.Errorf("user %s not found: %w", cmd.Username, err)
		}
		uid := u.ID()
		assigneeID = &uid
	}

	usecaseCmd := chatApp.AssignUserCommand{
		ChatID:     domainUUID.FromGoogleUUID(cmd.ChatID),
		AssigneeID: assigneeID,
		AssignedBy: domainUUID.FromGoogleUUID(actorID),
	}

	return e.retryOnConcurrentModification(ctx, func(ctx context.Context) error {
		_, err := e.chatUseCases.AssignUser.Execute(ctx, usecaseCmd)
		return err
	}, "failed to assign user")
}

// executeChangePriority performs komandu changing priority via UseCase
func (e *CommandExecutor) executeChangePriority(
	ctx context.Context,
	cmd ChangePriorityCommand,
	actorID uuid.UUID,
) error {
	usecaseCmd := chatApp.SetPriorityCommand{
		ChatID:   domainUUID.FromGoogleUUID(cmd.ChatID),
		Priority: cmd.Priority,
		SetBy:    domainUUID.FromGoogleUUID(actorID),
	}

	return e.retryOnConcurrentModification(ctx, func(ctx context.Context) error {
		_, err := e.chatUseCases.SetPriority.Execute(ctx, usecaseCmd)
		return err
	}, "failed to set priority")
}

// executeSetDueDate performs komandu setting deadline via UseCase
func (e *CommandExecutor) executeSetDueDate(ctx context.Context, cmd SetDueDateCommand, actorID uuid.UUID) error {
	usecaseCmd := chatApp.SetDueDateCommand{
		ChatID:  domainUUID.FromGoogleUUID(cmd.ChatID),
		DueDate: cmd.DueDate,
		SetBy:   domainUUID.FromGoogleUUID(actorID),
	}

	return e.retryOnConcurrentModification(ctx, func(ctx context.Context) error {
		_, err := e.chatUseCases.SetDueDate.Execute(ctx, usecaseCmd)
		return err
	}, "failed to set due date")
}

// executeChangeTitle performs komandu changing nazvaniya via UseCase
func (e *CommandExecutor) executeChangeTitle(ctx context.Context, cmd ChangeTitleCommand, actorID uuid.UUID) error {
	usecaseCmd := chatApp.RenameChatCommand{
		ChatID:    domainUUID.FromGoogleUUID(cmd.ChatID),
		NewTitle:  cmd.Title,
		RenamedBy: domainUUID.FromGoogleUUID(actorID),
	}

	return e.retryOnConcurrentModification(ctx, func(ctx context.Context) error {
		_, err := e.chatUseCases.Rename.Execute(ctx, usecaseCmd)
		return err
	}, "failed to rename")
}

// executeSetSeverity performs komandu setting severity via UseCase
func (e *CommandExecutor) executeSetSeverity(ctx context.Context, cmd SetSeverityCommand, actorID uuid.UUID) error {
	usecaseCmd := chatApp.SetSeverityCommand{
		ChatID:   domainUUID.FromGoogleUUID(cmd.ChatID),
		Severity: cmd.Severity,
		SetBy:    domainUUID.FromGoogleUUID(actorID),
	}

	return e.retryOnConcurrentModification(ctx, func(ctx context.Context) error {
		_, err := e.chatUseCases.SetSeverity.Execute(ctx, usecaseCmd)
		return err
	}, "failed to set severity")
}

// Task 007a: Participant Management and Chat Lifecycle Executors

// executeInviteUser performs command to add a participant to the chat
func (e *CommandExecutor) executeInviteUser(ctx context.Context, cmd InviteUserCommand, actorID uuid.UUID) error {
	// Resolve username to userID
	username := strings.TrimPrefix(cmd.Username, "@")
	user, userErr := e.userRepo.FindByUsername(ctx, username)
	if userErr != nil {
		return fmt.Errorf("user @%s not found: %w", username, userErr)
	}

	// Call AddParticipant use case with retry
	addCmd := chatApp.AddParticipantCommand{
		ChatID:  domainUUID.FromGoogleUUID(cmd.ChatID),
		UserID:  user.ID(),
		Role:    "Member", // Default role
		AddedBy: domainUUID.FromGoogleUUID(actorID),
	}

	return e.retryOnConcurrentModification(ctx, func(ctx context.Context) error {
		_, err := e.chatUseCases.AddParticipant.Execute(ctx, addCmd)
		return err
	}, "failed to add participant")
}

// executeRemoveUser performs command to remove a participant from the chat
func (e *CommandExecutor) executeRemoveUser(ctx context.Context, cmd RemoveUserCommand, actorID uuid.UUID) error {
	// Resolve username to userID
	username := strings.TrimPrefix(cmd.Username, "@")
	user, userErr := e.userRepo.FindByUsername(ctx, username)
	if userErr != nil {
		return fmt.Errorf("user @%s not found: %w", username, userErr)
	}

	// Call RemoveParticipant use case with retry
	removeCmd := chatApp.RemoveParticipantCommand{
		ChatID:    domainUUID.FromGoogleUUID(cmd.ChatID),
		UserID:    user.ID(),
		RemovedBy: domainUUID.FromGoogleUUID(actorID),
	}

	return e.retryOnConcurrentModification(ctx, func(ctx context.Context) error {
		_, err := e.chatUseCases.RemoveParticipant.Execute(ctx, removeCmd)
		return err
	}, "failed to remove participant")
}

// executeCloseChat performs command to close/archive the chat
func (e *CommandExecutor) executeCloseChat(ctx context.Context, cmd CloseChatCommand, actorID uuid.UUID) error {
	closeCmd := chatApp.CloseChatCommand{
		ChatID:   domainUUID.FromGoogleUUID(cmd.ChatID),
		ClosedBy: domainUUID.FromGoogleUUID(actorID),
	}

	return e.retryOnConcurrentModification(ctx, func(ctx context.Context) error {
		_, err := e.chatUseCases.CloseChat.Execute(ctx, closeCmd)
		return err
	}, "failed to close chat")
}

// executeReopenChat performs command to reopen a closed chat
func (e *CommandExecutor) executeReopenChat(ctx context.Context, cmd ReopenChatCommand, actorID uuid.UUID) error {
	reopenCmd := chatApp.ReopenChatCommand{
		ChatID:     domainUUID.FromGoogleUUID(cmd.ChatID),
		ReopenedBy: domainUUID.FromGoogleUUID(actorID),
	}

	return e.retryOnConcurrentModification(ctx, func(ctx context.Context) error {
		_, err := e.chatUseCases.ReopenChat.Execute(ctx, reopenCmd)
		return err
	}, "failed to reopen chat")
}

// executeDeleteChat performs command to delete the chat
func (e *CommandExecutor) executeDeleteChat(_ context.Context, _ DeleteChatCommand, _ uuid.UUID) error {
	// Use the existing Delete method on Chat domain
	return errors.New("delete chat not yet implemented - needs DeleteChatUseCase")
}

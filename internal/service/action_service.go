package service

import (
	"context"
	"fmt"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
	messageapp "github.com/lllypuk/flowra/internal/application/message"
	"github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// ActionService converts UI actions to chat messages with tags
type ActionService struct {
	sendMessageUC *messageapp.SendMessageUseCase
	userRepo      appcore.UserRepository
}

// NewActionService creates a new ActionService
func NewActionService(
	sendMessageUC *messageapp.SendMessageUseCase,
	userRepo appcore.UserRepository,
) *ActionService {
	return &ActionService{
		sendMessageUC: sendMessageUC,
		userRepo:      userRepo,
	}
}

// ChangeStatus creates a system message to change entity status
func (s *ActionService) ChangeStatus(
	ctx context.Context,
	chatID uuid.UUID,
	newStatus string,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	content := fmt.Sprintf("#status %s", newStatus)
	return s.executeAction(ctx, chatID, content, actorID)
}

// AssignUser creates a system message to assign a user
func (s *ActionService) AssignUser(
	ctx context.Context,
	chatID uuid.UUID,
	assigneeID *uuid.UUID,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	var content string
	if assigneeID == nil {
		content = "#assignee @none"
	} else {
		// Resolve userID to username
		usr, err := s.userRepo.GetByID(ctx, *assigneeID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve user: %w", err)
		}
		content = fmt.Sprintf("#assignee @%s", usr.Username)
	}
	return s.executeAction(ctx, chatID, content, actorID)
}

// SetPriority creates a system message to change priority
func (s *ActionService) SetPriority(
	ctx context.Context,
	chatID uuid.UUID,
	priority string,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	content := fmt.Sprintf("#priority %s", priority)
	return s.executeAction(ctx, chatID, content, actorID)
}

// SetDueDate creates a system message to set due date
func (s *ActionService) SetDueDate(
	ctx context.Context,
	chatID uuid.UUID,
	dueDate *time.Time,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	var content string
	if dueDate == nil {
		content = "#due"
	} else {
		content = fmt.Sprintf("#due %s", dueDate.Format("2006-01-02"))
	}
	return s.executeAction(ctx, chatID, content, actorID)
}

// InviteUser creates a system message to add a participant
func (s *ActionService) InviteUser(
	ctx context.Context,
	chatID uuid.UUID,
	userID uuid.UUID,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	usr, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}
	content := fmt.Sprintf("#invite @%s", usr.Username)
	return s.executeAction(ctx, chatID, content, actorID)
}

// RemoveUser creates a system message to remove a participant
func (s *ActionService) RemoveUser(
	ctx context.Context,
	chatID uuid.UUID,
	userID uuid.UUID,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	usr, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}
	content := fmt.Sprintf("#remove @%s", usr.Username)
	return s.executeAction(ctx, chatID, content, actorID)
}

// Rename creates a system message to rename the chat
func (s *ActionService) Rename(
	ctx context.Context,
	chatID uuid.UUID,
	newTitle string,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	content := fmt.Sprintf("#title %s", newTitle)
	return s.executeAction(ctx, chatID, content, actorID)
}

// Close creates a system message to close the chat
func (s *ActionService) Close(
	ctx context.Context,
	chatID uuid.UUID,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	return s.executeAction(ctx, chatID, "#close", actorID)
}

// Reopen creates a system message to reopen the chat
func (s *ActionService) Reopen(
	ctx context.Context,
	chatID uuid.UUID,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	return s.executeAction(ctx, chatID, "#reopen", actorID)
}

// Delete creates a system message to delete the chat
func (s *ActionService) Delete(
	ctx context.Context,
	chatID uuid.UUID,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	return s.executeAction(ctx, chatID, "#delete", actorID)
}

// executeAction is the common implementation for all actions
func (s *ActionService) executeAction(
	ctx context.Context,
	chatID uuid.UUID,
	content string,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	cmd := messageapp.SendMessageCommand{
		ChatID:   chatID,
		AuthorID: actorID,
		Content:  content,
		Type:     message.TypeSystem,
		ActorID:  &actorID,
	}

	result, err := s.sendMessageUC.Execute(ctx, cmd)
	if err != nil {
		return &appcore.ActionResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return &appcore.ActionResult{
		MessageID: result.Value.ID(),
		Success:   true,
	}, nil
}

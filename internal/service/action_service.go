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

// ActionService converts UI actions to chat messages with human-readable content
type ActionService struct {
	sendMessageUC *messageapp.SendMessageUseCase
	userRepo      appcore.UserRepository
	batcher       *ChangeBatcher
}

// NewActionService creates a new ActionService
func NewActionService(
	sendMessageUC *messageapp.SendMessageUseCase,
	userRepo appcore.UserRepository,
) *ActionService {
	svc := &ActionService{
		sendMessageUC: sendMessageUC,
		userRepo:      userRepo,
	}

	// Initialize batcher with flush function
	svc.batcher = NewChangeBatcher(2*time.Second, svc.flushBatchMessage)

	return svc
}

// getActorDisplayName returns the display name for the actor
func (s *ActionService) getActorDisplayName(ctx context.Context, actorID uuid.UUID) string {
	if s.userRepo == nil {
		return ""
	}
	usr, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil || usr == nil {
		return ""
	}
	return usr.FullName
}

// ChangeStatus creates a system message to change entity status
func (s *ActionService) ChangeStatus(
	ctx context.Context,
	chatID uuid.UUID,
	newStatus string,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	actorName := s.getActorDisplayName(ctx, actorID)
	
	// Add to batch instead of immediate execution
	err := s.batcher.AddChange(ctx, actorID, chatID, actorName, ChangeTypeStatus, newStatus)
	if err != nil {
		return &appcore.ActionResult{Success: false, Error: err.Error()}, err
	}
	
	return &appcore.ActionResult{Success: true}, nil
}

// AssignUser creates a system message to assign a user
func (s *ActionService) AssignUser(
	ctx context.Context,
	chatID uuid.UUID,
	assigneeID *uuid.UUID,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	actorName := s.getActorDisplayName(ctx, actorID)

	var assigneeName string
	if assigneeID == nil {
		assigneeName = "" // Empty means removed
	} else {
		name, err := s.resolveUserDisplayName(ctx, *assigneeID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve user: %w", err)
		}
		assigneeName = name
	}

	err := s.batcher.AddChange(ctx, actorID, chatID, actorName, ChangeTypeAssignee, assigneeName)
	if err != nil {
		return &appcore.ActionResult{Success: false, Error: err.Error()}, err
	}

	return &appcore.ActionResult{Success: true}, nil
}

// SetPriority creates a system message to change priority
func (s *ActionService) SetPriority(
	ctx context.Context,
	chatID uuid.UUID,
	priority string,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	actorName := s.getActorDisplayName(ctx, actorID)
	
	err := s.batcher.AddChange(ctx, actorID, chatID, actorName, ChangeTypePriority, priority)
	if err != nil {
		return &appcore.ActionResult{Success: false, Error: err.Error()}, err
	}
	
	return &appcore.ActionResult{Success: true}, nil
}

// SetDueDate creates a system message to set due date
func (s *ActionService) SetDueDate(
	ctx context.Context,
	chatID uuid.UUID,
	dueDate *time.Time,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	actorName := s.getActorDisplayName(ctx, actorID)

	var formattedDate string
	if dueDate == nil {
		formattedDate = "" // Empty means removed
	} else {
		formattedDate = dueDate.Format("January 2, 2006")
	}

	err := s.batcher.AddChange(ctx, actorID, chatID, actorName, ChangeTypeDueDate, formattedDate)
	if err != nil {
		return &appcore.ActionResult{Success: false, Error: err.Error()}, err
	}

	return &appcore.ActionResult{Success: true}, nil
}

// InviteUser creates a system message to add a participant
func (s *ActionService) InviteUser(
	ctx context.Context,
	chatID uuid.UUID,
	userID uuid.UUID,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	actorName := s.getActorDisplayName(ctx, actorID)
	usr, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}
	inviteeName := usr.FullName
	if inviteeName == "" {
		inviteeName = usr.Username
	}
	var content string
	if actorName != "" {
		content = fmt.Sprintf("✅ %s invited %s to the chat", actorName, inviteeName)
	} else {
		content = fmt.Sprintf("✅ %s was invited to the chat", inviteeName)
	}
	return s.executeAction(ctx, chatID, content, actorID)
}

// RemoveUser creates a system message to remove a participant
func (s *ActionService) RemoveUser(
	ctx context.Context,
	chatID uuid.UUID,
	userID uuid.UUID,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	actorName := s.getActorDisplayName(ctx, actorID)
	usr, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}
	removedName := usr.FullName
	if removedName == "" {
		removedName = usr.Username
	}
	var content string
	if actorName != "" {
		content = fmt.Sprintf("✅ %s removed %s from the chat", actorName, removedName)
	} else {
		content = fmt.Sprintf("✅ %s was removed from the chat", removedName)
	}
	return s.executeAction(ctx, chatID, content, actorID)
}

// Rename creates a system message to rename the chat
func (s *ActionService) Rename(
	ctx context.Context,
	chatID uuid.UUID,
	newTitle string,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	actorName := s.getActorDisplayName(ctx, actorID)
	var content string
	if actorName != "" {
		content = fmt.Sprintf("✅ %s changed title to: %s", actorName, newTitle)
	} else {
		content = fmt.Sprintf("✅ Title changed to: %s", newTitle)
	}
	return s.executeAction(ctx, chatID, content, actorID)
}

// Close creates a system message to close the chat
func (s *ActionService) Close(
	ctx context.Context,
	chatID uuid.UUID,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	actorName := s.getActorDisplayName(ctx, actorID)
	var content string
	if actorName != "" {
		content = fmt.Sprintf("✅ %s closed the chat", actorName)
	} else {
		content = "✅ Chat closed"
	}
	return s.executeAction(ctx, chatID, content, actorID)
}

// Reopen creates a system message to reopen the chat
func (s *ActionService) Reopen(
	ctx context.Context,
	chatID uuid.UUID,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	actorName := s.getActorDisplayName(ctx, actorID)
	var content string
	if actorName != "" {
		content = fmt.Sprintf("✅ %s reopened the chat", actorName)
	} else {
		content = "✅ Chat reopened"
	}
	return s.executeAction(ctx, chatID, content, actorID)
}

// Delete creates a system message to delete the chat
func (s *ActionService) Delete(
	ctx context.Context,
	chatID uuid.UUID,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	actorName := s.getActorDisplayName(ctx, actorID)
	var content string
	if actorName != "" {
		content = fmt.Sprintf("✅ %s deleted the chat", actorName)
	} else {
		content = "✅ Chat deleted"
	}
	return s.executeAction(ctx, chatID, content, actorID)
}

// formatAction formats a message with or without actor name
func (s *ActionService) formatAction(actorName, actionWithActor, actionWithoutActor string) string {
	if actorName != "" {
		return fmt.Sprintf("✅ %s %s", actorName, actionWithActor)
	}
	return fmt.Sprintf("✅ %s", actionWithoutActor)
}

// formatActionWithTarget formats a message with a target value
func (s *ActionService) formatActionWithTarget(actorName, actionWithActor, actionWithoutActor, target string) string {
	if actorName != "" {
		return fmt.Sprintf("✅ %s %s %s", actorName, actionWithActor, target)
	}
	return fmt.Sprintf("✅ %s %s", actionWithoutActor, target)
}

// resolveUserDisplayName resolves user ID to display name
func (s *ActionService) resolveUserDisplayName(ctx context.Context, userID uuid.UUID) (string, error) {
	usr, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if usr.FullName != "" {
		return usr.FullName, nil
	}
	return usr.Username, nil
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

// flushBatchMessage is called by the batcher to send a combined message
func (s *ActionService) flushBatchMessage(
	ctx context.Context,
	chatID uuid.UUID,
	content string,
	actorID uuid.UUID,
) error {
	_, err := s.executeAction(ctx, chatID, content, actorID)
	return err
}

// Shutdown stops the batcher and cleans up resources
func (s *ActionService) Shutdown() {
	if s.batcher != nil {
		s.batcher.Close()
	}
}

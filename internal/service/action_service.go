package service

import (
	"context"
	"fmt"
	"log/slog"
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
	logger        *slog.Logger
}

// NewActionService creates a new ActionService
func NewActionService(
	sendMessageUC *messageapp.SendMessageUseCase,
	userRepo appcore.UserRepository,
) *ActionService {
	svc := &ActionService{
		sendMessageUC: sendMessageUC,
		userRepo:      userRepo,
		logger:        slog.Default(),
	}

	// Initialize batcher with flush function and logger
	svc.batcher = NewChangeBatcher(defaultBatchWindow, svc.flushBatchMessage)
	svc.batcher.logger = svc.logger

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

// ChangeStatus executes status change via tag command and batches the human-readable message
func (s *ActionService) ChangeStatus(
	ctx context.Context,
	chatID uuid.UUID,
	newStatus string,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	// Execute the actual domain change via tag command
	tagContent := fmt.Sprintf("#status %s", newStatus)
	cmd := messageapp.SendMessageCommand{
		ChatID:   chatID,
		AuthorID: actorID,
		Content:  tagContent,
		Type:     message.TypeSystem,
		ActorID:  &actorID,
	}
	
	if _, err := s.sendMessageUC.Execute(ctx, cmd); err != nil {
		return &appcore.ActionResult{Success: false, Error: err.Error()}, err
	}

	// Add human-readable message to batch
	actorName := s.getActorDisplayName(ctx, actorID)
	err := s.batcher.AddChange(ctx, actorID, chatID, actorName, ChangeTypeStatus, newStatus)
	if err != nil {
		s.logger.Warn("failed to batch status change message", "error", err)
	}

	return &appcore.ActionResult{Success: true}, nil
}

// AssignUser executes assignee change via tag command and batches the human-readable message
func (s *ActionService) AssignUser(
	ctx context.Context,
	chatID uuid.UUID,
	assigneeID *uuid.UUID,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	// Execute the actual domain change via tag command
	var tagContent string
	var assigneeName string
	if assigneeID == nil {
		tagContent = "#assignee @none"
		assigneeName = ""
	} else {
		usr, err := s.userRepo.GetByID(ctx, *assigneeID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve user: %w", err)
		}
		tagContent = fmt.Sprintf("#assignee @%s", usr.Username)
		assigneeName = usr.FullName
		if assigneeName == "" {
			assigneeName = usr.Username
		}
	}

	cmd := messageapp.SendMessageCommand{
		ChatID:   chatID,
		AuthorID: actorID,
		Content:  tagContent,
		Type:     message.TypeSystem,
		ActorID:  &actorID,
	}
	
	if _, err := s.sendMessageUC.Execute(ctx, cmd); err != nil {
		return &appcore.ActionResult{Success: false, Error: err.Error()}, err
	}

	// Add human-readable message to batch
	actorName := s.getActorDisplayName(ctx, actorID)
	err := s.batcher.AddChange(ctx, actorID, chatID, actorName, ChangeTypeAssignee, assigneeName)
	if err != nil {
		s.logger.Warn("failed to batch assignee change message", "error", err)
	}

	return &appcore.ActionResult{Success: true}, nil
}

// SetPriority executes priority change via tag command and batches the human-readable message
func (s *ActionService) SetPriority(
	ctx context.Context,
	chatID uuid.UUID,
	priority string,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	// Execute the actual domain change via tag command
	tagContent := fmt.Sprintf("#priority %s", priority)
	cmd := messageapp.SendMessageCommand{
		ChatID:   chatID,
		AuthorID: actorID,
		Content:  tagContent,
		Type:     message.TypeSystem,
		ActorID:  &actorID,
	}
	
	if _, err := s.sendMessageUC.Execute(ctx, cmd); err != nil {
		return &appcore.ActionResult{Success: false, Error: err.Error()}, err
	}

	// Add human-readable message to batch
	actorName := s.getActorDisplayName(ctx, actorID)
	err := s.batcher.AddChange(ctx, actorID, chatID, actorName, ChangeTypePriority, priority)
	if err != nil {
		s.logger.Warn("failed to batch priority change message", "error", err)
	}

	return &appcore.ActionResult{Success: true}, nil
}

// SetDueDate executes due date change via tag command and batches the human-readable message
func (s *ActionService) SetDueDate(
	ctx context.Context,
	chatID uuid.UUID,
	dueDate *time.Time,
	actorID uuid.UUID,
) (*appcore.ActionResult, error) {
	// Execute the actual domain change via tag command
	var tagContent string
	var formattedDate string
	if dueDate == nil {
		tagContent = "#due none"
		formattedDate = ""
	} else {
		tagContent = fmt.Sprintf("#due %s", dueDate.Format("2006-01-02"))
		formattedDate = dueDate.Format("January 2, 2006")
	}

	cmd := messageapp.SendMessageCommand{
		ChatID:   chatID,
		AuthorID: actorID,
		Content:  tagContent,
		Type:     message.TypeSystem,
		ActorID:  &actorID,
	}
	
	if _, err := s.sendMessageUC.Execute(ctx, cmd); err != nil {
		return &appcore.ActionResult{Success: false, Error: err.Error()}, err
	}

	// Add human-readable message to batch
	actorName := s.getActorDisplayName(ctx, actorID)
	err := s.batcher.AddChange(ctx, actorID, chatID, actorName, ChangeTypeDueDate, formattedDate)
	if err != nil {
		s.logger.Warn("failed to batch due date change message", "error", err)
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

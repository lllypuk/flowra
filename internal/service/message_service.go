// Package service provides business logic services that orchestrate use cases.
package service

import (
	"context"

	messageapp "github.com/lllypuk/flowra/internal/application/message"
	"github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// MessageService provides message-related business operations.
type MessageService struct {
	sendMessageUC    *messageapp.SendMessageUseCase
	listMessagesUC   *messageapp.ListMessagesUseCase
	editMessageUC    *messageapp.EditMessageUseCase
	deleteMessageUC  *messageapp.DeleteMessageUseCase
	getMessageUC     *messageapp.GetMessageUseCase
	addReactionUC    *messageapp.AddReactionUseCase
	removeReactionUC *messageapp.RemoveReactionUseCase
	addAttachmentUC  *messageapp.AddAttachmentUseCase
}

// MessageServiceOption configures the MessageService.
type MessageServiceOption func(*MessageService)

// WithSendMessageUseCase sets the send message use case.
func WithSendMessageUseCase(uc *messageapp.SendMessageUseCase) MessageServiceOption {
	return func(s *MessageService) {
		s.sendMessageUC = uc
	}
}

// WithListMessagesUseCase sets the list messages use case.
func WithListMessagesUseCase(uc *messageapp.ListMessagesUseCase) MessageServiceOption {
	return func(s *MessageService) {
		s.listMessagesUC = uc
	}
}

// WithEditMessageUseCase sets the edit message use case.
func WithEditMessageUseCase(uc *messageapp.EditMessageUseCase) MessageServiceOption {
	return func(s *MessageService) {
		s.editMessageUC = uc
	}
}

// WithDeleteMessageUseCase sets the delete message use case.
func WithDeleteMessageUseCase(uc *messageapp.DeleteMessageUseCase) MessageServiceOption {
	return func(s *MessageService) {
		s.deleteMessageUC = uc
	}
}

// WithGetMessageUseCase sets the get message use case.
func WithGetMessageUseCase(uc *messageapp.GetMessageUseCase) MessageServiceOption {
	return func(s *MessageService) {
		s.getMessageUC = uc
	}
}

// WithAddReactionUseCase sets the add reaction use case.
func WithAddReactionUseCase(uc *messageapp.AddReactionUseCase) MessageServiceOption {
	return func(s *MessageService) {
		s.addReactionUC = uc
	}
}

// WithRemoveReactionUseCase sets the remove reaction use case.
func WithRemoveReactionUseCase(uc *messageapp.RemoveReactionUseCase) MessageServiceOption {
	return func(s *MessageService) {
		s.removeReactionUC = uc
	}
}

// WithAddAttachmentUseCase sets the add attachment use case.
func WithAddAttachmentUseCase(uc *messageapp.AddAttachmentUseCase) MessageServiceOption {
	return func(s *MessageService) {
		s.addAttachmentUC = uc
	}
}

// NewMessageService creates a new MessageService.
func NewMessageService(opts ...MessageServiceOption) *MessageService {
	s := &MessageService{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SendMessage sends a new message.
func (s *MessageService) SendMessage(
	ctx context.Context,
	cmd messageapp.SendMessageCommand,
) (messageapp.Result, error) {
	if s.sendMessageUC == nil {
		return messageapp.Result{}, messageapp.ErrMessageNotFound
	}
	return s.sendMessageUC.Execute(ctx, cmd)
}

// ListMessages lists messages in a chat.
func (s *MessageService) ListMessages(
	ctx context.Context,
	query messageapp.ListMessagesQuery,
) (messageapp.ListResult, error) {
	if s.listMessagesUC == nil {
		return messageapp.ListResult{}, messageapp.ErrMessageNotFound
	}
	return s.listMessagesUC.Execute(ctx, query)
}

// EditMessage edits a message.
func (s *MessageService) EditMessage(
	ctx context.Context,
	cmd messageapp.EditMessageCommand,
) (messageapp.Result, error) {
	if s.editMessageUC == nil {
		return messageapp.Result{}, messageapp.ErrMessageNotFound
	}
	return s.editMessageUC.Execute(ctx, cmd)
}

// DeleteMessage soft-deletes a message.
func (s *MessageService) DeleteMessage(
	ctx context.Context,
	cmd messageapp.DeleteMessageCommand,
) (messageapp.Result, error) {
	if s.deleteMessageUC == nil {
		return messageapp.Result{}, messageapp.ErrMessageNotFound
	}
	return s.deleteMessageUC.Execute(ctx, cmd)
}

// GetMessage gets a message by ID.
func (s *MessageService) GetMessage(ctx context.Context, messageID uuid.UUID) (*message.Message, error) {
	if s.getMessageUC == nil {
		return nil, messageapp.ErrMessageNotFound
	}
	query := messageapp.GetMessageQuery{
		MessageID: messageID,
	}
	result, err := s.getMessageUC.Execute(ctx, query)
	if err != nil {
		return nil, err
	}
	return result.Value, nil
}

// AddReaction adds a reaction to a message.
func (s *MessageService) AddReaction(
	ctx context.Context,
	cmd messageapp.AddReactionCommand,
) (messageapp.Result, error) {
	if s.addReactionUC == nil {
		return messageapp.Result{}, messageapp.ErrMessageNotFound
	}
	return s.addReactionUC.Execute(ctx, cmd)
}

// RemoveReaction removes a reaction from a message.
func (s *MessageService) RemoveReaction(
	ctx context.Context,
	cmd messageapp.RemoveReactionCommand,
) (messageapp.Result, error) {
	if s.removeReactionUC == nil {
		return messageapp.Result{}, messageapp.ErrMessageNotFound
	}
	return s.removeReactionUC.Execute(ctx, cmd)
}

// AddAttachment adds an attachment to a message.
func (s *MessageService) AddAttachment(
	ctx context.Context,
	cmd messageapp.AddAttachmentCommand,
) (messageapp.Result, error) {
	if s.addAttachmentUC == nil {
		return messageapp.Result{}, messageapp.ErrMessageNotFound
	}
	return s.addAttachmentUC.Execute(ctx, cmd)
}

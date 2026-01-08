package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/HTTP"
)

// Compile-time assertion that ChatService implements httphandler.ChatService.
var _ httphandler.ChatService = (*ChatService)(nil)

// CreateChatUseCase defines interface for use case creating chat.
type CreateChatUseCase interface {
	Execute(ctx context.Context, cmd chatapp.CreateChatCommand) (chatapp.Result, error)
}

// GetChatUseCase defines interface for use case receivения chat.
type GetChatUseCase interface {
	Execute(ctx context.Context, query chatapp.GetChatQuery) (*chatapp.GetChatResult, error)
}

// ListChatsUseCase defines interface for use case list chats.
type ListChatsUseCase interface {
	Execute(ctx context.Context, query chatapp.ListChatsQuery) (*chatapp.ListChatsResult, error)
}

// RenameChatUseCase defines interface for use case переименования chat.
type RenameChatUseCase interface {
	Execute(ctx context.Context, cmd chatapp.RenameChatCommand) (chatapp.Result, error)
}

// AddParticipantUseCase defines interface for use case adding participant.
type AddParticipantUseCase interface {
	Execute(ctx context.Context, cmd chatapp.AddParticipantCommand) (chatapp.Result, error)
}

// RemoveParticipantUseCase defines interface for use case removing participant.
type RemoveParticipantUseCase interface {
	Execute(ctx context.Context, cmd chatapp.RemoveParticipantCommand) (chatapp.Result, error)
}

// ChatService реализует httphandler.ChatService.
// Объединяет existingие use cases for workы с чатами.
type ChatService struct {
	createUC     CreateChatUseCase
	getUC        GetChatUseCase
	listUC       ListChatsUseCase
	renameUC     RenameChatUseCase
	addPartUC    AddParticipantUseCase
	removePartUC RemoveParticipantUseCase
	eventStore   appcore.EventStore
}

// ChatServiceConfig contains зависимости for ChatService.
type ChatServiceConfig struct {
	CreateUC     CreateChatUseCase
	GetUC        GetChatUseCase
	ListUC       ListChatsUseCase
	RenameUC     RenameChatUseCase
	AddPartUC    AddParticipantUseCase
	RemovePartUC RemoveParticipantUseCase
	EventStore   appcore.EventStore
}

// NewChatService создаёт New ChatService.
func NewChatService(cfg ChatServiceConfig) *ChatService {
	return &ChatService{
		createUC:     cfg.CreateUC,
		getUC:        cfg.GetUC,
		listUC:       cfg.ListUC,
		renameUC:     cfg.RenameUC,
		addPartUC:    cfg.AddPartUC,
		removePartUC: cfg.RemovePartUC,
		eventStore:   cfg.EventStore,
	}
}

// CreateChat создаёт New chat.
func (s *ChatService) CreateChat(
	ctx context.Context,
	cmd chatapp.CreateChatCommand,
) (chatapp.Result, error) {
	return s.createUC.Execute(ctx, cmd)
}

// GetChat returns chat по ID.
func (s *ChatService) GetChat(
	ctx context.Context,
	query chatapp.GetChatQuery,
) (*chatapp.GetChatResult, error) {
	return s.getUC.Execute(ctx, query)
}

// ListChats returns list chats workspace.
func (s *ChatService) ListChats(
	ctx context.Context,
	query chatapp.ListChatsQuery,
) (*chatapp.ListChatsResult, error) {
	return s.listUC.Execute(ctx, query)
}

// RenameChat переименовывает chat.
func (s *ChatService) RenameChat(
	ctx context.Context,
	cmd chatapp.RenameChatCommand,
) (chatapp.Result, error) {
	return s.renameUC.Execute(ctx, cmd)
}

// AddParticipant добавляет participant in chat.
func (s *ChatService) AddParticipant(
	ctx context.Context,
	cmd chatapp.AddParticipantCommand,
) (chatapp.Result, error) {
	return s.addPartUC.Execute(ctx, cmd)
}

// RemoveParticipant удаляет participant from chat.
func (s *ChatService) RemoveParticipant(
	ctx context.Context,
	cmd chatapp.RemoveParticipantCommand,
) (chatapp.Result, error) {
	return s.removePartUC.Execute(ctx, cmd)
}

// DeleteChat удаляет chat (soft delete via event sourcing).
func (s *ChatService) DeleteChat(
	ctx context.Context,
	chatID, deletedBy uuid.UUID,
) error {
	// Validate input
	if chatID.IsZero() {
		return errors.New("chatID is required")
	}
	if deletedBy.IsZero() {
		return errors.New("deletedBy is required")
	}

	// Loading aggregate from event store
	chatAggregate, err := s.loadAggregate(ctx, chatID)
	if err != nil {
		return err
	}

	// Применяем команду removing
	if deleteErr := chatAggregate.Delete(deletedBy); deleteErr != nil {
		return fmt.Errorf("failed to delete chat: %w", deleteErr)
	}

	// Saving event
	return s.saveAggregate(ctx, chatAggregate)
}

// loadAggregate loads Chat aggregate from event store.
func (s *ChatService) loadAggregate(ctx context.Context, chatID uuid.UUID) (*chat.Chat, error) {
	events, err := s.eventStore.LoadEvents(ctx, chatID.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", chatapp.ErrChatNotFound, err)
	}

	if len(events) == 0 {
		return nil, chatapp.ErrChatNotFound
	}

	chatAggregate := &chat.Chat{}
	for _, evt := range events {
		if applyErr := chatAggregate.Apply(evt); applyErr != nil {
			return nil, fmt.Errorf("failed to apply event: %w", applyErr)
		}
	}

	return chatAggregate, nil
}

// saveAggregate saves новые event aggregate.
func (s *ChatService) saveAggregate(ctx context.Context, chatAggregate *chat.Chat) error {
	newEvents := chatAggregate.GetUncommittedEvents()
	if len(newEvents) == 0 {
		return nil
	}

	currentVersion, _ := s.eventStore.GetVersion(ctx, chatAggregate.ID().String())

	if err := s.eventStore.SaveEvents(ctx, chatAggregate.ID().String(), newEvents, currentVersion); err != nil {
		if errors.Is(err, appcore.ErrConcurrencyConflict) {
			return appcore.ErrConcurrentUpdate
		}
		return fmt.Errorf("failed to save events: %w", err)
	}

	chatAggregate.MarkEventsAsCommitted()
	return nil
}

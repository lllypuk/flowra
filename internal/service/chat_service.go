package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
)

// Compile-time assertion that ChatService implements httphandler.ChatService.
var _ httphandler.ChatService = (*ChatService)(nil)

// CreateChatUseCase определяет интерфейс для use case создания чата.
type CreateChatUseCase interface {
	Execute(ctx context.Context, cmd chatapp.CreateChatCommand) (chatapp.Result, error)
}

// GetChatUseCase определяет интерфейс для use case получения чата.
type GetChatUseCase interface {
	Execute(ctx context.Context, query chatapp.GetChatQuery) (*chatapp.GetChatResult, error)
}

// ListChatsUseCase определяет интерфейс для use case списка чатов.
type ListChatsUseCase interface {
	Execute(ctx context.Context, query chatapp.ListChatsQuery) (*chatapp.ListChatsResult, error)
}

// RenameChatUseCase определяет интерфейс для use case переименования чата.
type RenameChatUseCase interface {
	Execute(ctx context.Context, cmd chatapp.RenameChatCommand) (chatapp.Result, error)
}

// AddParticipantUseCase определяет интерфейс для use case добавления участника.
type AddParticipantUseCase interface {
	Execute(ctx context.Context, cmd chatapp.AddParticipantCommand) (chatapp.Result, error)
}

// RemoveParticipantUseCase определяет интерфейс для use case удаления участника.
type RemoveParticipantUseCase interface {
	Execute(ctx context.Context, cmd chatapp.RemoveParticipantCommand) (chatapp.Result, error)
}

// ChatService реализует httphandler.ChatService.
// Объединяет существующие use cases для работы с чатами.
type ChatService struct {
	createUC     CreateChatUseCase
	getUC        GetChatUseCase
	listUC       ListChatsUseCase
	renameUC     RenameChatUseCase
	addPartUC    AddParticipantUseCase
	removePartUC RemoveParticipantUseCase
	eventStore   appcore.EventStore
}

// ChatServiceConfig содержит зависимости для ChatService.
type ChatServiceConfig struct {
	CreateUC     CreateChatUseCase
	GetUC        GetChatUseCase
	ListUC       ListChatsUseCase
	RenameUC     RenameChatUseCase
	AddPartUC    AddParticipantUseCase
	RemovePartUC RemoveParticipantUseCase
	EventStore   appcore.EventStore
}

// NewChatService создаёт новый ChatService.
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

// CreateChat создаёт новый чат.
func (s *ChatService) CreateChat(
	ctx context.Context,
	cmd chatapp.CreateChatCommand,
) (chatapp.Result, error) {
	return s.createUC.Execute(ctx, cmd)
}

// GetChat возвращает чат по ID.
func (s *ChatService) GetChat(
	ctx context.Context,
	query chatapp.GetChatQuery,
) (*chatapp.GetChatResult, error) {
	return s.getUC.Execute(ctx, query)
}

// ListChats возвращает список чатов workspace.
func (s *ChatService) ListChats(
	ctx context.Context,
	query chatapp.ListChatsQuery,
) (*chatapp.ListChatsResult, error) {
	return s.listUC.Execute(ctx, query)
}

// RenameChat переименовывает чат.
func (s *ChatService) RenameChat(
	ctx context.Context,
	cmd chatapp.RenameChatCommand,
) (chatapp.Result, error) {
	return s.renameUC.Execute(ctx, cmd)
}

// AddParticipant добавляет участника в чат.
func (s *ChatService) AddParticipant(
	ctx context.Context,
	cmd chatapp.AddParticipantCommand,
) (chatapp.Result, error) {
	return s.addPartUC.Execute(ctx, cmd)
}

// RemoveParticipant удаляет участника из чата.
func (s *ChatService) RemoveParticipant(
	ctx context.Context,
	cmd chatapp.RemoveParticipantCommand,
) (chatapp.Result, error) {
	return s.removePartUC.Execute(ctx, cmd)
}

// DeleteChat удаляет чат (soft delete через event sourcing).
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

	// Загружаем агрегат из event store
	chatAggregate, err := s.loadAggregate(ctx, chatID)
	if err != nil {
		return err
	}

	// Применяем команду удаления
	if deleteErr := chatAggregate.Delete(deletedBy); deleteErr != nil {
		return fmt.Errorf("failed to delete chat: %w", deleteErr)
	}

	// Сохраняем события
	return s.saveAggregate(ctx, chatAggregate)
}

// loadAggregate загружает Chat агрегат из event store.
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

// saveAggregate сохраняет новые события агрегата.
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

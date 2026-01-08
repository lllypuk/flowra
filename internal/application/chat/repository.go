package chat

import (
	"context"
	"time"

	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// ReadModel represents the read model for chat (materialized view)
type ReadModel struct {
	ID            uuid.UUID
	WorkspaceID   uuid.UUID
	Type          chat.Type
	Title         string
	IsPublic      bool
	CreatedBy     uuid.UUID
	CreatedAt     time.Time
	LastMessageAt *time.Time
	MessageCount  int
	Participants  []chat.Participant
}

// Filters represents filters for searching chats
type Filters struct {
	Type     *chat.Type
	IsPublic *bool
	UserID   *uuid.UUID // participant
	Offset   int
	Limit    int
}

// CommandRepository defines the interface for commands (state changes) of chats
// Interface is declared on the consumer side (application layer)
// Uses Event Sourcing pattern
type CommandRepository interface {
	// Load loads Chat from event store by reconstructing state from events
	Load(ctx context.Context, chatID uuid.UUID) (*chat.Chat, error)

	// Save saves new Chat events in event store
	Save(ctx context.Context, c *chat.Chat) error

	// GetEvents returns all events of a chat
	GetEvents(ctx context.Context, chatID uuid.UUID) ([]event.DomainEvent, error)
}

// QueryRepository defines the interface for queries (read-only) of chats
// Interface is declared on the consumer side (application layer)
// Uses Read Model for fast queries
type QueryRepository interface {
	// FindByID finds a chat by ID (from read model)
	FindByID(ctx context.Context, chatID uuid.UUID) (*ReadModel, error)

	// FindByWorkspace finds workspace chats with filters
	FindByWorkspace(ctx context.Context, workspaceID uuid.UUID, filters Filters) ([]*ReadModel, error)

	// FindByParticipant finds user chats
	FindByParticipant(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*ReadModel, error)

	// Count returns the total number of chats in a workspace
	Count(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

// Repository combines Command and Query interfaces for convenience
// Used when use case needs both types of operations
type Repository interface {
	CommandRepository
	QueryRepository
}

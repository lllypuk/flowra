package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// QueryRepository - repository interface for query operations on chats
// Note: This interface is defined by the consumer (application layer) as per idiomatic Go patterns
// Implementation will be provided by the infrastructure layer (e.g., MongoDB read model)
type QueryRepository interface {
	// FindByWorkspace retrieves chats for a workspace with optional type filtering
	FindByWorkspace(
		ctx context.Context,
		workspaceID uuid.UUID,
		chatType *chat.Type,
		limit int,
		offset int,
	) ([]uuid.UUID, error)

	// CountByWorkspace returns total count of chats for a workspace with optional type filtering
	CountByWorkspace(
		ctx context.Context,
		workspaceID uuid.UUID,
		chatType *chat.Type,
	) (int, error)
}

// ListChatsUseCase - use case для получения списка чатов
type ListChatsUseCase struct {
	chatRepo   QueryRepository
	eventStore shared.EventStore
}

// NewListChatsUseCase - конструктор
func NewListChatsUseCase(chatRepo QueryRepository, eventStore shared.EventStore) *ListChatsUseCase {
	return &ListChatsUseCase{
		chatRepo:   chatRepo,
		eventStore: eventStore,
	}
}

// Execute - выполнить запрос
func (uc *ListChatsUseCase) Execute(ctx context.Context, query ListChatsQuery) (*ListChatsResult, error) {
	// 1. Validate input
	if err := uc.validate(query); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Validate pagination
	const (
		defaultLimit = 20
		maxLimit     = 100
	)

	limit := query.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	// 3. Find chat IDs from read model
	chatIDs, err := uc.chatRepo.FindByWorkspace(ctx, query.WorkspaceID, query.Type, limit+1, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find chats: %w", err)
	}

	// 4. Load chats from event store and filter by access
	accessibleChats := make([]Chat, 0, len(chatIDs))
	for _, chatID := range chatIDs {
		chatAggregate, loadErr := loadAggregate(ctx, uc.eventStore, chatID)
		if loadErr != nil {
			// Skip chats that can't be loaded (might be deleted or corrupted)
			continue
		}

		// Check access: public chats or where user is participant
		if !chatAggregate.IsPublic() && !chatAggregate.HasParticipant(query.RequestedBy) {
			continue
		}

		// Add to result
		accessibleChats = append(accessibleChats, *mapChatToDTO(chatAggregate))
	}

	// 5. Check if has more
	hasMore := len(chatIDs) > limit
	if hasMore && len(accessibleChats) > limit {
		accessibleChats = accessibleChats[:limit]
	}

	// 6. Count total (for pagination info)
	total, err := uc.chatRepo.CountByWorkspace(ctx, query.WorkspaceID, query.Type)
	if err != nil {
		total = len(accessibleChats) // fallback
	}

	return &ListChatsResult{
		Chats:   accessibleChats,
		Total:   total,
		HasMore: hasMore,
	}, nil
}

func (uc *ListChatsUseCase) validate(query ListChatsQuery) error {
	if err := shared.ValidateUUID("workspaceID", query.WorkspaceID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("requestedBy", query.RequestedBy); err != nil {
		return err
	}
	// Type filter is optional
	return nil
}

package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
)

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

	offset := max(query.Offset, 0)

	// 3. Find chats from read model
	filters := Filters{
		Type:   query.Type,
		Offset: offset,
		Limit:  limit + 1,
	}
	readModels, err := uc.chatRepo.FindByWorkspace(ctx, query.WorkspaceID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to find chats: %w", err)
	}

	// 4. Filter by access and convert to DTO
	accessibleChats := make([]Chat, 0, len(readModels))
	for _, rm := range readModels {
		// Check access: public chats or where user is participant
		if !rm.IsPublic {
			hasAccess := false
			for _, p := range rm.Participants {
				if p.UserID() == query.RequestedBy {
					hasAccess = true
					break
				}
			}
			if !hasAccess {
				continue
			}
		}

		// Convert read model to DTO
		accessibleChats = append(accessibleChats, Chat{
			ID:          rm.ID,
			WorkspaceID: rm.WorkspaceID,
			Type:        rm.Type,
			IsPublic:    rm.IsPublic,
			CreatedBy:   rm.CreatedBy,
			CreatedAt:   rm.CreatedAt,
		})
	}

	// 5. Check if has more
	hasMore := len(readModels) > limit
	if hasMore && len(accessibleChats) > limit {
		accessibleChats = accessibleChats[:limit]
	}

	// 6. Count total (for pagination info)
	total, err := uc.chatRepo.Count(ctx, query.WorkspaceID)
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

package chat

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// GetChatUseCase - use case для получения чата
type GetChatUseCase struct {
	eventStore shared.EventStore
}

// NewGetChatUseCase - конструктор
func NewGetChatUseCase(eventStore shared.EventStore) *GetChatUseCase {
	return &GetChatUseCase{
		eventStore: eventStore,
	}
}

// Execute - выполнить запрос
func (uc *GetChatUseCase) Execute(ctx context.Context, query GetChatQuery) (*GetChatResult, error) {
	// 1. Validate input
	if err := uc.validate(query); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Load chat from event store
	chatAggregate, err := loadAggregate(ctx, uc.eventStore, query.ChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to load chat: %w", err)
	}

	// 3. Check access permissions (public or participant)
	if !chatAggregate.IsPublic() && !chatAggregate.HasParticipant(query.RequestedBy) {
		return nil, errors.New("access denied: user is not a participant")
	}

	// 4. Build Chat DTO
	chatDTO := mapChatToDTO(chatAggregate)

	// 5. Calculate permissions
	permissions := calculatePermissions(chatAggregate, query.RequestedBy)

	return &GetChatResult{
		Chat:        chatDTO,
		Permissions: permissions,
	}, nil
}

func (uc *GetChatUseCase) validate(query GetChatQuery) error {
	if err := shared.ValidateUUID("chatID", query.ChatID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("requestedBy", query.RequestedBy); err != nil {
		return err
	}
	return nil
}

// mapChatToDTO - map domain Chat to DTO
func mapChatToDTO(chatAggregate *chat.Chat) *Chat {
	dto := &Chat{
		ID:           chatAggregate.ID(),
		WorkspaceID:  chatAggregate.WorkspaceID(),
		Type:         chatAggregate.Type(),
		Title:        chatAggregate.Title(),
		IsPublic:     chatAggregate.IsPublic(),
		CreatedBy:    chatAggregate.CreatedBy(),
		CreatedAt:    chatAggregate.CreatedAt(),
		Version:      chatAggregate.Version(),
		Participants: make([]Participant, 0),
	}

	// Add task-specific fields if applicable
	if chatAggregate.Type() == chat.TypeTask || chatAggregate.Type() == chat.TypeBug ||
		chatAggregate.Type() == chat.TypeEpic {
		status := chatAggregate.Status()
		dto.Status = &status

		if assignedTo := chatAggregate.AssigneeID(); assignedTo != nil {
			dto.AssignedTo = assignedTo
		}

		if priority := chatAggregate.Priority(); priority != "" {
			dto.Priority = &priority
		}

		if dueDate := chatAggregate.DueDate(); dueDate != nil {
			dto.DueDate = dueDate
		}
	}

	// Add bug-specific fields
	if chatAggregate.Type() == chat.TypeBug {
		if severity := chatAggregate.Severity(); severity != "" {
			dto.Severity = &severity
		}
	}

	// Map participants
	for _, p := range chatAggregate.Participants() {
		dto.Participants = append(dto.Participants, Participant{
			UserID:   p.UserID(),
			Role:     p.Role(),
			JoinedAt: p.JoinedAt(),
		})
	}

	return dto
}

// calculatePermissions - calculate user permissions for the chat
func calculatePermissions(chatAggregate *chat.Chat, userID uuid.UUID) Permissions {
	permissions := Permissions{}

	// Public chats: everyone can read
	if chatAggregate.IsPublic() {
		permissions.CanRead = true
	}

	// Participants can read and write
	if chatAggregate.HasParticipant(userID) {
		permissions.CanRead = true
		permissions.CanWrite = true
	}

	// Creator can manage
	if chatAggregate.CreatedBy() == userID {
		permissions.CanManage = true
	}

	// Check if user is admin role
	if chatAggregate.IsParticipantAdmin(userID) {
		permissions.CanManage = true
	}

	return permissions
}

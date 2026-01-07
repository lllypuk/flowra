package chat

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// debug logging - remove after debugging
func debugf(format string, args ...any) {
	fmt.Printf("[DEBUG GetChat] "+format+"\n", args...)
}

// GetChatUseCase - use case для получения чата
type GetChatUseCase struct {
	eventStore appcore.EventStore
}

// NewGetChatUseCase - конструктор
func NewGetChatUseCase(eventStore appcore.EventStore) *GetChatUseCase {
	return &GetChatUseCase{
		eventStore: eventStore,
	}
}

// Execute - выполнить запрос
func (uc *GetChatUseCase) Execute(ctx context.Context, query GetChatQuery) (*GetChatResult, error) {
	debugf("Execute: starting, chatID=%s, requestedBy=%s", query.ChatID, query.RequestedBy)

	// 1. Validate input
	if err := uc.validate(query); err != nil {
		debugf("Execute: validation failed: %v", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Load chat from event store
	debugf("Execute: loading aggregate")
	chatAggregate, err := loadAggregate(ctx, uc.eventStore, query.ChatID)
	if err != nil {
		debugf("Execute: failed to load aggregate: %v", err)
		return nil, fmt.Errorf("failed to load chat: %w", err)
	}

	debugf("Execute: aggregate loaded, id=%s, isPublic=%v, participantsCount=%d",
		chatAggregate.ID(), chatAggregate.IsPublic(), len(chatAggregate.Participants()))

	// Log participants
	for i, p := range chatAggregate.Participants() {
		debugf("Execute: participant[%d]: userID=%s, role=%s", i, p.UserID(), p.Role())
	}

	// 3. Check access permissions (public or participant)
	hasParticipant := chatAggregate.HasParticipant(query.RequestedBy)
	debugf("Execute: checking access - isPublic=%v, hasParticipant=%v", chatAggregate.IsPublic(), hasParticipant)

	if !chatAggregate.IsPublic() && !hasParticipant {
		debugf("Execute: access denied")
		return nil, errors.New("access denied: user is not a participant")
	}

	// 4. Build Chat DTO
	chatDTO := mapChatToDTO(chatAggregate)

	// 5. Calculate permissions
	permissions := calculatePermissions(chatAggregate, query.RequestedBy)

	debugf("Execute: success")
	return &GetChatResult{
		Chat:        chatDTO,
		Permissions: permissions,
	}, nil
}

func (uc *GetChatUseCase) validate(query GetChatQuery) error {
	if err := appcore.ValidateUUID("chatID", query.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("requestedBy", query.RequestedBy); err != nil {
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

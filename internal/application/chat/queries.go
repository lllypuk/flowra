package chat

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// ===== Query Definitions =====

// GetChatQuery - запрос на получение чата
type GetChatQuery struct {
	ChatID      uuid.UUID
	RequestedBy uuid.UUID // для проверки доступа
}

// ListChatsQuery - запрос на список чатов
type ListChatsQuery struct {
	WorkspaceID uuid.UUID
	Type        *chat.Type // optional filter
	Limit       int
	Offset      int
	RequestedBy uuid.UUID
}

// ListParticipantsQuery - запрос на список участников
type ListParticipantsQuery struct {
	ChatID      uuid.UUID
	RequestedBy uuid.UUID
}

// ===== Result Definitions =====

// GetChatResult - результат получения чата
type GetChatResult struct {
	Chat        *Chat
	Permissions Permissions // read/write/admin
}

// ListChatsResult - результат списка чатов
type ListChatsResult struct {
	Chats   []Chat `json:"chats"`
	Total   int    `json:"total"`
	HasMore bool   `json:"has_more"`
}

// ListParticipantsResult - результат списка участников
type ListParticipantsResult struct {
	Participants []Participant `json:"participants"`
}

// ===== DTOs =====

// Chat - Data Transfer Object для чата
type Chat struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Type        chat.Type `json:"type"`
	Title       string    `json:"title"`
	IsPublic    bool      `json:"is_public"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	Version     int       `json:"version"`

	// Task-specific fields (optional)
	Status     *string    `json:"status,omitempty"`
	AssignedTo *uuid.UUID `json:"assigned_to,omitempty"`
	Priority   *string    `json:"priority,omitempty"`
	DueDate    *time.Time `json:"due_date,omitempty"`

	// Bug-specific fields (optional)
	Severity *string `json:"severity,omitempty"`

	// Participants
	Participants []Participant `json:"participants"`
}

// Permissions - права пользователя на чат
type Permissions struct {
	CanRead   bool `json:"can_read"`
	CanWrite  bool `json:"can_write"`
	CanManage bool `json:"can_manage"` // admin rights
}

// Participant - участник чата
type Participant struct {
	UserID   uuid.UUID `json:"user_id"`
	Role     chat.Role `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

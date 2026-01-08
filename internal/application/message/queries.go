package message

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// GetMessageQuery - retrieval messages по ID
type GetMessageQuery struct {
	MessageID uuid.UUID
}

// ListMessagesQuery - list сообщений in чате
type ListMessagesQuery struct {
	ChatID uuid.UUID
	Limit  int        // default: 50, max: 100
	Offset int        // for offset-based pagination
	Before *time.Time // for cursor-based pagination
}

// GetThreadQuery - retrieval треда (responseов on message)
type GetThreadQuery struct {
	ParentMessageID uuid.UUID
}

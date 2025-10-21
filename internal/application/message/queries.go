package message

import (
	"time"

	"github.com/flowra/flowra/internal/domain/uuid"
)

// GetMessageQuery - получение сообщения по ID
type GetMessageQuery struct {
	MessageID uuid.UUID
}

// ListMessagesQuery - список сообщений в чате
type ListMessagesQuery struct {
	ChatID uuid.UUID
	Limit  int        // default: 50, max: 100
	Offset int        // для offset-based pagination
	Before *time.Time // для cursor-based pagination
}

// GetThreadQuery - получение треда (ответов на сообщение)
type GetThreadQuery struct {
	ParentMessageID uuid.UUID
}

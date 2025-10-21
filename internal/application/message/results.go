package message

import (
	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/message"
)

// Result представляет результат для одного сообщения
type Result = shared.Result[*message.Message]

// ListResult представляет результат для списка сообщений
type ListResult = shared.Result[[]*message.Message]

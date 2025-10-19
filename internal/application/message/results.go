package message

import (
	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/message"
)

// Result представляет результат для одного сообщения
type Result = shared.Result[*message.Message]

// ListResult представляет результат для списка сообщений
type ListResult = shared.Result[[]*message.Message]

package message

import (
	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/message"
)

// Result представляет результат для одного сообщения
type Result = appcore.Result[*message.Message]

// ListResult представляет результат для списка сообщений
type ListResult = appcore.Result[[]*message.Message]

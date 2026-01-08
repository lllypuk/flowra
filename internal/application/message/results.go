package message

import (
	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/message"
)

// Result represents result for одного messages
type Result = appcore.Result[*message.Message]

// ListResult represents result for list сообщений
type ListResult = appcore.Result[[]*message.Message]

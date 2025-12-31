package chat

import (
	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// Result представляет результат command UseCase с event sourcing
type Result = appcore.EventSourcedResult[*chat.Chat]

// QueryResult представляет результат query UseCase (без событий)
type QueryResult = appcore.Result[*chat.Chat]

// QueryResults представляет результат для списка чатов
type QueryResults = appcore.Result[[]*chat.Chat]

// ParticipantsResult представляет результат для списка участников
type ParticipantsResult = appcore.Result[[]chat.Participant]

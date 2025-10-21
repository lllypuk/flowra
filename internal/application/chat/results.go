package chat

import (
	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// Result представляет результат command UseCase с event sourcing
type Result = shared.EventSourcedResult[*chat.Chat]

// QueryResult представляет результат query UseCase (без событий)
type QueryResult = shared.Result[*chat.Chat]

// QueryResults представляет результат для списка чатов
type QueryResults = shared.Result[[]*chat.Chat]

// ParticipantsResult представляет результат для списка участников
type ParticipantsResult = shared.Result[[]chat.Participant]

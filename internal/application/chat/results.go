package chat

import (
	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// Result represents the result of a command UseCase with event sourcing
type Result = appcore.EventSourcedResult[*chat.Chat]

// QueryResult represents the result of a query UseCase (without events)
type QueryResult = appcore.Result[*chat.Chat]

// QueryResults represents the result for a list of chats
type QueryResults = appcore.Result[[]*chat.Chat]

// ParticipantsResult represents the result for a list of participants
type ParticipantsResult = appcore.Result[[]chat.Participant]

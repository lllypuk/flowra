package tag

import (
	chatApp "github.com/lllypuk/flowra/internal/application/chat"
	taskApp "github.com/lllypuk/flowra/internal/application/task"
)

// ChatUseCases grouping all Chat UseCases to simplify dependency injection
// This acts as a thin adapter for accessing Chat application layer from tag domain
type ChatUseCases struct {
	// Entity Creation
	ConvertToTask *chatApp.ConvertToTaskUseCase
	ConvertToBug  *chatApp.ConvertToBugUseCase
	ConvertToEpic *chatApp.ConvertToEpicUseCase

	// Task Read Model Creation (synchronous, to avoid relying on async event delivery)
	CreateTask *taskApp.CreateTaskUseCase

	// Entity Management
	ChangeStatus *chatApp.ChangeStatusUseCase
	AssignUser   *chatApp.AssignUserUseCase
	SetPriority  *chatApp.SetPriorityUseCase
	SetDueDate   *chatApp.SetDueDateUseCase
	Rename       *chatApp.RenameChatUseCase
	SetSeverity  *chatApp.SetSeverityUseCase

	// Participant Management (Task 007a)
	AddParticipant    *chatApp.AddParticipantUseCase
	RemoveParticipant *chatApp.RemoveParticipantUseCase

	// Chat Lifecycle (Task 007a)
	CloseChat  *chatApp.CloseChatUseCase
	ReopenChat *chatApp.ReopenChatUseCase
	// DeleteChat will be added when DeleteChatUseCase is implemented
}

package tag

import (
	chatApp "github.com/lllypuk/flowra/internal/application/chat"
)

// ChatUseCases grouping all Chat UseCases to simplify dependency injection
// This acts as a thin adapter for accessing Chat application layer from tag domain
type ChatUseCases struct {
	ConvertToTask  *chatApp.ConvertToTaskUseCase
	ConvertToBug   *chatApp.ConvertToBugUseCase
	ConvertToEpic  *chatApp.ConvertToEpicUseCase
	ChangeStatus   *chatApp.ChangeStatusUseCase
	AssignUser     *chatApp.AssignUserUseCase
	SetPriority    *chatApp.SetPriorityUseCase
	SetDueDate     *chatApp.SetDueDateUseCase
	Rename         *chatApp.RenameChatUseCase
	SetSeverity    *chatApp.SetSeverityUseCase
}

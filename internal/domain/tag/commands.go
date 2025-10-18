package tag

import "github.com/google/uuid"

// Command представляет команду, которая должна быть выполнена
type Command interface {
	CommandType() string
}

// CreateTaskCommand - команда создания Task
type CreateTaskCommand struct {
	ChatID uuid.UUID
	Title  string
}

// CommandType возвращает тип команды
func (c CreateTaskCommand) CommandType() string {
	return "CreateTask"
}

// CreateBugCommand - команда создания Bug
type CreateBugCommand struct {
	ChatID uuid.UUID
	Title  string
}

// CommandType возвращает тип команды
func (c CreateBugCommand) CommandType() string {
	return "CreateBug"
}

// CreateEpicCommand - команда создания Epic
type CreateEpicCommand struct {
	ChatID uuid.UUID
	Title  string
}

// CommandType возвращает тип команды
func (c CreateEpicCommand) CommandType() string {
	return "CreateEpic"
}

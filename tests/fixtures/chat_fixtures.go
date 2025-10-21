package fixtures

import (
	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/domain/chat"
	domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

// CreateChatCommandBuilder создает builder для CreateChatCommand
type CreateChatCommandBuilder struct {
	cmd chatapp.CreateChatCommand
}

// NewCreateChatCommandBuilder создает новый builder
func NewCreateChatCommandBuilder() *CreateChatCommandBuilder {
	return &CreateChatCommandBuilder{
		cmd: chatapp.CreateChatCommand{
			WorkspaceID: domainUUID.NewUUID(),
			Title:       "Test Chat",
			Type:        chat.TypeDiscussion,
			CreatedBy:   domainUUID.NewUUID(),
		},
	}
}

// WithWorkspace устанавливает workspace ID (accepts domain UUID)
func (b *CreateChatCommandBuilder) WithWorkspace(id domainUUID.UUID) *CreateChatCommandBuilder {
	b.cmd.WorkspaceID = id
	return b
}

// WithTitle устанавливает title
func (b *CreateChatCommandBuilder) WithTitle(title string) *CreateChatCommandBuilder {
	b.cmd.Title = title
	return b
}

// AsTask устанавливает тип как Task
func (b *CreateChatCommandBuilder) AsTask() *CreateChatCommandBuilder {
	b.cmd.Type = chat.TypeTask
	return b
}

// AsBug устанавливает тип как Bug
func (b *CreateChatCommandBuilder) AsBug() *CreateChatCommandBuilder {
	b.cmd.Type = chat.TypeBug
	return b
}

// AsEpic устанавливает тип как Epic
func (b *CreateChatCommandBuilder) AsEpic() *CreateChatCommandBuilder {
	b.cmd.Type = chat.TypeEpic
	return b
}

// CreatedBy устанавливает creator ID (accepts domain UUID)
func (b *CreateChatCommandBuilder) CreatedBy(userID domainUUID.UUID) *CreateChatCommandBuilder {
	b.cmd.CreatedBy = userID
	return b
}

// Build возвращает готовую команду
func (b *CreateChatCommandBuilder) Build() chatapp.CreateChatCommand {
	return b.cmd
}

// AddParticipantCommandBuilder создает builder для AddParticipantCommand
type AddParticipantCommandBuilder struct {
	cmd chatapp.AddParticipantCommand
}

// NewAddParticipantCommandBuilder создает новый builder (accepts domain UUID)
func NewAddParticipantCommandBuilder(chatID domainUUID.UUID, userID domainUUID.UUID) *AddParticipantCommandBuilder {
	return &AddParticipantCommandBuilder{
		cmd: chatapp.AddParticipantCommand{
			ChatID:  chatID,
			UserID:  userID,
			Role:    chat.RoleMember,
			AddedBy: domainUUID.NewUUID(),
		},
	}
}

// WithRole устанавливает роль
func (b *AddParticipantCommandBuilder) WithRole(role chat.Role) *AddParticipantCommandBuilder {
	b.cmd.Role = role
	return b
}

// AddedBy устанавливает ID пользователя, добавившего участника (accepts domain UUID)
func (b *AddParticipantCommandBuilder) AddedBy(userID domainUUID.UUID) *AddParticipantCommandBuilder {
	b.cmd.AddedBy = userID
	return b
}

// Build возвращает готовую команду
func (b *AddParticipantCommandBuilder) Build() chatapp.AddParticipantCommand {
	return b.cmd
}

// ConvertToTaskCommandBuilder создает builder для ConvertToTaskCommand
type ConvertToTaskCommandBuilder struct {
	cmd chatapp.ConvertToTaskCommand
}

// NewConvertToTaskCommandBuilder создает новый builder (accepts domain UUID)
func NewConvertToTaskCommandBuilder(chatID domainUUID.UUID) *ConvertToTaskCommandBuilder {
	return &ConvertToTaskCommandBuilder{
		cmd: chatapp.ConvertToTaskCommand{
			ChatID:      chatID,
			Title:       "New Task",
			ConvertedBy: domainUUID.NewUUID(),
		},
	}
}

// WithTitle устанавливает title
func (b *ConvertToTaskCommandBuilder) WithTitle(title string) *ConvertToTaskCommandBuilder {
	b.cmd.Title = title
	return b
}

// ConvertedBy устанавливает ID пользователя, конвертировавшего чат (accepts domain UUID)
func (b *ConvertToTaskCommandBuilder) ConvertedBy(userID domainUUID.UUID) *ConvertToTaskCommandBuilder {
	b.cmd.ConvertedBy = userID
	return b
}

// Build возвращает готовую команду
func (b *ConvertToTaskCommandBuilder) Build() chatapp.ConvertToTaskCommand {
	return b.cmd
}

// ChangeStatusCommandBuilder создает builder для ChangeStatusCommand
type ChangeStatusCommandBuilder struct {
	cmd chatapp.ChangeStatusCommand
}

// NewChangeStatusCommandBuilder создает новый builder (accepts domain UUID)
func NewChangeStatusCommandBuilder(chatID domainUUID.UUID) *ChangeStatusCommandBuilder {
	return &ChangeStatusCommandBuilder{
		cmd: chatapp.ChangeStatusCommand{
			ChatID:    chatID,
			Status:    "In Progress",
			ChangedBy: domainUUID.NewUUID(),
		},
	}
}

// WithStatus устанавливает статус
func (b *ChangeStatusCommandBuilder) WithStatus(status string) *ChangeStatusCommandBuilder {
	b.cmd.Status = status
	return b
}

// ChangedBy устанавливает ID пользователя, изменившего статус (accepts domain UUID)
func (b *ChangeStatusCommandBuilder) ChangedBy(userID domainUUID.UUID) *ChangeStatusCommandBuilder {
	b.cmd.ChangedBy = userID
	return b
}

// Build возвращает готовую команду
func (b *ChangeStatusCommandBuilder) Build() chatapp.ChangeStatusCommand {
	return b.cmd
}

// AssignUserCommandBuilder создает builder для AssignUserCommand
type AssignUserCommandBuilder struct {
	cmd chatapp.AssignUserCommand
}

// NewAssignUserCommandBuilder создает новый builder (accepts domain UUID)
func NewAssignUserCommandBuilder(chatID domainUUID.UUID) *AssignUserCommandBuilder {
	return &AssignUserCommandBuilder{
		cmd: chatapp.AssignUserCommand{
			ChatID:     chatID,
			AssigneeID: nil,
			AssignedBy: domainUUID.NewUUID(),
		},
	}
}

// AssignTo устанавливает ID assignee (accepts domain UUID)
func (b *AssignUserCommandBuilder) AssignTo(userID domainUUID.UUID) *AssignUserCommandBuilder {
	b.cmd.AssigneeID = &userID
	return b
}

// AssignedBy устанавливает ID пользователя, который назначил (accepts domain UUID)
func (b *AssignUserCommandBuilder) AssignedBy(userID domainUUID.UUID) *AssignUserCommandBuilder {
	b.cmd.AssignedBy = userID
	return b
}

// Build возвращает готовую команду
func (b *AssignUserCommandBuilder) Build() chatapp.AssignUserCommand {
	return b.cmd
}

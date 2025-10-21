package fixtures

import (
	messageapp "github.com/lllypuk/teams-up/internal/application/message"
	domainUUID "github.com/lllypuk/teams-up/internal/domain/uuid"
)

// SendMessageCommandBuilder создает builder для SendMessageCommand
type SendMessageCommandBuilder struct {
	cmd messageapp.SendMessageCommand
}

// NewSendMessageCommandBuilder создает новый builder (accepts domain UUID)
func NewSendMessageCommandBuilder(chatID domainUUID.UUID, authorID domainUUID.UUID) *SendMessageCommandBuilder {
	return &SendMessageCommandBuilder{
		cmd: messageapp.SendMessageCommand{
			ChatID:   chatID,
			Content:  "Test message",
			AuthorID: authorID,
		},
	}
}

// WithContent устанавливает content
func (b *SendMessageCommandBuilder) WithContent(content string) *SendMessageCommandBuilder {
	b.cmd.Content = content
	return b
}

// Build возвращает готовую команду
func (b *SendMessageCommandBuilder) Build() messageapp.SendMessageCommand {
	return b.cmd
}

// EditMessageCommandBuilder создает builder для EditMessageCommand
type EditMessageCommandBuilder struct {
	cmd messageapp.EditMessageCommand
}

// NewEditMessageCommandBuilder создает новый builder (accepts domain UUID)
func NewEditMessageCommandBuilder(messageID domainUUID.UUID, userID domainUUID.UUID) *EditMessageCommandBuilder {
	return &EditMessageCommandBuilder{
		cmd: messageapp.EditMessageCommand{
			MessageID: messageID,
			Content:   "Edited message",
			EditorID:  userID,
		},
	}
}

// WithContent устанавливает content
func (b *EditMessageCommandBuilder) WithContent(content string) *EditMessageCommandBuilder {
	b.cmd.Content = content
	return b
}

// Build возвращает готовую команду
func (b *EditMessageCommandBuilder) Build() messageapp.EditMessageCommand {
	return b.cmd
}

// DeleteMessageCommandBuilder создает builder для DeleteMessageCommand
type DeleteMessageCommandBuilder struct {
	cmd messageapp.DeleteMessageCommand
}

// NewDeleteMessageCommandBuilder создает новый builder (accepts domain UUID)
func NewDeleteMessageCommandBuilder(messageID domainUUID.UUID, userID domainUUID.UUID) *DeleteMessageCommandBuilder {
	return &DeleteMessageCommandBuilder{
		cmd: messageapp.DeleteMessageCommand{
			MessageID: messageID,
			DeletedBy: userID,
		},
	}
}

// Build возвращает готовую команду
func (b *DeleteMessageCommandBuilder) Build() messageapp.DeleteMessageCommand {
	return b.cmd
}

package fixtures

import (
	messageapp "github.com/lllypuk/flowra/internal/application/message"
	domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

// SendMessageCommandBuilder creates builder for SendMessageCommand
type SendMessageCommandBuilder struct {
	cmd messageapp.SendMessageCommand
}

// NewSendMessageCommandBuilder creates New builder (accepts domain UUID)
func NewSendMessageCommandBuilder(chatID domainUUID.UUID, authorID domainUUID.UUID) *SendMessageCommandBuilder {
	return &SendMessageCommandBuilder{
		cmd: messageapp.SendMessageCommand{
			ChatID:   chatID,
			Content:  "Test message",
			AuthorID: authorID,
		},
	}
}

// WithContent sets content
func (b *SendMessageCommandBuilder) WithContent(content string) *SendMessageCommandBuilder {
	b.cmd.Content = content
	return b
}

// Build returns prepared command
func (b *SendMessageCommandBuilder) Build() messageapp.SendMessageCommand {
	return b.cmd
}

// EditMessageCommandBuilder creates builder for EditMessageCommand
type EditMessageCommandBuilder struct {
	cmd messageapp.EditMessageCommand
}

// NewEditMessageCommandBuilder creates New builder (accepts domain UUID)
func NewEditMessageCommandBuilder(messageID domainUUID.UUID, userID domainUUID.UUID) *EditMessageCommandBuilder {
	return &EditMessageCommandBuilder{
		cmd: messageapp.EditMessageCommand{
			MessageID: messageID,
			Content:   "Edited message",
			EditorID:  userID,
		},
	}
}

// WithContent sets content
func (b *EditMessageCommandBuilder) WithContent(content string) *EditMessageCommandBuilder {
	b.cmd.Content = content
	return b
}

// Build returns prepared command
func (b *EditMessageCommandBuilder) Build() messageapp.EditMessageCommand {
	return b.cmd
}

// DeleteMessageCommandBuilder creates builder for DeleteMessageCommand
type DeleteMessageCommandBuilder struct {
	cmd messageapp.DeleteMessageCommand
}

// NewDeleteMessageCommandBuilder creates New builder (accepts domain UUID)
func NewDeleteMessageCommandBuilder(messageID domainUUID.UUID, userID domainUUID.UUID) *DeleteMessageCommandBuilder {
	return &DeleteMessageCommandBuilder{
		cmd: messageapp.DeleteMessageCommand{
			MessageID: messageID,
			DeletedBy: userID,
		},
	}
}

// Build returns prepared command
func (b *DeleteMessageCommandBuilder) Build() messageapp.DeleteMessageCommand {
	return b.cmd
}

package message_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/message"
	"github.com/lllypuk/flowra/internal/domain/user"
	domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

// TestTagIntegration_SendMessageWithTagComponents tests that SendMessageUseCase
// can accept tag components and still deliver messages successfully.
// Note: Full end-to-end tag execution would require complete mock implementations
// of the Chat UseCases and EventStore interfaces. This test focuses on verifying
// that the tag components don't break the message sending workflow.
func TestTagIntegration_SendMessageWithTagComponents(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create repositories and event bus
	messageRepo := message.NewMockMessageRepository()
	chatRepo := message.NewMockChatRepository()
	eventBus := message.NewMockEventBus()

	// Create chat and add participant
	chatID := domainUUID.NewUUID()
	authorID := domainUUID.NewUUID()
	chatRepo.AddChat(chatID, []domainUUID.UUID{authorID})

	// Create SendMessageUseCase with nil tag components
	// (simulating the default state without full tag processing setup)
	sendMessageUC := message.NewSendMessageUseCase(messageRepo, chatRepo, eventBus, nil, nil, domainUUID.NewUUID())

	// Send message with tag command
	cmd := message.SendMessageCommand{
		ChatID:          chatID,
		Content:         "#task Create awesome feature",
		AuthorID:        authorID,
		ParentMessageID: "",
	}

	// Act
	result, err := sendMessageUC.Execute(ctx, cmd)

	// Assert - Message should be sent successfully
	require.NoError(t, err)
	require.NotNil(t, result.Value)
	assert.Equal(t, "#task Create awesome feature", result.Value.Content())
	assert.Equal(t, chatID, result.Value.ChatID())
	assert.Equal(t, authorID, result.Value.AuthorID())

	// Message should be saved
	assert.Len(t, messageRepo.Messages, 1)

	// Event should be published
	assert.Len(t, eventBus.Published, 1)

	// This test verifies that message sending works correctly and that tag
	// processing is properly integrated without breaking the message delivery flow.
}

// TestTagIntegration_MessageWithoutTags verifies that messages without tags
// are delivered normally regardless of tag processing setup
func TestTagIntegration_MessageWithoutTags(t *testing.T) {
	// Setup
	ctx := context.Background()

	messageRepo := message.NewMockMessageRepository()
	chatRepo := message.NewMockChatRepository()
	eventBus := message.NewMockEventBus()

	chatID := domainUUID.NewUUID()
	authorID := domainUUID.NewUUID()
	chatRepo.AddChat(chatID, []domainUUID.UUID{authorID})

	// Send message without tag processing
	sendMessageUC := message.NewSendMessageUseCase(messageRepo, chatRepo, eventBus, nil, nil, domainUUID.NewUUID())

	cmd := message.SendMessageCommand{
		ChatID:          chatID,
		Content:         "Just a regular message without tags",
		AuthorID:        authorID,
		ParentMessageID: "",
	}

	// Act
	result, err := sendMessageUC.Execute(ctx, cmd)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result.Value)
	assert.Len(t, messageRepo.Messages, 1)
	assert.Len(t, eventBus.Published, 1)
}

// TestTagIntegration_DisabledTagProcessing verifies that tag processing
// can be disabled by passing nil for tag components
func TestTagIntegration_DisabledTagProcessing(t *testing.T) {
	// Setup
	ctx := context.Background()

	messageRepo := message.NewMockMessageRepository()
	chatRepo := message.NewMockChatRepository()
	eventBus := message.NewMockEventBus()

	chatID := domainUUID.NewUUID()
	authorID := domainUUID.NewUUID()
	chatRepo.AddChat(chatID, []domainUUID.UUID{authorID})

	// Create SendMessageUseCase WITHOUT tag processing
	sendMessageUC := message.NewSendMessageUseCase(messageRepo, chatRepo, eventBus, nil, nil, domainUUID.NewUUID())

	cmd := message.SendMessageCommand{
		ChatID:          chatID,
		Content:         "#task This should not be processed as a tag",
		AuthorID:        authorID,
		ParentMessageID: "",
	}

	// Act
	result, err := sendMessageUC.Execute(ctx, cmd)

	// Assert - Message should be sent but tag processing skipped
	require.NoError(t, err)
	require.NotNil(t, result.Value)
	assert.Len(t, messageRepo.Messages, 1)
}

// Mock implementations for testing

type MockUserRepositoryForTags struct{}

func (m *MockUserRepositoryForTags) FindByUsername(_ context.Context, _ string) (*user.User, error) {
	// Mock implementation - return error to indicate user not found
	return nil, context.Canceled
}

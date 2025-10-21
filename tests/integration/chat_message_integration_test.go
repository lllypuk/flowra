//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatapp "github.com/flowra/flowra/internal/application/chat"
	messageapp "github.com/flowra/flowra/internal/application/message"
	"github.com/flowra/flowra/internal/domain/chat"
	"github.com/flowra/flowra/internal/domain/uuid"
	"github.com/flowra/flowra/tests/fixtures"
	"github.com/flowra/flowra/tests/testutil"
)

// TestChatMessageIntegration_SendMessage_UpdatesChatMessageCount проверяет, что отправка сообщения обновляет счетчик
func TestChatMessageIntegration_SendMessage_UpdatesChatMessageCount(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	creator := uuid.NewUUID()

	// Create chat
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(creator).
		Build()

	chatResult, err := suite.CreateChat.Execute(ctx, createChatCmd)
	testutil.AssertNoError(t, err)
	chatID := chatResult.Value.ID().ToGoogleUUID()

	// Act: Send message
	sendMsgCmd := fixtures.NewSendMessageCommandBuilder(chatID, creator).
		WithContent("Hello World").
		Build()

	msgResult, err := suite.SendMessage.Execute(ctx, sendMsgCmd)

	// Assert
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, msgResult)
	testutil.AssertEqual(t, "Hello World", msgResult.Value.Content)

	// Verify message was saved
	allMessages := suite.MessageRepo.GetAll()
	testutil.AssertLen(t, allMessages, 1)
}

// TestChatMessageIntegration_SendMessage_AsNonParticipant_Fails проверяет, что non-participant не может отправлять сообщения
func TestChatMessageIntegration_SendMessage_AsNonParticipant_Fails(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	creator := uuid.NewUUID()
	nonParticipant := uuid.NewUUID()

	// Create chat
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(creator).
		Build()

	chatResult, err := suite.CreateChat.Execute(ctx, createChatCmd)
	testutil.AssertNoError(t, err)
	chatID := chatResult.Value.ID().ToGoogleUUID()

	// Act: Non-participant tries to send message
	sendMsgCmd := fixtures.NewSendMessageCommandBuilder(chatID, nonParticipant).
		WithContent("Unauthorized message").
		Build()

	msgResult, err := suite.SendMessage.Execute(ctx, sendMsgCmd)

	// Assert
	testutil.AssertError(t, err, "should fail for non-participant")
	testutil.AssertNil(t, msgResult)
}

// TestChatMessageIntegration_AddParticipant_CanSendMessage проверяет, что добавленный участник может отправлять сообщения
func TestChatMessageIntegration_AddParticipant_CanSendMessage(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	creator := uuid.NewUUID()
	newParticipant := uuid.NewUUID()

	// Create chat
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(creator).
		Build()

	chatResult, err := suite.CreateChat.Execute(ctx, createChatCmd)
	testutil.AssertNoError(t, err)
	chatID := chatResult.Value.ID().ToGoogleUUID()

	// Act: Add participant
	addParticipantCmd := fixtures.NewAddParticipantCommandBuilder(chatID, newParticipant.ToGoogleUUID()).
		AddedBy(creator).
		Build()

	_, err = suite.AddParticipant.Execute(ctx, addParticipantCmd)
	testutil.AssertNoError(t, err)

	// Act: New participant sends message
	sendMsgCmd := fixtures.NewSendMessageCommandBuilder(chatID, newParticipant).
		WithContent("Hello from new participant").
		Build()

	msgResult, err := suite.SendMessage.Execute(ctx, sendMsgCmd)

	// Assert
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, msgResult)

	// Verify message was saved
	allMessages := suite.MessageRepo.GetAll()
	testutil.AssertLen(t, allMessages, 1)
}

// TestChatMessageIntegration_MultipleMessages_MaintainOrder проверяет порядок сообщений
func TestChatMessageIntegration_MultipleMessages_MaintainOrder(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	creator := uuid.NewUUID()

	// Create chat
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(creator).
		Build()

	chatResult, err := suite.CreateChat.Execute(ctx, createChatCmd)
	testutil.AssertNoError(t, err)
	chatID := chatResult.Value.ID().ToGoogleUUID()

	// Act: Send multiple messages
	messages := []string{"First", "Second", "Third"}
	for _, content := range messages {
		cmd := fixtures.NewSendMessageCommandBuilder(chatID, creator).
			WithContent(content).
			Build()

		_, err := suite.SendMessage.Execute(ctx, cmd)
		testutil.AssertNoError(t, err)
	}

	// Assert: Verify all messages saved
	allMessages := suite.MessageRepo.GetAll()
	testutil.AssertLen(t, allMessages, len(messages))

	// Verify content order
	for i, msg := range allMessages {
		testutil.AssertEqual(t, messages[i], msg.Content)
	}
}

// TestChatMessageIntegration_Message_PublishesEvent проверяет, что сообщение публикует событие
func TestChatMessageIntegration_Message_PublishesEvent(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	creator := uuid.NewUUID()

	// Create chat
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(creator).
		Build()

	chatResult, err := suite.CreateChat.Execute(ctx, createChatCmd)
	testutil.AssertNoError(t, err)
	chatID := chatResult.Value.ID().ToGoogleUUID()

	// Reset event bus to track only message events
	suite.EventBus.Reset()

	// Act: Send message
	sendMsgCmd := fixtures.NewSendMessageCommandBuilder(chatID, creator).
		WithContent("Test message").
		Build()

	_, err = suite.SendMessage.Execute(ctx, sendMsgCmd)
	testutil.AssertNoError(t, err)

	// Assert: Check if message event was published
	publishedEvents := suite.EventBus.PublishedEvents()
	testutil.AssertGreaterOrEqual(t, len(publishedEvents), 1)

	// Verify at least one message-related event
	hasMessageEvent := false
	for _, evt := range publishedEvents {
		if evt.AggregateType() == "Message" {
			hasMessageEvent = true
			break
		}
	}
	testutil.AssertTrue(t, hasMessageEvent, "should publish message event")
}

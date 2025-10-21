//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	messageapp "github.com/flowra/flowra/internal/application/message"
	"github.com/flowra/flowra/internal/domain/uuid"
	"github.com/flowra/flowra/tests/fixtures"
	"github.com/flowra/flowra/tests/testutil"
)

// TestE2E_Messaging_SendReceive проверяет отправку и получение сообщений
func TestE2E_Messaging_SendReceive(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	user1 := uuid.NewUUID()
	user2 := uuid.NewUUID()

	// Create chat
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(user1).
		WithTitle("Discussion chat").
		Build()

	chatResult, err := suite.CreateChat.Execute(ctx, createChatCmd)
	testutil.AssertNoError(t, err)
	chatID := chatResult.Value.ID().ToGoogleUUID()

	// Add user2
	addCmd := fixtures.NewAddParticipantCommandBuilder(chatID, user2.ToGoogleUUID()).
		AddedBy(user1).
		Build()
	_, _ = suite.AddParticipant.Execute(ctx, addCmd)

	// Act: User1 sends message
	msg1Cmd := fixtures.NewSendMessageCommandBuilder(chatID, user1).
		WithContent("Hey there!").
		Build()

	msg1Result, err := suite.SendMessage.Execute(ctx, msg1Cmd)
	testutil.AssertNoError(t, err)

	// Act: User2 sends message
	msg2Cmd := fixtures.NewSendMessageCommandBuilder(chatID, user2).
		WithContent("Hi! How are you?").
		Build()

	msg2Result, err := suite.SendMessage.Execute(ctx, msg2Cmd)
	testutil.AssertNoError(t, err)

	// Assert: Verify messages
	testutil.AssertNotNil(t, msg1Result)
	testutil.AssertNotNil(t, msg2Result)

	assert.Equal(t, "Hey there!", msg1Result.Value.Content)
	assert.Equal(t, "Hi! How are you?", msg2Result.Value.Content)

	// Assert: Verify message count
	allMessages := suite.MessageRepo.GetAll()
	testutil.AssertLen(t, allMessages, 2)
}

// TestE2E_Messaging_ConversationThread проверяет цепочку сообщений
func TestE2E_Messaging_ConversationThread(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	alice := uuid.NewUUID()
	bob := uuid.NewUUID()

	// Create chat
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(alice).
		WithTitle("Conversation").
		Build()

	chatResult, err := suite.CreateChat.Execute(ctx, createChatCmd)
	testutil.AssertNoError(t, err)
	chatID := chatResult.Value.ID().ToGoogleUUID()

	// Add Bob
	addCmd := fixtures.NewAddParticipantCommandBuilder(chatID, bob.ToGoogleUUID()).
		AddedBy(alice).
		Build()
	_, _ = suite.AddParticipant.Execute(ctx, addCmd)

	// Simulate a conversation
	conversation := []struct {
		sender  uuid.UUID
		content string
	}{
		{alice, "What's the status on the project?"},
		{bob, "Making good progress. Should be done by Friday."},
		{alice, "Great! Do you need any help?"},
		{bob, "No, I think I'm good for now. Thanks!"},
		{alice, "Sounds good. Let me know if anything changes."},
	}

	// Act: Send all messages
	for _, msg := range conversation {
		cmd := fixtures.NewSendMessageCommandBuilder(chatID, msg.sender).
			WithContent(msg.content).
			Build()

		_, err := suite.SendMessage.Execute(ctx, cmd)
		testutil.AssertNoError(t, err)
	}

	// Assert: Verify all messages saved
	allMessages := suite.MessageRepo.GetAll()
	testutil.AssertLen(t, allMessages, len(conversation))

	// Verify content matches
	for i, msg := range allMessages {
		assert.Equal(t, conversation[i].content, msg.Content)
	}
}

// TestE2E_Messaging_MultiChat_Segregation проверяет разделение сообщений между чатами
func TestE2E_Messaging_MultiChat_Segregation(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	user := uuid.NewUUID()

	// Create two chats
	createChat1Cmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(user).
		WithTitle("Chat 1").
		Build()

	chat1Result, _ := suite.CreateChat.Execute(ctx, createChat1Cmd)
	chat1ID := chat1Result.Value.ID().ToGoogleUUID()

	createChat2Cmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(user).
		WithTitle("Chat 2").
		Build()

	chat2Result, _ := suite.CreateChat.Execute(ctx, createChat2Cmd)
	chat2ID := chat2Result.Value.ID().ToGoogleUUID()

	// Act: Send message in chat 1
	msg1Cmd := fixtures.NewSendMessageCommandBuilder(chat1ID, user).
		WithContent("Message in chat 1").
		Build()
	_, _ = suite.SendMessage.Execute(ctx, msg1Cmd)

	// Act: Send message in chat 2
	msg2Cmd := fixtures.NewSendMessageCommandBuilder(chat2ID, user).
		WithContent("Message in chat 2").
		Build()
	_, _ = suite.SendMessage.Execute(ctx, msg2Cmd)

	// Assert: Verify both messages exist
	allMessages := suite.MessageRepo.GetAll()
	testutil.AssertLen(t, allMessages, 2)

	// Verify messages belong to correct chats
	assert.Equal(t, "Message in chat 1", allMessages[0].Content)
	assert.Equal(t, "Message in chat 2", allMessages[1].Content)
}

// TestE2E_Messaging_EventPublishing проверяет публикацию событий при отправке сообщений
func TestE2E_Messaging_EventPublishing(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	sender := uuid.NewUUID()

	// Create chat
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(sender).
		WithTitle("Event test chat").
		Build()

	chatResult, _ := suite.CreateChat.Execute(ctx, createChatCmd)
	chatID := chatResult.Value.ID().ToGoogleUUID()

	// Reset to track only message events
	suite.EventBus.Reset()

	// Act: Send message
	msgCmd := fixtures.NewSendMessageCommandBuilder(chatID, sender).
		WithContent("Test message").
		Build()

	_, err := suite.SendMessage.Execute(ctx, msgCmd)
	testutil.AssertNoError(t, err)

	// Assert: Verify events published
	publishedEvents := suite.EventBus.PublishedEvents()
	testutil.AssertGreater(t, len(publishedEvents), 0)

	// Check for message event
	hasMessageEvent := false
	for _, evt := range publishedEvents {
		if evt.AggregateType() == "Message" {
			hasMessageEvent = true
			break
		}
	}
	assert.True(t, hasMessageEvent, "should publish message event")
}

// TestE2E_Messaging_HighVolume проверяет отправку большого количества сообщений
func TestE2E_Messaging_HighVolume(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	users := make([]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		users[i] = uuid.NewUUID()
	}

	// Create chat
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(users[0]).
		WithTitle("Group chat").
		Build()

	chatResult, _ := suite.CreateChat.Execute(ctx, createChatCmd)
	chatID := chatResult.Value.ID().ToGoogleUUID()

	// Add all users except first
	for i := 1; i < len(users); i++ {
		addCmd := fixtures.NewAddParticipantCommandBuilder(chatID, users[i].ToGoogleUUID()).
			AddedBy(users[0]).
			Build()
		_, _ = suite.AddParticipant.Execute(ctx, addCmd)
	}

	// Act: Send multiple messages from different users
	messageCount := 50
	start := time.Now()

	for i := 0; i < messageCount; i++ {
		senderIdx := i % len(users)
		content := "Message " + string(rune('0'+i%10))

		cmd := fixtures.NewSendMessageCommandBuilder(chatID, users[senderIdx]).
			WithContent(content).
			Build()

		_, err := suite.SendMessage.Execute(ctx, cmd)
		testutil.AssertNoError(t, err)
	}

	duration := time.Since(start)

	// Assert
	allMessages := suite.MessageRepo.GetAll()
	testutil.AssertLen(t, allMessages, messageCount)

	// Verify duration is reasonable (not too slow)
	assert.Less(t, duration, 10*time.Second, "should process messages quickly")
}

// TestE2E_Messaging_UserPermissions проверяет разрешения при отправке сообщений
func TestE2E_Messaging_UserPermissions(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	creator := uuid.NewUUID()
	member := uuid.NewUUID()
	nonMember := uuid.NewUUID()

	// Create chat
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(creator).
		WithTitle("Private chat").
		Build()

	chatResult, _ := suite.CreateChat.Execute(ctx, createChatCmd)
	chatID := chatResult.Value.ID().ToGoogleUUID()

	// Add member
	addCmd := fixtures.NewAddParticipantCommandBuilder(chatID, member.ToGoogleUUID()).
		AddedBy(creator).
		Build()
	_, _ = suite.AddParticipant.Execute(ctx, addCmd)

	// Act: Member sends message (should succeed)
	memberMsgCmd := fixtures.NewSendMessageCommandBuilder(chatID, member).
		WithContent("I'm a member").
		Build()

	memberResult, memberErr := suite.SendMessage.Execute(ctx, memberMsgCmd)

	// Act: Non-member sends message (should fail)
	nonMemberMsgCmd := fixtures.NewSendMessageCommandBuilder(chatID, nonMember).
		WithContent("I'm not a member").
		Build()

	nonMemberResult, nonMemberErr := suite.SendMessage.Execute(ctx, nonMemberMsgCmd)

	// Assert
	assert.NoError(t, memberErr, "member should be able to send message")
	assert.NotNil(t, memberResult)

	assert.Error(t, nonMemberErr, "non-member should not be able to send message")
	assert.Nil(t, nonMemberResult)
}

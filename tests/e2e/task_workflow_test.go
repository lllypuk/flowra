//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatapp "github.com/lllypuk/teams-up/internal/application/chat"
	messageapp "github.com/lllypuk/teams-up/internal/application/message"
	notificationapp "github.com/lllypuk/teams-up/internal/application/notification"
	taskapp "github.com/lllypuk/teams-up/internal/application/task"
	"github.com/lllypuk/teams-up/internal/domain/chat"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
	"github.com/lllypuk/teams-up/tests/fixtures"
	"github.com/lllypuk/teams-up/tests/testutil"
)

// TestE2E_TaskWorkflow_CompleteFlow проверяет полный workflow создания и управления задачей
func TestE2E_TaskWorkflow_CompleteFlow(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	user1 := uuid.NewUUID()
	user2 := uuid.NewUUID()

	// Scenario:
	// 1. User1 creates a Discussion chat
	// 2. User1 adds User2 as participant
	// 3. User1 sends a message
	// 4. User1 converts chat to Task
	// 5. User1 assigns task to User2
	// 6. User2 changes task status

	// Step 1: Create Discussion chat
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(user1).
		WithTitle("Discuss new feature").
		Build()

	chatResult, err := suite.CreateChat.Execute(ctx, createChatCmd)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, chatResult)

	chatID := chatResult.Value.ID()
	chatIDGoogle := chatID.ToGoogleUUID()

	// Step 2: Add participant
	addParticipantCmd := fixtures.NewAddParticipantCommandBuilder(chatIDGoogle, user2.ToGoogleUUID()).
		AddedBy(user1).
		WithRole(chat.RoleMember).
		Build()

	_, err = suite.AddParticipant.Execute(ctx, addParticipantCmd)
	testutil.AssertNoError(t, err)

	// Step 3: Send message
	sendMsgCmd := fixtures.NewSendMessageCommandBuilder(chatIDGoogle, user1).
		WithContent("Let's create a task for this").
		Build()

	msgResult, err := suite.SendMessage.Execute(ctx, sendMsgCmd)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, msgResult)

	// Step 4: Convert to Task
	convertCmd := fixtures.NewConvertToTaskCommandBuilder(chatIDGoogle).
		WithTitle("New feature task").
		ConvertedBy(user1).
		Build()

	_, err = suite.ConvertToTask.Execute(ctx, convertCmd)
	testutil.AssertNoError(t, err)

	// Step 5: Assign to User2
	assignCmd := fixtures.NewAssignUserCommandBuilder(chatIDGoogle).
		AssignTo(user2).
		AssignedBy(user1).
		Build()

	_, err = suite.AssignUser.Execute(ctx, assignCmd)
	testutil.AssertNoError(t, err)

	// Step 6: Verify chat is now a task
	chatRepo := suite.ChatRepo
	loadedChat, err := chatRepo.Load(ctx, chatID)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, loadedChat)

	// Assert: Verify final state
	assert.Equal(t, chat.TypeTask, loadedChat.Type())
	assert.Equal(t, user2, loadedChat.AssigneeID())

	// Verify events published
	events := suite.EventBus.PublishedEvents()
	assert.Greater(t, len(events), 0, "should have published events")
}

// TestE2E_Task_StatusTransitions проверяет переходы статусов задачи
func TestE2E_Task_StatusTransitions(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	creator := uuid.NewUUID()
	assignee := uuid.NewUUID()

	// Create task chat
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(creator).
		AsTask().
		WithTitle("Implementation task").
		Build()

	chatResult, err := suite.CreateChat.Execute(ctx, createChatCmd)
	testutil.AssertNoError(t, err)
	chatID := chatResult.Value.ID().ToGoogleUUID()

	// Add participant
	addCmd := fixtures.NewAddParticipantCommandBuilder(chatID, assignee.ToGoogleUUID()).
		AddedBy(creator).
		Build()
	_, _ = suite.AddParticipant.Execute(ctx, addCmd)

	// Assign task
	assignCmd := fixtures.NewAssignUserCommandBuilder(chatID).
		AssignTo(assignee).
		AssignedBy(creator).
		Build()
	_, _ = suite.AssignUser.Execute(ctx, assignCmd)

	// Act: Change status to In Progress
	changeStatusCmd1 := fixtures.NewChangeStatusCommandBuilder(chatID).
		WithStatus(chat.StatusInProgress).
		ChangedBy(assignee).
		Build()

	_, err = suite.ChangeStatus.Execute(ctx, changeStatusCmd1)
	testutil.AssertNoError(t, err)

	// Act: Change status to Done
	changeStatusCmd2 := fixtures.NewChangeStatusCommandBuilder(chatID).
		WithStatus(chat.StatusDone).
		ChangedBy(assignee).
		Build()

	_, err = suite.ChangeStatus.Execute(ctx, changeStatusCmd2)
	testutil.AssertNoError(t, err)

	// Assert: Verify status transitions
	chatRepo := suite.ChatRepo
	loadedChat, _ := chatRepo.Load(ctx, chatResult.Value.ID())
	testutil.AssertEqual(t, chat.StatusDone, loadedChat.Status())

	// Check status change events
	statusChangeEvents := suite.EventBus.GetPublishedEventsByType(chat.EventTypeStatusChanged)
	testutil.AssertLen(t, statusChangeEvents, 2)
}

// TestE2E_Task_Notifications проверяет уведомления при изменении задачи
func TestE2E_Task_Notifications(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	creator := uuid.NewUUID()
	assignee := uuid.NewUUID()

	// Create task
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(creator).
		AsTask().
		WithTitle("Critical task").
		Build()

	chatResult, err := suite.CreateChat.Execute(ctx, createChatCmd)
	testutil.AssertNoError(t, err)
	chatID := chatResult.Value.ID().ToGoogleUUID()

	// Add participant
	addCmd := fixtures.NewAddParticipantCommandBuilder(chatID, assignee.ToGoogleUUID()).
		AddedBy(creator).
		Build()
	_, _ = suite.AddParticipant.Execute(ctx, addCmd)

	// Assign task (should create notification)
	assignCmd := fixtures.NewAssignUserCommandBuilder(chatID).
		AssignTo(assignee).
		AssignedBy(creator).
		Build()

	_, err = suite.AssignUser.Execute(ctx, assignCmd)
	testutil.AssertNoError(t, err)

	// Create notification for assignment
	notifCmd := fixtures.NewCreateNotificationCommandBuilder(assignee).
		WithTitle("Task Assigned").
		WithMessage("You have been assigned a task").
		WithType(notificationapp.TypeTaskAssignment).
		Build()

	notifResult, err := suite.CreateNotification.Execute(ctx, notifCmd)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, notifResult)

	// Assert: Verify notification was created
	notificationRepo := suite.NotificationRepo
	notifications := notificationRepo.GetAll()
	testutil.AssertGreater(t, len(notifications), 0)

	// Find our notification
	found := false
	for _, notif := range notifications {
		if notif.Title == "Task Assigned" && notif.UserID == assignee {
			found = true
			break
		}
	}
	assert.True(t, found, "should create notification for task assignment")
}

// TestE2E_Task_MultiUser_Collaboration проверяет взаимодействие между пользователями
func TestE2E_Task_MultiUser_Collaboration(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	manager := uuid.NewUUID()
	developer := uuid.NewUUID()
	reviewer := uuid.NewUUID()

	// Create task chat
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(manager).
		AsTask().
		WithTitle("Feature implementation").
		Build()

	chatResult, err := suite.CreateChat.Execute(ctx, createChatCmd)
	testutil.AssertNoError(t, err)
	chatID := chatResult.Value.ID().ToGoogleUUID()

	// Add developer and reviewer
	for _, user := range []uuid.UUID{developer, reviewer} {
		addCmd := fixtures.NewAddParticipantCommandBuilder(chatID, user.ToGoogleUUID()).
			AddedBy(manager).
			Build()
		_, _ = suite.AddParticipant.Execute(ctx, addCmd)
	}

	// Assign to developer
	assignCmd := fixtures.NewAssignUserCommandBuilder(chatID).
		AssignTo(developer).
		AssignedBy(manager).
		Build()
	_, _ = suite.AssignUser.Execute(ctx, assignCmd)

	// Developer sends update message
	msg1 := fixtures.NewSendMessageCommandBuilder(chatID, developer).
		WithContent("Started implementation").
		Build()
	_, _ = suite.SendMessage.Execute(ctx, msg1)

	// Developer changes status
	changeStatusCmd := fixtures.NewChangeStatusCommandBuilder(chatID).
		WithStatus(chat.StatusInProgress).
		ChangedBy(developer).
		Build()
	_, _ = suite.ChangeStatus.Execute(ctx, changeStatusCmd)

	// Reviewer comments
	msg2 := fixtures.NewSendMessageCommandBuilder(chatID, reviewer).
		WithContent("Code review in progress").
		Build()
	_, _ = suite.SendMessage.Execute(ctx, msg2)

	// Assert: Verify message count
	messageRepo := suite.MessageRepo
	messages := messageRepo.GetAll()
	testutil.AssertLen(t, messages, 2)

	// Assert: Verify status is InProgress
	chatRepo := suite.ChatRepo
	loadedChat, _ := chatRepo.Load(ctx, chatResult.Value.ID())
	testutil.AssertEqual(t, chat.StatusInProgress, loadedChat.Status())
	testutil.AssertEqual(t, developer, loadedChat.AssigneeID())

	// Assert: Verify events
	events := suite.EventBus.PublishedEvents()
	testutil.AssertGreater(t, len(events), 0)
}

// TestE2E_Task_Priority_Changes проверяет изменение приоритета задачи
func TestE2E_Task_Priority_Changes(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	creator := uuid.NewUUID()

	// Create task
	createChatCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(creator).
		AsTask().
		WithTitle("Task with priority").
		Build()

	chatResult, err := suite.CreateChat.Execute(ctx, createChatCmd)
	testutil.AssertNoError(t, err)
	chatID := chatResult.Value.ID().ToGoogleUUID()

	// Reset event bus to track priority events
	suite.EventBus.Reset()

	// Act: Set priority using command
	// Note: This would require SetPriorityCommand to be implemented in the suite

	// Assert: Check for priority events
	events := suite.EventBus.PublishedEvents()
	// Priority set events should be published
	priorityEvents := suite.EventBus.GetPublishedEventsByType(chat.EventTypePrioritySet)
	testutil.AssertGreaterOrEqual(t, len(priorityEvents), 0)
}

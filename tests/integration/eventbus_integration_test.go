//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/tests/fixtures"
	"github.com/lllypuk/flowra/tests/mocks"
	"github.com/lllypuk/flowra/tests/testutil"
)

// TestEventBusIntegration_ChatCreated_PublishesEvent проверяет, что событие ChatCreated опубликовано
func TestEventBusIntegration_ChatCreated_PublishesEvent(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	chatRepo := mocks.NewMockChatRepository()
	eventBus := mocks.NewMockEventBus()

	createChatUseCase := chatapp.NewCreateChatUseCase(chatRepo, eventBus)

	workspaceID := uuid.NewUUID()
	creatorID := uuid.NewUUID()

	cmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(creatorID).
		WithTitle("Test Chat").
		Build()

	// Act
	result, err := createChatUseCase.Execute(ctx, cmd)

	// Assert
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, result)

	// Проверяем, что событие было опубликовано
	publishedEvents := eventBus.PublishedEvents()
	testutil.AssertGreater(t, len(publishedEvents), 0, "should publish at least one event")

	// Проверяем, что это событие ChatCreated
	chatCreatedEvent := testutil.AssertEventPublished(t, publishedEvents, chat.EventTypeChatCreated)
	testutil.AssertNotNil(t, chatCreatedEvent)
}

// TestEventBusIntegration_MultipleEvents_PublishedInOrder проверяет последовательность событий
func TestEventBusIntegration_MultipleEvents_PublishedInOrder(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	chatRepo := mocks.NewMockChatRepository()
	eventBus := mocks.NewMockEventBus()

	createChatUseCase := chatapp.NewCreateChatUseCase(chatRepo, eventBus)
	addParticipantUseCase := chatapp.NewAddParticipantUseCase(chatRepo, eventBus)

	workspaceID := uuid.NewUUID()
	creatorID := uuid.NewUUID()
	participantID := uuid.NewUUID()

	// Act: Create chat
	createCmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(creatorID).
		Build()

	createResult, err := createChatUseCase.Execute(ctx, createCmd)
	testutil.AssertNoError(t, err)
	chatID := createResult.Value.ID()

	// Act: Add participant
	addCmd := fixtures.NewAddParticipantCommandBuilder(chatID.ToGoogleUUID(), participantID.ToGoogleUUID()).
		AddedBy(creatorID).
		Build()

	_, err = addParticipantUseCase.Execute(ctx, addCmd)
	testutil.AssertNoError(t, err)

	// Assert: Check event order
	publishedEvents := eventBus.PublishedEvents()
	testutil.AssertLen(t, publishedEvents, 2)

	// First event should be ChatCreated
	testutil.AssertEventType(t, publishedEvents[0], chat.EventTypeChatCreated)

	// Second event should be ParticipantAdded
	testutil.AssertEventType(t, publishedEvents[1], chat.EventTypeParticipantAdded)
}

// TestEventBusIntegration_EventWithMetadata проверяет метаданные события
func TestEventBusIntegration_EventWithMetadata(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	chatRepo := mocks.NewMockChatRepository()
	eventBus := mocks.NewMockEventBus()

	createChatUseCase := chatapp.NewCreateChatUseCase(chatRepo, eventBus)

	workspaceID := uuid.NewUUID()
	creatorID := uuid.NewUUID()

	cmd := fixtures.NewCreateChatCommandBuilder().
		WithWorkspace(workspaceID).
		CreatedBy(creatorID).
		Build()

	// Act
	result, err := createChatUseCase.Execute(ctx, cmd)
	testutil.AssertNoError(t, err)

	// Assert
	publishedEvents := eventBus.PublishedEvents()
	testutil.AssertGreater(t, len(publishedEvents), 0)

	event := publishedEvents[0]
	testutil.AssertNotNil(t, event.Metadata())
	testutil.AssertNotNil(t, event.AggregateID())
	testutil.AssertGreaterOrEqual(t, event.Version(), 1)
}

// TestEventBusIntegration_HandlerSubscription проверяет подписку на события
func TestEventBusIntegration_HandlerSubscription(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	eventBus := mocks.NewMockEventBus()

	handlerCalled := false
	handler := func(ctx context.Context, evt interface{}) error {
		handlerCalled = true
		return nil
	}

	// Subscribe to event
	eventBus.Subscribe(chat.EventTypeChatCreated, func(ctx context.Context, evt interface{}) error {
		return handler(ctx, evt)
	})

	// Verify handler count
	handlerCount := eventBus.HandlerCount(chat.EventTypeChatCreated)
	testutil.AssertEqual(t, 1, handlerCount)
}

// TestEventBusIntegration_GetPublishedEventsByType проверяет фильтрацию событий по типу
func TestEventBusIntegration_GetPublishedEventsByType(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	workspaceID := uuid.NewUUID()
	creatorID := uuid.NewUUID()

	// Act: Create multiple chats
	for i := 0; i < 3; i++ {
		cmd := fixtures.NewCreateChatCommandBuilder().
			WithWorkspace(workspaceID).
			CreatedBy(creatorID).
			Build()

		_, err := suite.CreateChat.Execute(ctx, cmd)
		testutil.AssertNoError(t, err)
	}

	// Assert: Check events by type
	chatCreatedEvents := suite.EventBus.GetPublishedEventsByType(chat.EventTypeChatCreated)
	testutil.AssertLen(t, chatCreatedEvents, 3)

	for _, evt := range chatCreatedEvents {
		testutil.AssertEventType(t, evt, chat.EventTypeChatCreated)
	}
}

// TestEventBusIntegration_EventPublishCount проверяет количество опубликованных событий
func TestEventBusIntegration_EventPublishCount(t *testing.T) {
	// Arrange
	ctx := testutil.NewTestContext(t)
	suite := testutil.NewTestSuite(t)

	initialCount := suite.EventBus.PublishedCount()
	testutil.AssertEqual(t, 0, initialCount)

	// Act
	cmd := fixtures.NewCreateChatCommandBuilder().Build()
	_, err := suite.CreateChat.Execute(ctx, cmd)
	testutil.AssertNoError(t, err)

	// Assert
	finalCount := suite.EventBus.PublishedCount()
	testutil.AssertGreater(t, finalCount, initialCount)
}

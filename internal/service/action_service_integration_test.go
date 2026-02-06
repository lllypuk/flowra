package service

import (
	"context"
	"sync"
	"testing"
	"time"

	domainu "github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionService_RapidChanges_Batched(t *testing.T) {
	var messageContent string
	var callCount int
	var mu sync.Mutex

	// Mock flush function
	flushFunc := func(ctx context.Context, chatID domainu.UUID, content string, actorID domainu.UUID) error {
		mu.Lock()
		messageContent = content
		callCount++
		mu.Unlock()
		return nil
	}

	batcher := NewChangeBatcher(100*time.Millisecond, flushFunc)
	defer batcher.Close()

	actorID := domainu.NewUUID()
	chatID := domainu.NewUUID()

	// Make rapid changes
	err := batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypeStatus, "In Progress")
	require.NoError(t, err)

	err = batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypePriority, "High")
	require.NoError(t, err)

	err = batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypeAssignee, "Jane")
	require.NoError(t, err)

	// Wait for batch to flush
	time.Sleep(250 * time.Millisecond)

	mu.Lock()
	content := messageContent
	count := callCount
	mu.Unlock()

	// Should have been called once with combined message
	assert.Equal(t, 1, count)
	assert.Contains(t, content, "John")
	assert.Contains(t, content, "changed status to In Progress")
	assert.Contains(t, content, "set priority to High")
	assert.Contains(t, content, "assigned this to Jane")
}

func TestActionService_DifferentActors_SeparateBatches(t *testing.T) {
	var callCount int
	var mu sync.Mutex

	// Mock flush function
	flushFunc := func(ctx context.Context, chatID domainu.UUID, content string, actorID domainu.UUID) error {
		mu.Lock()
		callCount++
		mu.Unlock()
		return nil
	}

	batcher := NewChangeBatcher(100*time.Millisecond, flushFunc)
	defer batcher.Close()

	actor1 := domainu.NewUUID()
	actor2 := domainu.NewUUID()
	chatID := domainu.NewUUID()

	// Changes from actor1
	err := batcher.AddChange(context.Background(), actor1, chatID, "John", ChangeTypeStatus, "In Progress")
	require.NoError(t, err)

	// Changes from actor2 - should trigger separate batch
	err = batcher.AddChange(context.Background(), actor2, chatID, "Jane", ChangeTypePriority, "High")
	require.NoError(t, err)

	// Wait for batches to flush
	time.Sleep(250 * time.Millisecond)

	mu.Lock()
	count := callCount
	mu.Unlock()

	// Should have been called twice (once per actor)
	assert.Equal(t, 2, count)
}

func TestActionService_DifferentChats_SeparateBatches(t *testing.T) {
	var callCount int
	var mu sync.Mutex

	// Mock flush function
	flushFunc := func(ctx context.Context, chatID domainu.UUID, content string, actorID domainu.UUID) error {
		mu.Lock()
		callCount++
		mu.Unlock()
		return nil
	}

	batcher := NewChangeBatcher(100*time.Millisecond, flushFunc)
	defer batcher.Close()

	actorID := domainu.NewUUID()
	chat1 := domainu.NewUUID()
	chat2 := domainu.NewUUID()

	// Changes to different chats
	err := batcher.AddChange(context.Background(), actorID, chat1, "John", ChangeTypeStatus, "In Progress")
	require.NoError(t, err)

	err = batcher.AddChange(context.Background(), actorID, chat2, "John", ChangeTypePriority, "High")
	require.NoError(t, err)

	// Wait for batches to flush
	time.Sleep(250 * time.Millisecond)

	mu.Lock()
	count := callCount
	mu.Unlock()

	// Should have been called twice (once per chat)
	assert.Equal(t, 2, count)
}


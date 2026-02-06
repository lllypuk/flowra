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

func TestChangeBatcher_SingleChange(t *testing.T) {
	var flushedContent string
	var mu sync.Mutex
	flushFunc := func(ctx context.Context, chatID domainu.UUID, content string, actorID domainu.UUID) error {
		mu.Lock()
		flushedContent = content
		mu.Unlock()
		return nil
	}

	batcher := NewChangeBatcher(100*time.Millisecond, flushFunc)
	defer batcher.Close()

	actorID := domainu.NewUUID()
	chatID := domainu.NewUUID()

	err := batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypeStatus, "In Progress")
	require.NoError(t, err)

	// Wait for batch to flush
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	content := flushedContent
	mu.Unlock()

	assert.Contains(t, content, "John")
	assert.Contains(t, content, "changed status to In Progress")
}

func TestChangeBatcher_MultipleChanges(t *testing.T) {
	var flushedContent string
	var mu sync.Mutex
	flushFunc := func(ctx context.Context, chatID domainu.UUID, content string, actorID domainu.UUID) error {
		mu.Lock()
		flushedContent = content
		mu.Unlock()
		return nil
	}

	batcher := NewChangeBatcher(100*time.Millisecond, flushFunc)
	defer batcher.Close()

	actorID := domainu.NewUUID()
	chatID := domainu.NewUUID()

	// Add multiple changes rapidly
	err := batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypeStatus, "In Progress")
	require.NoError(t, err)

	err = batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypePriority, "High")
	require.NoError(t, err)

	err = batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypeAssignee, "Jane")
	require.NoError(t, err)

	// Wait for batch to flush
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	content := flushedContent
	mu.Unlock()

	// Should combine all changes
	assert.Contains(t, content, "John")
	assert.Contains(t, content, "changed status to In Progress")
	assert.Contains(t, content, "set priority to High")
	assert.Contains(t, content, "assigned this to Jane")
	assert.Contains(t, content, "and") // Should have conjunctions
}

func TestChangeBatcher_TwoChanges(t *testing.T) {
	var flushedContent string
	var mu sync.Mutex
	flushFunc := func(ctx context.Context, chatID domainu.UUID, content string, actorID domainu.UUID) error {
		mu.Lock()
		flushedContent = content
		mu.Unlock()
		return nil
	}

	batcher := NewChangeBatcher(100*time.Millisecond, flushFunc)
	defer batcher.Close()

	actorID := domainu.NewUUID()
	chatID := domainu.NewUUID()

	err := batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypeStatus, "Done")
	require.NoError(t, err)

	err = batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypePriority, "Low")
	require.NoError(t, err)

	// Wait for batch to flush
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	content := flushedContent
	mu.Unlock()

	// Two changes should use "and" without commas
	assert.Contains(t, content, "John")
	assert.Contains(t, content, "changed status to Done")
	assert.Contains(t, content, "and")
	assert.Contains(t, content, "set priority to Low")
}

func TestChangeBatcher_DifferentActors(t *testing.T) {
	var flushedCount int
	var mu sync.Mutex
	flushFunc := func(ctx context.Context, chatID domainu.UUID, content string, actorID domainu.UUID) error {
		mu.Lock()
		flushedCount++
		mu.Unlock()
		return nil
	}

	batcher := NewChangeBatcher(100*time.Millisecond, flushFunc)
	defer batcher.Close()

	actor1 := domainu.NewUUID()
	actor2 := domainu.NewUUID()
	chatID := domainu.NewUUID()

	// Changes from different actors should not be batched together
	err := batcher.AddChange(context.Background(), actor1, chatID, "John", ChangeTypeStatus, "In Progress")
	require.NoError(t, err)

	err = batcher.AddChange(context.Background(), actor2, chatID, "Jane", ChangeTypePriority, "High")
	require.NoError(t, err)

	// Wait for batches to flush
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	count := flushedCount
	mu.Unlock()

	// Should have 2 separate batches
	assert.Equal(t, 2, count)
}

func TestChangeBatcher_DifferentChats(t *testing.T) {
	var flushedCount int
	var mu sync.Mutex
	flushFunc := func(ctx context.Context, chatID domainu.UUID, content string, actorID domainu.UUID) error {
		mu.Lock()
		flushedCount++
		mu.Unlock()
		return nil
	}

	batcher := NewChangeBatcher(100*time.Millisecond, flushFunc)
	defer batcher.Close()

	actorID := domainu.NewUUID()
	chat1 := domainu.NewUUID()
	chat2 := domainu.NewUUID()

	// Changes to different chats should not be batched together
	err := batcher.AddChange(context.Background(), actorID, chat1, "John", ChangeTypeStatus, "In Progress")
	require.NoError(t, err)

	err = batcher.AddChange(context.Background(), actorID, chat2, "John", ChangeTypePriority, "High")
	require.NoError(t, err)

	// Wait for batches to flush
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	count := flushedCount
	mu.Unlock()

	// Should have 2 separate batches
	assert.Equal(t, 2, count)
}

func TestChangeBatcher_TimerReset(t *testing.T) {
	var flushTime time.Time
	var mu sync.Mutex
	flushFunc := func(ctx context.Context, chatID domainu.UUID, content string, actorID domainu.UUID) error {
		mu.Lock()
		flushTime = time.Now()
		mu.Unlock()
		return nil
	}

	batcher := NewChangeBatcher(150*time.Millisecond, flushFunc)
	defer batcher.Close()

	actorID := domainu.NewUUID()
	chatID := domainu.NewUUID()

	start := time.Now()

	// Add first change
	err := batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypeStatus, "In Progress")
	require.NoError(t, err)

	// Wait 100ms and add another change (should reset timer)
	time.Sleep(100 * time.Millisecond)
	err = batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypePriority, "High")
	require.NoError(t, err)

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	ft := flushTime
	mu.Unlock()

	elapsed := ft.Sub(start)

	// Should have taken at least 250ms (100ms wait + 150ms batch window)
	// but less than 350ms (to account for timer reset)
	assert.Greater(t, elapsed.Milliseconds(), int64(200))
	assert.Less(t, elapsed.Milliseconds(), int64(350))
}

func TestChangeBatcher_RemoveAssignee(t *testing.T) {
	var flushedContent string
	var mu sync.Mutex
	flushFunc := func(ctx context.Context, chatID domainu.UUID, content string, actorID domainu.UUID) error {
		mu.Lock()
		flushedContent = content
		mu.Unlock()
		return nil
	}

	batcher := NewChangeBatcher(100*time.Millisecond, flushFunc)
	defer batcher.Close()

	actorID := domainu.NewUUID()
	chatID := domainu.NewUUID()

	// Empty value means removal
	err := batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypeAssignee, "")
	require.NoError(t, err)

	// Wait for batch to flush
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	content := flushedContent
	mu.Unlock()

	assert.Contains(t, content, "removed the assignee")
}

func TestChangeBatcher_RemoveDueDate(t *testing.T) {
	var flushedContent string
	var mu sync.Mutex
	flushFunc := func(ctx context.Context, chatID domainu.UUID, content string, actorID domainu.UUID) error {
		mu.Lock()
		flushedContent = content
		mu.Unlock()
		return nil
	}

	batcher := NewChangeBatcher(100*time.Millisecond, flushFunc)
	defer batcher.Close()

	actorID := domainu.NewUUID()
	chatID := domainu.NewUUID()

	// Empty value means removal
	err := batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypeDueDate, "")
	require.NoError(t, err)

	// Wait for batch to flush
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	content := flushedContent
	mu.Unlock()

	assert.Contains(t, content, "removed the due date")
}

func TestChangeBatcher_NoActorName(t *testing.T) {
	var flushedContent string
	var mu sync.Mutex
	flushFunc := func(ctx context.Context, chatID domainu.UUID, content string, actorID domainu.UUID) error {
		mu.Lock()
		flushedContent = content
		mu.Unlock()
		return nil
	}

	batcher := NewChangeBatcher(100*time.Millisecond, flushFunc)
	defer batcher.Close()

	actorID := domainu.NewUUID()
	chatID := domainu.NewUUID()

	// Empty actor name
	err := batcher.AddChange(context.Background(), actorID, chatID, "", ChangeTypeStatus, "Done")
	require.NoError(t, err)

	// Wait for batch to flush
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	content := flushedContent
	mu.Unlock()

	// Should still work without actor name
	assert.Contains(t, content, "changed status to Done")
	assert.Contains(t, content, "✅")
}

func TestChangeBatcher_Close(t *testing.T) {
	flushFunc := func(ctx context.Context, chatID domainu.UUID, content string, actorID domainu.UUID) error {
		return nil
	}

	batcher := NewChangeBatcher(100*time.Millisecond, flushFunc)

	actorID := domainu.NewUUID()
	chatID := domainu.NewUUID()

	err := batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypeStatus, "In Progress")
	require.NoError(t, err)

	// Close should not panic
	batcher.Close()

	// Adding after close should not panic (though it won't flush)
	err = batcher.AddChange(context.Background(), actorID, chatID, "John", ChangeTypePriority, "High")
	// Error is acceptable here since batcher is closed
	_ = err
}

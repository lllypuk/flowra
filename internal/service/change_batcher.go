package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// ChangeType represents the type of change made
type ChangeType string

const (
	ChangeTypeStatus   ChangeType = "status"
	ChangeTypePriority ChangeType = "priority"
	ChangeTypeAssignee ChangeType = "assignee"
	ChangeTypeDueDate  ChangeType = "due_date"
	ChangeTypeTitle    ChangeType = "title"

	// defaultBatchWindow is the default time window for batching changes
	defaultBatchWindow = 2 * time.Second
	// cleanupInterval is how often to clean up abandoned batches
	cleanupInterval = 10 * time.Second
	// twoItems is used for checking if we have exactly 2 items for simple conjunction
	twoItems = 2
)

// Change represents a single change to an entity
type Change struct {
	Type      ChangeType
	Value     string // human-readable value
	Timestamp time.Time
}

// PendingBatch represents a batch of changes waiting to be flushed
type PendingBatch struct {
	ActorID   uuid.UUID
	ChatID    uuid.UUID
	ActorName string
	Changes   []Change
	FirstTime time.Time
	LastTime  time.Time
	timer     *time.Timer
}

// batchKey uniquely identifies a batch by actor and chat
type batchKey struct {
	ActorID uuid.UUID
	ChatID  uuid.UUID
}

// ChangeBatcher batches rapid UI changes into single messages
type ChangeBatcher struct {
	batches      map[batchKey]*PendingBatch
	mu           sync.Mutex
	batchWindow  time.Duration
	cleanupTimer time.Duration
	flushFunc    func(ctx context.Context, chatID uuid.UUID, content string, actorID uuid.UUID) error
	stopCleanup  chan struct{}
	wg           sync.WaitGroup
	logger       *slog.Logger
}

// NewChangeBatcher creates a new change batcher
func NewChangeBatcher(
	batchWindow time.Duration,
	flushFunc func(ctx context.Context, chatID uuid.UUID, content string, actorID uuid.UUID) error,
) *ChangeBatcher {
	if batchWindow == 0 {
		batchWindow = defaultBatchWindow
	}

	cb := &ChangeBatcher{
		batches:      make(map[batchKey]*PendingBatch),
		batchWindow:  batchWindow,
		cleanupTimer: cleanupInterval,
		flushFunc:    flushFunc,
		stopCleanup:  make(chan struct{}),
	}

	// Start cleanup goroutine
	cb.wg.Add(1)
	go cb.cleanupLoop()

	return cb
}

// AddChange adds a change to the batch or creates a new batch
func (cb *ChangeBatcher) AddChange(
	_ context.Context,
	actorID uuid.UUID,
	chatID uuid.UUID,
	actorName string,
	changeType ChangeType,
	value string,
) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	key := batchKey{ActorID: actorID, ChatID: chatID}
	batch, exists := cb.batches[key]

	now := time.Now()
	change := Change{
		Type:      changeType,
		Value:     value,
		Timestamp: now,
	}

	if !exists {
		// Create new batch
		batch = &PendingBatch{
			ActorID:   actorID,
			ChatID:    chatID,
			ActorName: actorName,
			Changes:   []Change{change},
			FirstTime: now,
			LastTime:  now,
		}
		cb.batches[key] = batch

		// Schedule flush with background context to avoid cancellation
		batch.timer = time.AfterFunc(cb.batchWindow, func() {
			cb.flushBatch(context.Background(), key)
		})
	} else {
		// Add to existing batch
		batch.Changes = append(batch.Changes, change)
		batch.LastTime = now

		// Reset timer with background context to avoid cancellation
		if batch.timer != nil {
			batch.timer.Stop()
		}
		batch.timer = time.AfterFunc(cb.batchWindow, func() {
			cb.flushBatch(context.Background(), key)
		})
	}

	return nil
}

// flushBatch flushes a batch and sends a combined message
func (cb *ChangeBatcher) flushBatch(ctx context.Context, key batchKey) {
	cb.mu.Lock()
	batch, exists := cb.batches[key]
	if !exists {
		cb.mu.Unlock()
		return
	}
	delete(cb.batches, key)
	cb.mu.Unlock()

	// Format combined message
	content := cb.formatBatchMessage(batch)

	// Send message
	if cb.flushFunc != nil {
		if err := cb.flushFunc(ctx, batch.ChatID, content, batch.ActorID); err != nil {
			if cb.logger != nil {
				cb.logger.ErrorContext(ctx, "failed to flush batch message",
					"chat_id", batch.ChatID.String(),
					"actor_id", batch.ActorID.String(),
					"error", err.Error(),
				)
			}
		}
	}
}

// formatBatchMessage formats a batch of changes into a human-readable message
func (cb *ChangeBatcher) formatBatchMessage(batch *PendingBatch) string {
	if len(batch.Changes) == 0 {
		return ""
	}

	if len(batch.Changes) == 1 {
		// Single change - format simply
		change := batch.Changes[0]
		return cb.formatSingleChange(batch.ActorName, change)
	}

	// Multiple changes - combine them
	var parts []string
	for _, change := range batch.Changes {
		parts = append(parts, cb.formatChangeAction(change))
	}

	actorPrefix := ""
	if batch.ActorName != "" {
		actorPrefix = batch.ActorName + " "
	}

	// Join with commas and "and" for last item
	if len(parts) == twoItems {
		return fmt.Sprintf("✅ %s%s and %s", actorPrefix, parts[0], parts[1])
	}

	lastPart := parts[len(parts)-1]
	otherParts := parts[:len(parts)-1]
	return fmt.Sprintf("✅ %s%s, and %s", actorPrefix, strings.Join(otherParts, ", "), lastPart)
}

// formatSingleChange formats a single change
func (cb *ChangeBatcher) formatSingleChange(actorName string, change Change) string {
	actorPrefix := ""
	if actorName != "" {
		actorPrefix = actorName + " "
	}

	switch change.Type {
	case ChangeTypeStatus:
		return fmt.Sprintf("✅ %schanged status to %s", actorPrefix, change.Value)
	case ChangeTypePriority:
		return fmt.Sprintf("✅ %sset priority to %s", actorPrefix, change.Value)
	case ChangeTypeAssignee:
		if change.Value == "" {
			return fmt.Sprintf("✅ %sremoved the assignee", actorPrefix)
		}
		return fmt.Sprintf("✅ %sassigned this to %s", actorPrefix, change.Value)
	case ChangeTypeDueDate:
		if change.Value == "" {
			return fmt.Sprintf("✅ %sremoved the due date", actorPrefix)
		}
		return fmt.Sprintf("✅ %sset due date to %s", actorPrefix, change.Value)
	case ChangeTypeTitle:
		return fmt.Sprintf("✅ %schanged title to: %s", actorPrefix, change.Value)
	default:
		return fmt.Sprintf("✅ %smade a change", actorPrefix)
	}
}

// formatChangeAction formats a change as an action phrase (for combining)
func (cb *ChangeBatcher) formatChangeAction(change Change) string {
	switch change.Type {
	case ChangeTypeStatus:
		return fmt.Sprintf("changed status to %s", change.Value)
	case ChangeTypePriority:
		return fmt.Sprintf("set priority to %s", change.Value)
	case ChangeTypeAssignee:
		if change.Value == "" {
			return "removed the assignee"
		}
		return fmt.Sprintf("assigned this to %s", change.Value)
	case ChangeTypeDueDate:
		if change.Value == "" {
			return "removed the due date"
		}
		return fmt.Sprintf("set due date to %s", change.Value)
	case ChangeTypeTitle:
		return fmt.Sprintf("changed title to: %s", change.Value)
	default:
		return "made a change"
	}
}

// cleanupLoop periodically cleans up abandoned batches
func (cb *ChangeBatcher) cleanupLoop() {
	defer cb.wg.Done()

	ticker := time.NewTicker(cb.cleanupTimer)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cb.cleanupAbandonedBatches()
		case <-cb.stopCleanup:
			return
		}
	}
}

// cleanupAbandonedBatches removes batches older than 10 seconds
func (cb *ChangeBatcher) cleanupAbandonedBatches() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	for key, batch := range cb.batches {
		if now.Sub(batch.LastTime) > 10*time.Second {
			if batch.timer != nil {
				batch.timer.Stop()
			}
			delete(cb.batches, key)
		}
	}
}

// Close stops the batcher and waits for cleanup to finish
func (cb *ChangeBatcher) Close() {
	close(cb.stopCleanup)
	cb.wg.Wait()

	// Stop all timers
	cb.mu.Lock()
	for _, batch := range cb.batches {
		if batch.timer != nil {
			batch.timer.Stop()
		}
	}
	cb.batches = make(map[batchKey]*PendingBatch)
	cb.mu.Unlock()
}

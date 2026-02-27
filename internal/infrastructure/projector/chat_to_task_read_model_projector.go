package projector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ChatToTaskReadModelProjector rebuilds tasks_read_model from chat.* event streams.
type ChatToTaskReadModelProjector struct {
	eventStore    appcore.EventStore
	readModelColl *mongo.Collection
	logger        *slog.Logger
}

// NewChatToTaskReadModelProjector creates a new projector that maps chat state to task read model shape.
func NewChatToTaskReadModelProjector(
	eventStore appcore.EventStore,
	readModelColl *mongo.Collection,
	logger *slog.Logger,
) *ChatToTaskReadModelProjector {
	if logger == nil {
		logger = slog.Default()
	}
	return &ChatToTaskReadModelProjector{
		eventStore:    eventStore,
		readModelColl: readModelColl,
		logger:        logger,
	}
}

// RebuildOne rebuilds a single tasks_read_model document from chat.* events only.
func (p *ChatToTaskReadModelProjector) RebuildOne(ctx context.Context, chatID uuid.UUID) error {
	events, err := p.eventStore.LoadEvents(ctx, chatID.String())
	if err != nil {
		return fmt.Errorf("failed to load events for chat %s: %w", chatID, err)
	}

	chatEvents := filterChatEvents(events)
	if len(chatEvents) == 0 {
		return appcore.ErrAggregateNotFound
	}

	aggregate, err := replayChatEvents(chatEvents)
	if err != nil {
		return fmt.Errorf("failed to rebuild chat aggregate: %w", err)
	}

	return p.syncReadModel(ctx, aggregate)
}

// RebuildAll rebuilds tasks_read_model for every chat aggregate found in events.
func (p *ChatToTaskReadModelProjector) RebuildAll(ctx context.Context) error {
	aggregateIDs, err := p.getAllAggregateIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get aggregate IDs: %w", err)
	}

	successCount := 0
	failCount := 0

	for _, id := range aggregateIDs {
		if rebuildErr := p.RebuildOne(ctx, id); rebuildErr != nil {
			p.logger.ErrorContext(ctx, "failed to rebuild task projection from chat",
				slog.String("chat_id", id.String()),
				slog.String("error", rebuildErr.Error()),
			)
			failCount++
			continue
		}
		successCount++
	}

	p.logger.InfoContext(ctx, "completed task projection rebuild from chat streams",
		slog.Int("total", len(aggregateIDs)),
		slog.Int("success", successCount),
		slog.Int("failed", failCount),
	)

	if failCount > 0 {
		return fmt.Errorf("rebuild completed with %d failures out of %d total", failCount, len(aggregateIDs))
	}

	return nil
}

// ProcessEvent rebuilds one tasks_read_model document for a chat aggregate.
func (p *ChatToTaskReadModelProjector) ProcessEvent(ctx context.Context, evt event.DomainEvent) error {
	if !isAggregateType(evt.AggregateType(), aggregateTypeChat) {
		return fmt.Errorf("invalid aggregate type: expected '%s', got '%s'", aggregateTypeChat, evt.AggregateType())
	}

	if !strings.HasPrefix(evt.EventType(), chatEventPrefix) {
		return nil
	}

	chatID, err := uuid.ParseUUID(evt.AggregateID())
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	return p.RebuildOne(ctx, chatID)
}

// VerifyConsistency checks if tasks_read_model matches state derived from chat.* events.
func (p *ChatToTaskReadModelProjector) VerifyConsistency(ctx context.Context, chatID uuid.UUID) (bool, error) {
	events, err := p.eventStore.LoadEvents(ctx, chatID.String())
	if err != nil {
		if errors.Is(err, appcore.ErrAggregateNotFound) {
			return p.readModelAbsent(ctx, chatID)
		}
		return false, fmt.Errorf("failed to load events: %w", err)
	}

	chatEvents := filterChatEvents(events)
	if len(chatEvents) == 0 {
		return p.readModelAbsent(ctx, chatID)
	}

	aggregate, err := replayChatEvents(chatEvents)
	if err != nil {
		return false, fmt.Errorf("failed to replay chat events: %w", err)
	}

	expectedDoc, shouldExist, err := buildTaskProjectionDocument(aggregate)
	if err != nil {
		return false, fmt.Errorf("failed to build expected projection: %w", err)
	}

	if !shouldExist {
		return p.readModelAbsent(ctx, chatID)
	}

	var actualDoc taskProjectionDocument
	readErr := p.readModelColl.FindOne(ctx, bson.M{"task_id": chatID.String()}).Decode(&actualDoc)
	if readErr != nil {
		if errors.Is(readErr, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, fmt.Errorf("failed to load read model: %w", readErr)
	}

	return compareTaskProjectionDocuments(expectedDoc, &actualDoc), nil
}

func (p *ChatToTaskReadModelProjector) syncReadModel(ctx context.Context, aggregate *chatdomain.Chat) error {
	doc, shouldExist, err := buildTaskProjectionDocument(aggregate)
	if err != nil {
		return err
	}

	filter := bson.M{"task_id": aggregate.ID().String()}
	if !shouldExist {
		if _, deleteErr := p.readModelColl.DeleteOne(ctx, filter); deleteErr != nil {
			return fmt.Errorf("failed to delete task read model: %w", deleteErr)
		}
		return nil
	}

	update := bson.M{"$set": doc}
	opts := options.UpdateOne().SetUpsert(true)
	if _, updateErr := p.readModelColl.UpdateOne(ctx, filter, update, opts); updateErr != nil {
		return fmt.Errorf("failed to upsert task read model: %w", updateErr)
	}

	return nil
}

func (p *ChatToTaskReadModelProjector) readModelAbsent(ctx context.Context, chatID uuid.UUID) (bool, error) {
	count, err := p.readModelColl.CountDocuments(ctx, bson.M{"task_id": chatID.String()})
	if err != nil {
		return false, fmt.Errorf("failed to count task read model documents: %w", err)
	}
	return count == 0, nil
}

func (p *ChatToTaskReadModelProjector) getAllAggregateIDs(ctx context.Context) ([]uuid.UUID, error) {
	eventsColl := p.readModelColl.Database().Collection("events")
	filter := bson.M{"aggregate_type": bson.M{"$in": []string{aggregateTypeChat, "Chat"}}}
	result := eventsColl.Distinct(ctx, "aggregate_id", filter)

	var stringIDs []string
	if err := result.Decode(&stringIDs); err != nil {
		return nil, fmt.Errorf("failed to decode aggregate IDs: %w", err)
	}

	aggregateIDs := make([]uuid.UUID, 0, len(stringIDs))
	for _, idStr := range stringIDs {
		id, err := uuid.ParseUUID(idStr)
		if err != nil {
			p.logger.WarnContext(ctx, "skipping invalid aggregate ID",
				slog.String("aggregate_id", idStr),
				slog.String("error", err.Error()),
			)
			continue
		}
		aggregateIDs = append(aggregateIDs, id)
	}

	return aggregateIDs, nil
}

func replayChatEvents(events []event.DomainEvent) (*chatdomain.Chat, error) {
	aggregate := chatdomain.NewEmptyChat()
	for _, evt := range events {
		if err := aggregate.Apply(evt); err != nil {
			return nil, fmt.Errorf("failed to apply event %s: %w", evt.EventType(), err)
		}
	}

	if aggregate.ID().IsZero() {
		return nil, appcore.ErrAggregateNotFound
	}

	return aggregate, nil
}

func filterChatEvents(events []event.DomainEvent) []event.DomainEvent {
	chatEvents := make([]event.DomainEvent, 0, len(events))
	for _, evt := range events {
		if strings.HasPrefix(evt.EventType(), chatEventPrefix) {
			chatEvents = append(chatEvents, evt)
		}
	}
	return chatEvents
}

type taskProjectionDocument struct {
	TaskID      string                     `bson:"task_id"`
	ChatID      string                     `bson:"chat_id"`
	Title       string                     `bson:"title"`
	EntityType  string                     `bson:"entity_type"`
	Status      string                     `bson:"status"`
	Priority    string                     `bson:"priority"`
	Severity    *string                    `bson:"severity"`
	AssignedTo  *string                    `bson:"assigned_to"`
	DueDate     *time.Time                 `bson:"due_date"`
	CreatedBy   string                     `bson:"created_by"`
	CreatedAt   time.Time                  `bson:"created_at"`
	Version     int                        `bson:"version"`
	Attachments []taskProjectionAttachment `bson:"attachments"`
}

type taskProjectionAttachment struct {
	FileID   string `bson:"file_id"`
	FileName string `bson:"file_name"`
	FileSize int64  `bson:"file_size"`
	MimeType string `bson:"mime_type"`
}

func buildTaskProjectionDocument(aggregate *chatdomain.Chat) (*taskProjectionDocument, bool, error) {
	if aggregate == nil || aggregate.ID().IsZero() {
		return nil, false, appcore.ErrAggregateNotFound
	}

	if !aggregate.IsTyped() || aggregate.IsDeleted() {
		return nil, false, nil
	}

	entityType, err := mapChatTypeToTaskEntityType(aggregate.Type())
	if err != nil {
		return nil, false, err
	}

	priority := normalizeTaskPriority(aggregate.Priority())
	status := normalizeTaskStatus(aggregate.Status())

	doc := &taskProjectionDocument{
		TaskID:      aggregate.ID().String(),
		ChatID:      aggregate.ID().String(),
		Title:       aggregate.Title(),
		EntityType:  string(entityType),
		Status:      string(status),
		Priority:    string(priority),
		CreatedBy:   aggregate.CreatedBy().String(),
		CreatedAt:   aggregate.CreatedAt(),
		Version:     aggregate.Version(),
		Attachments: make([]taskProjectionAttachment, 0, len(aggregate.Attachments())),
	}

	if aggregate.Type() == chatdomain.TypeBug && strings.TrimSpace(aggregate.Severity()) != "" {
		severity := aggregate.Severity()
		doc.Severity = &severity
	}
	if aggregate.AssigneeID() != nil {
		assigneeID := aggregate.AssigneeID().String()
		doc.AssignedTo = &assigneeID
	}
	if aggregate.DueDate() != nil {
		dueDate := *aggregate.DueDate()
		doc.DueDate = &dueDate
	}
	for _, attachment := range aggregate.Attachments() {
		doc.Attachments = append(doc.Attachments, taskProjectionAttachment{
			FileID:   attachment.FileID().String(),
			FileName: attachment.FileName(),
			FileSize: attachment.FileSize(),
			MimeType: attachment.MimeType(),
		})
	}

	return doc, true, nil
}

func compareTaskProjectionDocuments(expected, actual *taskProjectionDocument) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	if expected.TaskID != actual.TaskID ||
		expected.ChatID != actual.ChatID ||
		expected.Title != actual.Title ||
		expected.EntityType != actual.EntityType ||
		expected.Status != actual.Status ||
		expected.Priority != actual.Priority ||
		expected.CreatedBy != actual.CreatedBy ||
		expected.Version != actual.Version ||
		!expected.CreatedAt.Equal(actual.CreatedAt) {
		return false
	}

	if !equalStringPtr(expected.Severity, actual.Severity) {
		return false
	}

	if !equalStringPtr(expected.AssignedTo, actual.AssignedTo) {
		return false
	}

	if !equalTimePtr(expected.DueDate, actual.DueDate) {
		return false
	}

	return equalTaskProjectionAttachments(expected.Attachments, actual.Attachments)
}

func equalStringPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalTimePtr(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

func equalTaskProjectionAttachments(a, b []taskProjectionAttachment) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func mapChatTypeToTaskEntityType(chatType chatdomain.Type) (taskdomain.EntityType, error) {
	switch chatType {
	case chatdomain.TypeTask:
		return taskdomain.TypeTask, nil
	case chatdomain.TypeBug:
		return taskdomain.TypeBug, nil
	case chatdomain.TypeEpic:
		return taskdomain.TypeEpic, nil
	case chatdomain.TypeDiscussion:
		return "", fmt.Errorf("unsupported chat type for task projection: %s", chatType)
	default:
		return "", fmt.Errorf("unsupported chat type for task projection: %s", chatType)
	}
}

func normalizeTaskStatus(status string) taskdomain.Status {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case "", "new", "planned":
		return taskdomain.StatusToDo
	case "investigating":
		return taskdomain.StatusInProgress
	case "fixed":
		return taskdomain.StatusInReview
	case "verified", "completed", "closed":
		return taskdomain.StatusDone
	case strings.ToLower(string(taskdomain.StatusBacklog)):
		return taskdomain.StatusBacklog
	case strings.ToLower(string(taskdomain.StatusToDo)):
		return taskdomain.StatusToDo
	case strings.ToLower(string(taskdomain.StatusInProgress)):
		return taskdomain.StatusInProgress
	case strings.ToLower(string(taskdomain.StatusInReview)):
		return taskdomain.StatusInReview
	case strings.ToLower(string(taskdomain.StatusDone)):
		return taskdomain.StatusDone
	case strings.ToLower(string(taskdomain.StatusCancelled)):
		return taskdomain.StatusCancelled
	default:
		return taskdomain.StatusToDo
	}
}

func normalizeTaskPriority(priority string) taskdomain.Priority {
	normalized := strings.ToLower(strings.TrimSpace(priority))
	switch normalized {
	case strings.ToLower(string(taskdomain.PriorityLow)):
		return taskdomain.PriorityLow
	case strings.ToLower(string(taskdomain.PriorityMedium)):
		return taskdomain.PriorityMedium
	case strings.ToLower(string(taskdomain.PriorityHigh)):
		return taskdomain.PriorityHigh
	case strings.ToLower(string(taskdomain.PriorityCritical)):
		return taskdomain.PriorityCritical
	default:
		return taskdomain.PriorityMedium
	}
}

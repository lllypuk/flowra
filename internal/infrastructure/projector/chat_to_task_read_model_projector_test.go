//nolint:testpackage // tests validate internal mapping helpers used by projection logic.
package projector

import (
	"testing"
	"time"

	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeTaskStatus(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output taskdomain.Status
	}{
		{name: "empty", input: "", output: taskdomain.StatusToDo},
		{name: "bug new", input: "New", output: taskdomain.StatusToDo},
		{name: "epic planned", input: "Planned", output: taskdomain.StatusToDo},
		{name: "bug investigating", input: "Investigating", output: taskdomain.StatusInProgress},
		{name: "bug fixed", input: "Fixed", output: taskdomain.StatusInReview},
		{name: "bug verified", input: "Verified", output: taskdomain.StatusDone},
		{name: "epic completed", input: "Completed", output: taskdomain.StatusDone},
		{name: "closed", input: "Closed", output: taskdomain.StatusDone},
		{name: "native status", input: "In Progress", output: taskdomain.StatusInProgress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.output, normalizeTaskStatus(tt.input))
		})
	}
}

func TestBuildTaskProjectionDocument_TypedChat(t *testing.T) {
	workspaceID := uuid.NewUUID()
	actorID := uuid.NewUUID()
	assigneeID := uuid.NewUUID()
	fileID := uuid.NewUUID()
	dueDate := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	chatAggregate, err := chatdomain.NewChat(workspaceID, chatdomain.TypeDiscussion, true, actorID)
	require.NoError(t, err)
	require.NoError(t, chatAggregate.ConvertToBug("Bug card", actorID))
	require.NoError(t, chatAggregate.SetPriority("Critical", actorID))
	require.NoError(t, chatAggregate.SetSeverity("Major", actorID))
	require.NoError(t, chatAggregate.AssignUser(&assigneeID, actorID))
	require.NoError(t, chatAggregate.SetDueDate(&dueDate, actorID))
	require.NoError(t, chatAggregate.AddAttachment(fileID, "report.pdf", 1024, "application/pdf", actorID))

	doc, shouldExist, err := buildTaskProjectionDocument(chatAggregate)
	require.NoError(t, err)
	require.True(t, shouldExist)
	require.NotNil(t, doc)

	assert.Equal(t, chatAggregate.ID().String(), doc.TaskID)
	assert.Equal(t, string(taskdomain.TypeBug), doc.EntityType)
	assert.Equal(t, string(taskdomain.StatusToDo), doc.Status)
	assert.Equal(t, string(taskdomain.PriorityCritical), doc.Priority)
	require.NotNil(t, doc.Severity)
	assert.Equal(t, "Major", *doc.Severity)
	require.NotNil(t, doc.AssignedTo)
	assert.Equal(t, assigneeID.String(), *doc.AssignedTo)
	require.NotNil(t, doc.DueDate)
	assert.True(t, doc.DueDate.Equal(dueDate))
	require.Len(t, doc.Attachments, 1)
	assert.Equal(t, fileID.String(), doc.Attachments[0].FileID)
	assert.Equal(t, "report.pdf", doc.Attachments[0].FileName)
}

func TestBuildTaskProjectionDocument_DiscussionChat(t *testing.T) {
	workspaceID := uuid.NewUUID()
	actorID := uuid.NewUUID()

	chatAggregate, err := chatdomain.NewChat(workspaceID, chatdomain.TypeDiscussion, true, actorID)
	require.NoError(t, err)

	doc, shouldExist, err := buildTaskProjectionDocument(chatAggregate)
	require.NoError(t, err)
	assert.False(t, shouldExist)
	assert.Nil(t, doc)
}

func TestFilterChatEvents(t *testing.T) {
	events := []event.DomainEvent{
		&stubEvent{eventType: "chat.created"},
		&stubEvent{eventType: "task.created"},
		&stubEvent{eventType: "chat.status_changed"},
	}

	filtered := filterChatEvents(events)
	require.Len(t, filtered, 2)
	assert.Equal(t, "chat.created", filtered[0].EventType())
	assert.Equal(t, "chat.status_changed", filtered[1].EventType())
}

type stubEvent struct {
	eventType string
}

func (e *stubEvent) EventType() string        { return e.eventType }
func (e *stubEvent) AggregateID() string      { return "" }
func (e *stubEvent) AggregateType() string    { return "" }
func (e *stubEvent) OccurredAt() time.Time    { return time.Time{} }
func (e *stubEvent) Version() int             { return 0 }
func (e *stubEvent) Metadata() event.Metadata { return event.Metadata{} }

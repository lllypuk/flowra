package mongodb

import (
	"testing"
	"time"

	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildChatReadModelMutation_DiscussionUnsetsTypedFields(t *testing.T) {
	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()
	c, err := chatdomain.NewChat(workspaceID, chatdomain.TypeDiscussion, false, userID)
	require.NoError(t, err)

	setDoc, unsetDoc := buildChatReadModelMutation(c)

	assert.Equal(t, string(chatdomain.TypeDiscussion), setDoc["type"])
	assert.Equal(t, "", unsetDoc["status"])
	assert.Equal(t, "", unsetDoc["priority"])
	assert.Equal(t, "", unsetDoc["assigned_to"])
	assert.Equal(t, "", unsetDoc["due_date"])
	assert.Equal(t, "", unsetDoc["severity"])
}

func TestBuildChatReadModelMutation_TaskClearsNullableFields(t *testing.T) {
	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()
	c, err := chatdomain.NewChat(workspaceID, chatdomain.TypeDiscussion, false, userID)
	require.NoError(t, err)
	require.NoError(t, c.ConvertToTask("Mutation Contract", userID))

	setDoc, unsetDoc := buildChatReadModelMutation(c)

	assert.Equal(t, c.Status(), setDoc["status"])
	assert.Equal(t, c.Priority(), setDoc["priority"])
	assert.Equal(t, "", unsetDoc["assigned_to"])
	assert.Equal(t, "", unsetDoc["due_date"])
	assert.Equal(t, "", unsetDoc["severity"])
}

func TestBuildChatReadModelMutation_BugKeepsSeverityWhenSet(t *testing.T) {
	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()
	assigneeID := uuid.NewUUID()
	dueDate := time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)

	c, err := chatdomain.NewChat(workspaceID, chatdomain.TypeDiscussion, false, userID)
	require.NoError(t, err)
	require.NoError(t, c.ConvertToBug("Mutation Bug", userID))
	require.NoError(t, c.SetSeverity("Critical", userID))
	require.NoError(t, c.AssignUser(&assigneeID, userID))
	require.NoError(t, c.SetDueDate(&dueDate, userID))

	setDoc, unsetDoc := buildChatReadModelMutation(c)

	assert.Equal(t, "Critical", setDoc["severity"])
	assert.Equal(t, assigneeID.String(), setDoc["assigned_to"])
	assert.Equal(t, dueDate, setDoc["due_date"])
	assert.NotContains(t, unsetDoc, "severity")
	assert.NotContains(t, unsetDoc, "assigned_to")
	assert.NotContains(t, unsetDoc, "due_date")
}

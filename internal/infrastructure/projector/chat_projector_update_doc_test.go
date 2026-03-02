//nolint:testpackage // Needs package access for buildChatReadModelMutation helper.
package projector

import (
	"testing"

	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildChatReadModelMutation_TaskWithoutNullableFieldsUsesUnset(t *testing.T) {
	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()
	c, err := chatdomain.NewChat(workspaceID, chatdomain.TypeDiscussion, false, userID)
	require.NoError(t, err)
	require.NoError(t, c.ConvertToTask("Projector Mutation Contract", userID))

	setDoc, unsetDoc := buildChatReadModelMutation(c)

	assert.Equal(t, string(chatdomain.TypeTask), setDoc["type"])
	assert.Equal(t, c.Status(), setDoc["status"])
	assert.Equal(t, c.Priority(), setDoc["priority"])
	assert.Empty(t, unsetDoc["assigned_to"])
	assert.Empty(t, unsetDoc["due_date"])
	assert.Empty(t, unsetDoc["severity"])
}

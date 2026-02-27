//nolint:testpackage // Tests unexported status option helpers.
package httphandler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetChatStatusOptions(t *testing.T) {
	t.Run("task statuses", func(t *testing.T) {
		assert.Equal(t, []SelectOption{
			{Value: "To Do", Label: "To Do"},
			{Value: "In Progress", Label: "In Progress"},
			{Value: "Done", Label: "Done"},
		}, getChatStatusOptions(chatTypeTask))
	})

	t.Run("bug statuses", func(t *testing.T) {
		assert.Equal(t, []SelectOption{
			{Value: "New", Label: "New"},
			{Value: "Investigating", Label: "Investigating"},
			{Value: "Fixed", Label: "Fixed"},
			{Value: "Verified", Label: "Verified"},
		}, getChatStatusOptions(chatTypeBug))
	})

	t.Run("epic statuses", func(t *testing.T) {
		assert.Equal(t, []SelectOption{
			{Value: "Planned", Label: "Planned"},
			{Value: "In Progress", Label: "In Progress"},
			{Value: "Completed", Label: "Completed"},
		}, getChatStatusOptions(chatTypeEpic))
	})

	t.Run("unknown chat type falls back to task statuses", func(t *testing.T) {
		assert.Equal(t, []SelectOption{
			{Value: "To Do", Label: "To Do"},
			{Value: "In Progress", Label: "In Progress"},
			{Value: "Done", Label: "Done"},
		}, getChatStatusOptions("unknown"))
	})
}

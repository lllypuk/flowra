package httphandler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestApplyMessageGrouping_SingleMessage(t *testing.T) {
	now := time.Now()
	messages := []MessageViewData{
		{IsBotMessage: true, CreatedAt: now},
	}

	applyMessageGrouping(messages)

	assert.True(t, messages[0].IsGroupStart)
	assert.True(t, messages[0].IsGroupEnd)
}

func TestApplyMessageGrouping_TwoMessagesWithinThreshold(t *testing.T) {
	now := time.Now()
	messages := []MessageViewData{
		{IsBotMessage: true, CreatedAt: now},
		{IsBotMessage: true, CreatedAt: now.Add(2 * time.Second)},
	}

	applyMessageGrouping(messages)

	assert.True(t, messages[0].IsGroupStart)
	assert.False(t, messages[0].IsGroupEnd)
	assert.False(t, messages[1].IsGroupStart)
	assert.True(t, messages[1].IsGroupEnd)
}

func TestApplyMessageGrouping_TwoMessagesBeyondThreshold(t *testing.T) {
	now := time.Now()
	messages := []MessageViewData{
		{IsBotMessage: true, CreatedAt: now},
		{IsBotMessage: true, CreatedAt: now.Add(10 * time.Second)},
	}

	applyMessageGrouping(messages)

	// Should be separate groups
	assert.True(t, messages[0].IsGroupStart)
	assert.True(t, messages[0].IsGroupEnd)
	assert.True(t, messages[1].IsGroupStart)
	assert.True(t, messages[1].IsGroupEnd)
}

func TestApplyMessageGrouping_ThreeConsecutiveMessages(t *testing.T) {
	now := time.Now()
	messages := []MessageViewData{
		{IsBotMessage: true, CreatedAt: now},
		{IsBotMessage: true, CreatedAt: now.Add(1 * time.Second)},
		{IsBotMessage: true, CreatedAt: now.Add(2 * time.Second)},
	}

	applyMessageGrouping(messages)

	assert.True(t, messages[0].IsGroupStart)
	assert.False(t, messages[0].IsGroupEnd)
	assert.False(t, messages[1].IsGroupStart)
	assert.False(t, messages[1].IsGroupEnd)
	assert.False(t, messages[2].IsGroupStart)
	assert.True(t, messages[2].IsGroupEnd)
}

func TestApplyMessageGrouping_UserMessageBreaksGroup(t *testing.T) {
	now := time.Now()
	messages := []MessageViewData{
		{IsBotMessage: true, CreatedAt: now},
		{IsBotMessage: false, IsSystemMessage: false, CreatedAt: now.Add(1 * time.Second)},
		{IsBotMessage: true, CreatedAt: now.Add(2 * time.Second)},
	}

	applyMessageGrouping(messages)

	// User message should not be grouped
	assert.True(t, messages[0].IsGroupStart)
	assert.True(t, messages[0].IsGroupEnd)
	assert.False(t, messages[1].IsGroupStart)
	assert.False(t, messages[1].IsGroupEnd)
	assert.True(t, messages[2].IsGroupStart)
	assert.True(t, messages[2].IsGroupEnd)
}

func TestApplyMessageGrouping_SystemAndBotMessagesTogether(t *testing.T) {
	now := time.Now()
	messages := []MessageViewData{
		{IsBotMessage: true, CreatedAt: now},
		{IsSystemMessage: true, CreatedAt: now.Add(1 * time.Second)},
		{IsBotMessage: true, CreatedAt: now.Add(2 * time.Second)},
	}

	applyMessageGrouping(messages)

	// Bot and system messages should group together
	assert.True(t, messages[0].IsGroupStart)
	assert.False(t, messages[0].IsGroupEnd)
	assert.False(t, messages[1].IsGroupStart)
	assert.False(t, messages[1].IsGroupEnd)
	assert.False(t, messages[2].IsGroupStart)
	assert.True(t, messages[2].IsGroupEnd)
}

func TestApplyMessageGrouping_MixedMessages(t *testing.T) {
	now := time.Now()
	messages := []MessageViewData{
		{IsBotMessage: true, CreatedAt: now},                                      // Group 1 start
		{IsBotMessage: true, CreatedAt: now.Add(1 * time.Second)},                 // Group 1 end
		{IsBotMessage: false, IsSystemMessage: false, CreatedAt: now.Add(2 * time.Second)}, // User message
		{IsBotMessage: true, CreatedAt: now.Add(3 * time.Second)},                 // Group 2 start
		{IsSystemMessage: true, CreatedAt: now.Add(4 * time.Second)},              // Group 2 middle
		{IsBotMessage: true, CreatedAt: now.Add(5 * time.Second)},                 // Group 2 end
		{IsBotMessage: false, IsSystemMessage: false, CreatedAt: now.Add(6 * time.Second)}, // User message
	}

	applyMessageGrouping(messages)

	// Group 1
	assert.True(t, messages[0].IsGroupStart)
	assert.False(t, messages[0].IsGroupEnd)
	assert.False(t, messages[1].IsGroupStart)
	assert.True(t, messages[1].IsGroupEnd)

	// User message (no grouping flags)
	assert.False(t, messages[2].IsGroupStart)
	assert.False(t, messages[2].IsGroupEnd)

	// Group 2
	assert.True(t, messages[3].IsGroupStart)
	assert.False(t, messages[3].IsGroupEnd)
	assert.False(t, messages[4].IsGroupStart)
	assert.False(t, messages[4].IsGroupEnd)
	assert.False(t, messages[5].IsGroupStart)
	assert.True(t, messages[5].IsGroupEnd)

	// User message (no grouping flags)
	assert.False(t, messages[6].IsGroupStart)
	assert.False(t, messages[6].IsGroupEnd)
}

func TestApplyMessageGrouping_EmptySlice(t *testing.T) {
	messages := []MessageViewData{}

	// Should not panic
	applyMessageGrouping(messages)
}

func TestIsGroupableMessage(t *testing.T) {
	tests := []struct {
		name     string
		msg      MessageViewData
		expected bool
	}{
		{
			name:     "bot message is groupable",
			msg:      MessageViewData{IsBotMessage: true},
			expected: true,
		},
		{
			name:     "system message is groupable",
			msg:      MessageViewData{IsSystemMessage: true},
			expected: true,
		},
		{
			name:     "user message is not groupable",
			msg:      MessageViewData{IsBotMessage: false, IsSystemMessage: false},
			expected: false,
		},
		{
			name:     "both bot and system is groupable",
			msg:      MessageViewData{IsBotMessage: true, IsSystemMessage: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGroupableMessage(&tt.msg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCanGroupWith(t *testing.T) {
	now := time.Now()
	threshold := 5 * time.Second

	tests := []struct {
		name     string
		current  MessageViewData
		other    MessageViewData
		expected bool
	}{
		{
			name:     "both groupable within threshold",
			current:  MessageViewData{IsBotMessage: true, CreatedAt: now},
			other:    MessageViewData{IsBotMessage: true, CreatedAt: now.Add(2 * time.Second)},
			expected: true,
		},
		{
			name:     "both groupable beyond threshold",
			current:  MessageViewData{IsBotMessage: true, CreatedAt: now},
			other:    MessageViewData{IsBotMessage: true, CreatedAt: now.Add(10 * time.Second)},
			expected: false,
		},
		{
			name:     "other not groupable",
			current:  MessageViewData{IsBotMessage: true, CreatedAt: now},
			other:    MessageViewData{IsBotMessage: false, CreatedAt: now.Add(1 * time.Second)},
			expected: false,
		},
		{
			name:     "at exact threshold",
			current:  MessageViewData{IsBotMessage: true, CreatedAt: now},
			other:    MessageViewData{IsBotMessage: true, CreatedAt: now.Add(5 * time.Second)},
			expected: true,
		},
		{
			name:     "one nanosecond over threshold",
			current:  MessageViewData{IsBotMessage: true, CreatedAt: now},
			other:    MessageViewData{IsBotMessage: true, CreatedAt: now.Add(5*time.Second + time.Nanosecond)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canGroupWith(&tt.current, &tt.other, threshold)
			assert.Equal(t, tt.expected, result)
		})
	}
}

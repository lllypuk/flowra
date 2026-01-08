package tag //nolint:testpackage // Чтобы test unexported functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsKnownTag(t *testing.T) {
	parser := NewParser()

	// Системные tags должны быть известны
	assert.True(t, parser.isKnownTag("task"))
	assert.True(t, parser.isKnownTag("bug"))
	assert.True(t, parser.isKnownTag("epic"))
	assert.True(t, parser.isKnownTag("status"))
	assert.True(t, parser.isKnownTag("assignee"))
	assert.True(t, parser.isKnownTag("priority"))
	assert.True(t, parser.isKnownTag("due"))
	assert.True(t, parser.isKnownTag("title"))
	assert.True(t, parser.isKnownTag("severity"))

	// Неизвестные tags
	assert.False(t, parser.isKnownTag("unknown"))
	assert.False(t, parser.isKnownTag("hashtag"))
	assert.False(t, parser.isKnownTag("random"))
}

func TestGetTagDefinition(t *testing.T) {
	parser := NewParser()

	t.Run("known tag", func(t *testing.T) {
		def, exists := parser.GetTagDefinition("task")
		assert.True(t, exists)
		assert.Equal(t, "task", def.Name)
		assert.True(t, def.RequiresValue)
		assert.Equal(t, ValueTypeString, def.ValueType)
	})

	t.Run("unknown tag", func(t *testing.T) {
		_, exists := parser.GetTagDefinition("unknown")
		assert.False(t, exists)
	})
}

func TestTagDefinitions(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name          string
		tagName       string
		requiresValue bool
		valueType     ValueType
		hasValidator  bool
	}{
		// Entity Creation Tags
		{
			name:          "task",
			tagName:       "task",
			requiresValue: true,
			valueType:     ValueTypeString,
			hasValidator:  true,
		},
		{
			name:          "bug",
			tagName:       "bug",
			requiresValue: true,
			valueType:     ValueTypeString,
			hasValidator:  true,
		},
		{
			name:          "epic",
			tagName:       "epic",
			requiresValue: true,
			valueType:     ValueTypeString,
			hasValidator:  true,
		},
		// Entity Management Tags
		{
			name:          "status",
			tagName:       "status",
			requiresValue: true,
			valueType:     ValueTypeEnum,
			hasValidator:  true,
		},
		{
			name:          "assignee",
			tagName:       "assignee",
			requiresValue: false, // опциональное value
			valueType:     ValueTypeUsername,
			hasValidator:  true,
		},
		{
			name:          "priority",
			tagName:       "priority",
			requiresValue: true,
			valueType:     ValueTypeEnum,
			hasValidator:  true,
		},
		{
			name:          "due",
			tagName:       "due",
			requiresValue: false, // опциональное value
			valueType:     ValueTypeDate,
			hasValidator:  true,
		},
		{
			name:          "title",
			tagName:       "title",
			requiresValue: true,
			valueType:     ValueTypeString,
			hasValidator:  true,
		},
		// Bug-Specific Tags
		{
			name:          "severity",
			tagName:       "severity",
			requiresValue: true,
			valueType:     ValueTypeEnum,
			hasValidator:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, exists := parser.GetTagDefinition(tt.tagName)
			assert.True(t, exists, "tag should be registered")
			assert.Equal(t, tt.tagName, def.Name)
			assert.Equal(t, tt.requiresValue, def.RequiresValue)
			assert.Equal(t, tt.valueType, def.ValueType)

			if tt.hasValidator {
				assert.NotNil(t, def.Validator, "validator should be set")
			}
		})
	}
}

func TestValueTypeString(t *testing.T) {
	tests := []struct {
		valueType ValueType
		expected  string
	}{
		{ValueTypeString, "String"},
		{ValueTypeUsername, "Username"},
		{ValueTypeDate, "Date"},
		{ValueTypeEnum, "Enum"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.valueType.String())
		})
	}
}

func TestAllowedValuesForEnumTags(t *testing.T) {
	parser := NewParser()

	t.Run("priority has allowed values", func(t *testing.T) {
		def, _ := parser.GetTagDefinition("priority")
		assert.Equal(t, []string{"High", "Medium", "Low"}, def.AllowedValues)
	})

	t.Run("severity has allowed values", func(t *testing.T) {
		def, _ := parser.GetTagDefinition("severity")
		assert.Equal(t, []string{"Critical", "Major", "Minor", "Trivial"}, def.AllowedValues)
	})

	t.Run("status has nil allowed values (context-dependent)", func(t *testing.T) {
		def, _ := parser.GetTagDefinition("status")
		assert.nil(t, def.AllowedValues, "status values depend on entity type")
	})
}

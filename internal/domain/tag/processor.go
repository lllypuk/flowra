package tag

import (
	"strings"

	"github.com/google/uuid"
)

// Processor handles parsed tags and generates commands
type Processor struct {
	parser *Parser
}

// NewProcessor creates a new tag processor
func NewProcessor() *Processor {
	return &Processor{
		parser: NewParser(),
	}
}

// ProcessMessage handles message with tags and returns result
// currentEntityType - type of current active entity in chat ("Task", "Bug", "Epic")
// can be empty string if no active entity
// if message creates a new entity, Entity Management Tags apply to it
func (p *Processor) ProcessMessage(
	chatID uuid.UUID,
	message string,
	currentEntityType string,
) *ProcessingResult {
	// parse message
	parseResult := p.parser.Parse(message)

	// process tags
	result := p.ProcessTags(chatID, parseResult.Tags, currentEntityType)
	result.OriginalMessage = message
	result.PlainText = parseResult.PlainText

	return result
}

// ProcessTags handles tags and returns result
// currentEntityType - type of current active entity in chat ("Task", "Bug", "Epic")
// can be empty string if no active entity
// if message creates a new entity, Entity Management Tags apply to it
//
//nolint:gocognit,funlen // Complexity justified: sequential tag processing logic
func (p *Processor) ProcessTags(
	chatID uuid.UUID,
	parsedTags []ParsedTag,
	currentEntityType string,
) *ProcessingResult {
	result := &ProcessingResult{
		AppliedTags: []TagApplication{},
		Errors:      []TagError{},
	}

	// track entity type for Entity Management Tags
	entityType := currentEntityType

	for _, tag := range parsedTags {
		switch tag.Key {
		// ====== Entity Creation Tags ======
		case "task":
			if err := ValidateEntityCreation("task", tag.Value); err != nil {
				result.Errors = append(result.Errors, TagError{
					TagKey:   tag.Key,
					TagValue: tag.Value,
					Error:    err,
					Severity: ErrorSeverityError,
				})
				continue
			}
			cmd := CreateTaskCommand{
				ChatID: chatID,
				Title:  strings.TrimSpace(tag.Value),
			}
			result.AppliedTags = append(result.AppliedTags, TagApplication{
				TagKey:   tag.Key,
				TagValue: tag.Value,
				Command:  cmd,
				Success:  true,
			})
			// if created an entity, use its type for subsequent tags
			entityType = "Task"

		case "bug":
			if err := ValidateEntityCreation("bug", tag.Value); err != nil {
				result.Errors = append(result.Errors, TagError{
					TagKey:   tag.Key,
					TagValue: tag.Value,
					Error:    err,
					Severity: ErrorSeverityError,
				})
				continue
			}
			cmd := CreateBugCommand{
				ChatID: chatID,
				Title:  strings.TrimSpace(tag.Value),
			}
			result.AppliedTags = append(result.AppliedTags, TagApplication{
				TagKey:   tag.Key,
				TagValue: tag.Value,
				Command:  cmd,
				Success:  true,
			})
			entityType = "Bug"

		case "epic":
			if err := ValidateEntityCreation("epic", tag.Value); err != nil {
				result.Errors = append(result.Errors, TagError{
					TagKey:   tag.Key,
					TagValue: tag.Value,
					Error:    err,
					Severity: ErrorSeverityError,
				})
				continue
			}
			cmd := CreateEpicCommand{
				ChatID: chatID,
				Title:  strings.TrimSpace(tag.Value),
			}
			result.AppliedTags = append(result.AppliedTags, TagApplication{
				TagKey:   tag.Key,
				TagValue: tag.Value,
				Command:  cmd,
				Success:  true,
			})
			entityType = "Epic"

		// ====== Entity Management Tags ======
		case "status":
			if entityType == "" {
				result.Errors = append(result.Errors, TagError{
					TagKey:   tag.Key,
					TagValue: tag.Value,
					Error:    ErrNoActiveEntity,
					Severity: ErrorSeverityError,
				})
				continue
			}
			if err := ValidateStatus(entityType, tag.Value); err != nil {
				result.Errors = append(result.Errors, TagError{
					TagKey:   tag.Key,
					TagValue: tag.Value,
					Error:    err,
					Severity: ErrorSeverityError,
				})
				continue
			}
			cmd := ChangeStatusCommand{
				ChatID: chatID,
				Status: tag.Value,
			}
			result.AppliedTags = append(result.AppliedTags, TagApplication{
				TagKey:   tag.Key,
				TagValue: tag.Value,
				Command:  cmd,
				Success:  true,
			})

		case "assignee":
			if err := validateUsername(tag.Value); err != nil {
				result.Errors = append(result.Errors, TagError{
					TagKey:   tag.Key,
					TagValue: tag.Value,
					Error:    err,
					Severity: ErrorSeverityError,
				})
				continue
			}
			cmd := AssignUserCommand{
				ChatID:   chatID,
				Username: tag.Value,
				UserID:   nil, // will be resolved at service level
			}
			result.AppliedTags = append(result.AppliedTags, TagApplication{
				TagKey:   tag.Key,
				TagValue: tag.Value,
				Command:  cmd,
				Success:  true,
			})

		case "priority":
			if err := validatePriority(tag.Value); err != nil {
				result.Errors = append(result.Errors, TagError{
					TagKey:   tag.Key,
					TagValue: tag.Value,
					Error:    err,
					Severity: ErrorSeverityError,
				})
				continue
			}
			cmd := ChangePriorityCommand{
				ChatID:   chatID,
				Priority: tag.Value,
			}
			result.AppliedTags = append(result.AppliedTags, TagApplication{
				TagKey:   tag.Key,
				TagValue: tag.Value,
				Command:  cmd,
				Success:  true,
			})

		case "due":
			dueDate, err := ValidateDueDate(tag.Value)
			if err != nil {
				result.Errors = append(result.Errors, TagError{
					TagKey:   tag.Key,
					TagValue: tag.Value,
					Error:    err,
					Severity: ErrorSeverityError,
				})
				continue
			}
			cmd := SetDueDateCommand{
				ChatID:  chatID,
				DueDate: dueDate,
			}
			result.AppliedTags = append(result.AppliedTags, TagApplication{
				TagKey:   tag.Key,
				TagValue: tag.Value,
				Command:  cmd,
				Success:  true,
			})

		case "title":
			if err := ValidateTitle(tag.Value); err != nil {
				result.Errors = append(result.Errors, TagError{
					TagKey:   tag.Key,
					TagValue: tag.Value,
					Error:    err,
					Severity: ErrorSeverityError,
				})
				continue
			}
			cmd := ChangeTitleCommand{
				ChatID: chatID,
				Title:  strings.TrimSpace(tag.Value),
			}
			result.AppliedTags = append(result.AppliedTags, TagApplication{
				TagKey:   tag.Key,
				TagValue: tag.Value,
				Command:  cmd,
				Success:  true,
			})

		case "severity":
			if err := validateSeverity(tag.Value); err != nil {
				result.Errors = append(result.Errors, TagError{
					TagKey:   tag.Key,
					TagValue: tag.Value,
					Error:    err,
					Severity: ErrorSeverityError,
				})
				continue
			}
			cmd := SetSeverityCommand{
				ChatID:   chatID,
				Severity: tag.Value,
			}
			result.AppliedTags = append(result.AppliedTags, TagApplication{
				TagKey:   tag.Key,
				TagValue: tag.Value,
				Command:  cmd,
				Success:  true,
			})

		// ====== Task 007a: Participant Management Tags ======
		case "invite":
			if err := validateUsername(tag.Value); err != nil {
				result.Errors = append(result.Errors, TagError{
					TagKey:   tag.Key,
					TagValue: tag.Value,
					Error:    err,
					Severity: ErrorSeverityError,
				})
				continue
			}
			cmd := InviteUserCommand{
				ChatID:   chatID,
				Username: tag.Value,
			}
			result.AppliedTags = append(result.AppliedTags, TagApplication{
				TagKey:   tag.Key,
				TagValue: tag.Value,
				Command:  cmd,
				Success:  true,
			})

		case "remove":
			if err := validateUsername(tag.Value); err != nil {
				result.Errors = append(result.Errors, TagError{
					TagKey:   tag.Key,
					TagValue: tag.Value,
					Error:    err,
					Severity: ErrorSeverityError,
				})
				continue
			}
			cmd := RemoveUserCommand{
				ChatID:   chatID,
				Username: tag.Value,
			}
			result.AppliedTags = append(result.AppliedTags, TagApplication{
				TagKey:   tag.Key,
				TagValue: tag.Value,
				Command:  cmd,
				Success:  true,
			})

		// ====== Task 007a: Chat Lifecycle Tags ======
		case "close":
			// Cannot close a discussion
			if entityType == "" {
				result.Errors = append(result.Errors, TagError{
					TagKey:   tag.Key,
					Error:    ErrNoActiveEntity,
					Severity: ErrorSeverityError,
				})
				continue
			}
			cmd := CloseChatCommand{ChatID: chatID}
			result.AppliedTags = append(result.AppliedTags, TagApplication{
				TagKey:  tag.Key,
				Command: cmd,
				Success: true,
			})

		case "reopen":
			cmd := ReopenChatCommand{ChatID: chatID}
			result.AppliedTags = append(result.AppliedTags, TagApplication{
				TagKey:  tag.Key,
				Command: cmd,
				Success: true,
			})

		case "delete":
			cmd := DeleteChatCommand{ChatID: chatID}
			result.AppliedTags = append(result.AppliedTags, TagApplication{
				TagKey:  tag.Key,
				Command: cmd,
				Success: true,
			})
		}
	}

	return result
}

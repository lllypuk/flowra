package tag

import (
	"strings"

	"github.com/google/uuid"
)

// Processor обрабатывает распарсенные теги и генерирует команды
type Processor struct {
	parser *Parser
}

// NewProcessor создает новый процессор тегов
func NewProcessor() *Processor {
	return &Processor{
		parser: NewParser(),
	}
}

// ProcessMessage обрабатывает сообщение с тегами и возвращает результат
// currentEntityType - тип текущей активной сущности в чате ("Task", "Bug", "Epic")
// Может быть пустой строкой, если нет активной сущности
// Если в сообщении создается новая сущность, Entity Management Tags применяются к ней
func (p *Processor) ProcessMessage(
	chatID uuid.UUID,
	message string,
	currentEntityType string,
) *ProcessingResult {
	// Парсим сообщение
	parseResult := p.parser.Parse(message)

	// Обрабатываем теги
	result := p.ProcessTags(chatID, parseResult.Tags, currentEntityType)
	result.OriginalMessage = message
	result.PlainText = parseResult.PlainText

	return result
}

// ProcessTags обрабатывает теги и возвращает результат
// currentEntityType - тип текущей активной сущности в чате ("Task", "Bug", "Epic")
// Может быть пустой строкой, если нет активной сущности
// Если в сообщении создается новая сущность, Entity Management Tags применяются к ней
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

	// Отслеживаем тип сущности для Entity Management Tags
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
			// Если создали сущность, используем ее тип для последующих тегов
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
				UserID:   nil, // Будет резолвлен на уровне service
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
		}
	}

	return result
}

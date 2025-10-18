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

// ProcessTags обрабатывает теги и возвращает команды и ошибки
// currentEntityType - тип текущей активной сущности в чате ("Task", "Bug", "Epic")
// Может быть пустой строкой, если нет активной сущности
// Если в сообщении создается новая сущность, Entity Management Tags применяются к ней
//
//nolint:gocognit,funlen // Complexity justified: sequential tag processing logic
func (p *Processor) ProcessTags(
	chatID uuid.UUID,
	parsedTags []ParsedTag,
	currentEntityType string,
) ([]Command, []error) {
	var commands []Command
	var errors []error

	// Отслеживаем тип сущности для Entity Management Tags
	entityType := currentEntityType

	for _, tag := range parsedTags {
		switch tag.Key {
		// ====== Entity Creation Tags ======
		case "task":
			if err := ValidateEntityCreation("task", tag.Value); err != nil {
				errors = append(errors, err)
				continue
			}
			commands = append(commands, CreateTaskCommand{
				ChatID: chatID,
				Title:  strings.TrimSpace(tag.Value),
			})
			// Если создали сущность, используем ее тип для последующих тегов
			entityType = "Task"

		case "bug":
			if err := ValidateEntityCreation("bug", tag.Value); err != nil {
				errors = append(errors, err)
				continue
			}
			commands = append(commands, CreateBugCommand{
				ChatID: chatID,
				Title:  strings.TrimSpace(tag.Value),
			})
			entityType = "Bug"

		case "epic":
			if err := ValidateEntityCreation("epic", tag.Value); err != nil {
				errors = append(errors, err)
				continue
			}
			commands = append(commands, CreateEpicCommand{
				ChatID: chatID,
				Title:  strings.TrimSpace(tag.Value),
			})
			entityType = "Epic"

		// ====== Entity Management Tags ======
		case "status":
			if entityType == "" {
				errors = append(errors, ErrNoActiveEntity)
				continue
			}
			if err := ValidateStatus(entityType, tag.Value); err != nil {
				errors = append(errors, err)
				continue
			}
			commands = append(commands, ChangeStatusCommand{
				ChatID: chatID,
				Status: tag.Value,
			})

		case "assignee":
			if err := validateUsername(tag.Value); err != nil {
				errors = append(errors, err)
				continue
			}
			commands = append(commands, AssignUserCommand{
				ChatID:   chatID,
				Username: tag.Value,
				UserID:   nil, // Будет резолвлен на уровне service
			})

		case "priority":
			if err := validatePriority(tag.Value); err != nil {
				errors = append(errors, err)
				continue
			}
			commands = append(commands, ChangePriorityCommand{
				ChatID:   chatID,
				Priority: tag.Value,
			})

		case "due":
			dueDate, err := ValidateDueDate(tag.Value)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			commands = append(commands, SetDueDateCommand{
				ChatID:  chatID,
				DueDate: dueDate,
			})

		case "title":
			if err := ValidateTitle(tag.Value); err != nil {
				errors = append(errors, err)
				continue
			}
			commands = append(commands, ChangeTitleCommand{
				ChatID: chatID,
				Title:  strings.TrimSpace(tag.Value),
			})

		case "severity":
			if err := validateSeverity(tag.Value); err != nil {
				errors = append(errors, err)
				continue
			}
			commands = append(commands, SetSeverityCommand{
				ChatID:   chatID,
				Severity: tag.Value,
			})
		}
	}

	return commands, errors
}

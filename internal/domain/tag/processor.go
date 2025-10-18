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
func (p *Processor) ProcessTags(chatID uuid.UUID, parsedTags []ParsedTag) ([]Command, []error) {
	var commands []Command
	var errors []error

	for _, tag := range parsedTags {
		switch tag.Key {
		case "task":
			if err := ValidateEntityCreation("task", tag.Value); err != nil {
				errors = append(errors, err)
				continue
			}
			commands = append(commands, CreateTaskCommand{
				ChatID: chatID,
				Title:  strings.TrimSpace(tag.Value),
			})

		case "bug":
			if err := ValidateEntityCreation("bug", tag.Value); err != nil {
				errors = append(errors, err)
				continue
			}
			commands = append(commands, CreateBugCommand{
				ChatID: chatID,
				Title:  strings.TrimSpace(tag.Value),
			})

		case "epic":
			if err := ValidateEntityCreation("epic", tag.Value); err != nil {
				errors = append(errors, err)
				continue
			}
			commands = append(commands, CreateEpicCommand{
				ChatID: chatID,
				Title:  strings.TrimSpace(tag.Value),
			})
		}
	}

	return commands, errors
}

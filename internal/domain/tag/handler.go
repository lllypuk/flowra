package tag

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lllypuk/teams-up/internal/domain/chat"
	"github.com/lllypuk/teams-up/internal/domain/message"
	domainUUID "github.com/lllypuk/teams-up/internal/domain/uuid"
)

const (
	entityTypeTask = "Task"
	entityTypeBug  = "Bug"
	entityTypeEpic = "Epic"
)

// Handler обрабатывает сообщения с тегами
type Handler struct {
	processor   *Processor
	executor    *CommandExecutor
	messageRepo MessageRepository
	chatRepo    ChatRepository
}

// NewHandler создает новый Handler
func NewHandler(
	processor *Processor,
	executor *CommandExecutor,
	messageRepo MessageRepository,
	chatRepo ChatRepository,
) *Handler {
	return &Handler{
		processor:   processor,
		executor:    executor,
		messageRepo: messageRepo,
		chatRepo:    chatRepo,
	}
}

// HandleMessageWithTags обрабатывает сообщение с тегами
func (h *Handler) HandleMessageWithTags(
	ctx context.Context,
	chatID uuid.UUID,
	authorID uuid.UUID,
	content string,
) error {
	// Конвертация UUID
	domainChatID := domainUUID.FromGoogleUUID(chatID)

	// 1. Получение контекста чата
	c, err := h.chatRepo.Load(ctx, domainChatID)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}

	// Определяем текущий тип entity для валидации
	currentEntityType := h.getEntityType(c)

	// 2. Обработка тегов через Processor
	result := h.processor.ProcessMessage(chatID, content, currentEntityType)

	// 3. Сохранение сообщения пользователя
	msg, err := message.NewMessage(
		domainChatID,
		domainUUID.FromGoogleUUID(authorID),
		result.PlainText,    // текст без тегов
		domainUUID.UUID(""), // не thread
	)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	if err = h.messageRepo.Save(ctx, msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// 4. Выполнение команд
	executionErrors := h.executeCommands(ctx, result.AppliedTags, authorID)

	// 5. Добавление ошибок выполнения к результату
	result.Errors = append(result.Errors, executionErrors...)

	// 6. Генерация и отправка bot response
	if botResponse := result.GenerateBotResponse(); botResponse != "" {
		if sendErr := h.sendBotResponse(ctx, chatID, botResponse); sendErr != nil {
			// Логируем, но не фейлим весь процесс
			// TODO: add proper logging
			_ = sendErr // временно игнорируем ошибку отправки bot response
		}
	}

	return nil
}

// executeCommands выполняет все команды из результата обработки
func (h *Handler) executeCommands(
	ctx context.Context,
	applications []TagApplication,
	actorID uuid.UUID,
) []TagError {
	var errors []TagError

	for _, app := range applications {
		if app.Command == nil {
			continue
		}

		if err := h.executor.Execute(ctx, app.Command, actorID); err != nil {
			errors = append(errors, TagError{
				TagKey:   app.TagKey,
				TagValue: app.TagValue,
				Error:    err,
				Severity: ErrorSeverityError,
			})
		}
	}

	return errors
}

// sendBotResponse отправляет bot response в чат
func (h *Handler) sendBotResponse(ctx context.Context, chatID uuid.UUID, response string) error {
	domainChatID := domainUUID.FromGoogleUUID(chatID)

	// Создаем системное сообщение от бота
	// TODO: использовать настоящий bot user ID вместо пустого
	botMessage, err := message.NewMessage(
		domainChatID,
		domainUUID.UUID("00000000-0000-0000-0000-000000000000"), // System bot ID
		response,
		domainUUID.UUID(""), // не thread
	)
	if err != nil {
		return fmt.Errorf("failed to create bot message: %w", err)
	}

	if err = h.messageRepo.Save(ctx, botMessage); err != nil {
		return fmt.Errorf("failed to save bot message: %w", err)
	}

	return nil
}

// getEntityType возвращает тип entity для валидации
func (h *Handler) getEntityType(c *chat.Chat) string {
	switch c.Type() {
	case chat.TypeTask:
		return entityTypeTask
	case chat.TypeBug:
		return entityTypeBug
	case chat.TypeEpic:
		return entityTypeEpic
	case chat.TypeDiscussion:
		return ""
	default:
		return ""
	}
}

package tag

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/message"
	domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

const (
	entityTypeTask = "Task"
	entityTypeBug  = "Bug"
	entityTypeEpic = "Epic"
)

// Handler handles messages с тегами
type Handler struct {
	processor   *Processor
	executor    *CommandExecutor
	messageRepo MessageRepository
	chatRepo    ChatRepository
}

// NewHandler creates New Handler
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

// HandleMessageWithTags handles message с тегами
func (h *Handler) HandleMessageWithTags(
	ctx context.Context,
	chatID uuid.UUID,
	authorID uuid.UUID,
	content string,
) error {
	// Конвертация UUID
	domainChatID := domainUUID.FromGoogleUUID(chatID)

	// 1. retrieval context chat
	c, err := h.chatRepo.Load(ctx, domainChatID)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}

	// Определяем текущий type entity for validации
	currentEntityType := h.getEntityType(c)

	// 2. handling тегов via Processor
	result := h.processor.ProcessMessage(chatID, content, currentEntityType)

	// 3. storage messages user
	msg, err := message.NewMessage(
		domainChatID,
		domainUUID.FromGoogleUUID(authorID),
		result.PlainText,    // text без тегов
		domainUUID.UUID(""), // not thread
	)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	if err = h.messageRepo.Save(ctx, msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// 4. performing commands
	executionErrors := h.executeCommands(ctx, result.AppliedTags, authorID)

	// 5. Adding errors выполнения to результату
	result.Errors = append(result.Errors, executionErrors...)

	// 6. Генерация and sendа bot response
	if botResponse := result.GenerateBotResponse(); botResponse != "" {
		if sendErr := h.sendBotResponse(ctx, chatID, botResponse); sendErr != nil {
			// Логируем, но not фейлим весь процесс
			// TODO: add proper logging
			_ = sendErr // временно игнорируем error sendи bot response
		}
	}

	return nil
}

// executeCommands performs all commands from result обworkки
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
				severity: ErrorSeverityError,
			})
		}
	}

	return errors
}

// sendBotResponse отправляет bot response in chat
func (h *Handler) sendBotResponse(ctx context.Context, chatID uuid.UUID, response string) error {
	domainChatID := domainUUID.FromGoogleUUID(chatID)

	// Creating системное message от бота
	// TODO: исuserь настоящий bot user ID вместо пустого
	botMessage, err := message.NewMessage(
		domainChatID,
		domainUUID.UUID("00000000-0000-0000-0000-000000000000"), // System bot ID
		response,
		domainUUID.UUID(""), // not thread
	)
	if err != nil {
		return fmt.Errorf("failed to create bot message: %w", err)
	}

	if err = h.messageRepo.Save(ctx, botMessage); err != nil {
		return fmt.Errorf("failed to save bot message: %w", err)
	}

	return nil
}

// getEntityType returns type entity for validации
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

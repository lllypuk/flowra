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

// Handler handles messages s tegami
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

// HandleMessageWithTags handles message s tegami
func (h *Handler) HandleMessageWithTags(
	ctx context.Context,
	chatID uuid.UUID,
	authorID uuid.UUID,
	content string,
) error {
	// konvertatsiya UUID
	domainChatID := domainUUID.FromGoogleUUID(chatID)

	// 1. retrieval context chat
	c, err := h.chatRepo.Load(ctx, domainChatID)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}

	// opredelyaem tekuschiy type entity for valid
	currentEntityType := h.getEntityType(c)

	// 2. handling tegov via Processor
	result := h.processor.ProcessMessage(chatID, content, currentEntityType)

	// 3. storage messages user
	msg, err := message.NewMessage(
		domainChatID,
		domainUUID.FromGoogleUUID(authorID),
		result.PlainText,    // text bez tegov
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

	// 5. Adding errors vypolneniya to rezultatu
	result.Errors = append(result.Errors, executionErrors...)

	// 6. generatsiya and send bot response
	if botResponse := result.GenerateBotResponse(); botResponse != "" {
		if sendErr := h.sendBotResponse(ctx, chatID, botResponse); sendErr != nil {
			// logiruem, no not feylim ves protsess
			// TODO: add proper logging
			_ = sendErr // vremenno ignoriruem error send bot response
		}
	}

	return nil
}

// executeCommands performs all commands from result work
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

// sendBotResponse otpravlyaet bot response in chat
func (h *Handler) sendBotResponse(ctx context.Context, chatID uuid.UUID, response string) error {
	domainChatID := domainUUID.FromGoogleUUID(chatID)

	// Creating sistemnoe message ot bota
	// TODO: user nastoyaschiy bot user ID vmesto pustogo
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

// getEntityType returns type entity for valid
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

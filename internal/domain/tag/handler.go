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

// Handler handles messages with tags
type Handler struct {
	processor   *Processor
	executor    *CommandExecutor
	messageRepo MessageRepository
	chatRepo    ChatRepository
}

// NewHandler creates a new Handler
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

// HandleMessageWithTags handles a message with tags
func (h *Handler) HandleMessageWithTags(
	ctx context.Context,
	chatID uuid.UUID,
	authorID uuid.UUID,
	content string,
) error {
	// convert UUID
	domainChatID := domainUUID.FromGoogleUUID(chatID)

	// 1. retrieve chat context
	c, err := h.chatRepo.Load(ctx, domainChatID)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}

	// determine current entity type for validation
	currentEntityType := h.getEntityType(c)

	// 2. process tags via Processor
	result := h.processor.ProcessMessage(chatID, content, currentEntityType)

	// 3. save user message
	msg, err := message.NewMessage(
		domainChatID,
		domainUUID.FromGoogleUUID(authorID),
		result.PlainText,    // text without tags
		domainUUID.UUID(""), // not a thread
	)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	if err = h.messageRepo.Save(ctx, msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// 4. execute commands
	executionErrors := h.executeCommands(ctx, result.AppliedTags, authorID)

	// 5. add execution errors to result
	result.Errors = append(result.Errors, executionErrors...)

	// 6. generate and send bot response
	if botResponse := result.GenerateBotResponse(); botResponse != "" {
		if sendErr := h.sendBotResponse(ctx, chatID, botResponse); sendErr != nil {
			// log but don't fail the entire process
			// TODO: add proper logging
			_ = sendErr // temporarily ignore bot response send error
		}
	}

	return nil
}

// executeCommands executes all commands from processing result
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

// sendBotResponse sends bot response to chat
func (h *Handler) sendBotResponse(ctx context.Context, chatID uuid.UUID, response string) error {
	domainChatID := domainUUID.FromGoogleUUID(chatID)

	// create system message from bot
	// TODO: use actual bot user ID instead of empty
	botMessage, err := message.NewMessage(
		domainChatID,
		domainUUID.UUID("00000000-0000-0000-0000-000000000000"), // System bot ID
		response,
		domainUUID.UUID(""), // not a thread
	)
	if err != nil {
		return fmt.Errorf("failed to create bot message: %w", err)
	}

	if err = h.messageRepo.Save(ctx, botMessage); err != nil {
		return fmt.Errorf("failed to save bot message: %w", err)
	}

	return nil
}

// getEntityType returns entity type for validation
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

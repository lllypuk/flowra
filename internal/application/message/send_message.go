package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	messagedomain "github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/tag"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// ChatRepository defines interface for access to chats (consumer-side interface)
type ChatRepository interface {
	FindByID(ctx context.Context, chatID uuid.UUID) (*chatapp.ReadModel, error)
}

// SendMessageUseCase handles sending messages
type SendMessageUseCase struct {
	messageRepo  Repository
	chatRepo     ChatRepository
	eventBus     event.Bus
	tagProcessor *tag.Processor       // Tag processor for parsing tags from message content
	tagExecutor  *tag.CommandExecutor // Tag executor for executing tag commands
}

// NewSendMessageUseCase creates New SendMessageUseCase
func NewSendMessageUseCase(
	messageRepo Repository,
	chatRepo ChatRepository,
	eventBus event.Bus,
	tagProcessor *tag.Processor,
	tagExecutor *tag.CommandExecutor,
) *SendMessageUseCase {
	return &SendMessageUseCase{
		messageRepo:  messageRepo,
		chatRepo:     chatRepo,
		eventBus:     eventBus,
		tagProcessor: tagProcessor,
		tagExecutor:  tagExecutor,
	}
}

// Execute performs sending messages
func (uc *SendMessageUseCase) Execute(
	ctx context.Context,
	cmd SendMessageCommand,
) (Result, error) {
	// 1. validation
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// 2. check access to chat
	chatReadModel, err := uc.chatRepo.FindByID(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, ErrChatNotFound
	}

	// check that user is a participant of chat
	if !uc.isParticipant(chatReadModel, cmd.AuthorID) {
		return Result{}, ErrNotChatParticipant
	}

	// 3. check parent message (if it is reply)
	if !cmd.ParentMessageID.IsZero() {
		parent, parentErr := uc.messageRepo.FindByID(ctx, cmd.ParentMessageID)
		if parentErr != nil {
			return Result{}, ErrParentNotFound
		}
		// check that parent is in the same chat
		if parent.ChatID() != cmd.ChatID {
			return Result{}, ErrParentInDifferentChat
		}
	}

	// 4. create message
	msg, err := messagedomain.NewMessage(
		cmd.ChatID,
		cmd.AuthorID,
		cmd.Content,
		cmd.ParentMessageID,
	)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create message: %w", err)
	}

	// 5. save
	if saveErr := uc.messageRepo.Save(ctx, msg); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save message: %w", saveErr)
	}

	// 6. publish event (for WebSocket broadcast)
	evt := messagedomain.NewCreated(
		msg.ID(),
		cmd.ChatID,
		cmd.AuthorID,
		cmd.Content,
		cmd.ParentMessageID,
		event.Metadata{
			UserID:    cmd.AuthorID.String(),
			Timestamp: msg.CreatedAt(),
		},
	)
	// not critical, message already saved
	// TODO: log error
	_ = uc.eventBus.Publish(ctx, evt)

	// 7. async tag handling (do not block response)
	// Use background context since HTTP request context will be canceled after response
	if uc.tagProcessor != nil && uc.tagExecutor != nil {
		go uc.processTagsAsync(context.Background(), msg, cmd.AuthorID, chatReadModel.Type)
	}

	return Result{
		Value: msg,
	}, nil
}

func (uc *SendMessageUseCase) validate(cmd SendMessageCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("content", cmd.Content); err != nil {
		return ErrEmptyContent
	}
	if len(cmd.Content) > MaxContentLength {
		return ErrContentTooLong
	}
	if err := appcore.ValidateUUID("authorID", cmd.AuthorID); err != nil {
		return err
	}
	return nil
}

func (uc *SendMessageUseCase) isParticipant(chatReadModel *chatapp.ReadModel, userID uuid.UUID) bool {
	for _, p := range chatReadModel.Participants {
		if p.UserID() == userID {
			return true
		}
	}
	return false
}

// processTagsAsync handles tags in message content asynchronously
// executed in goroutine to not block main response
func (uc *SendMessageUseCase) processTagsAsync(
	ctx context.Context,
	msg *messagedomain.Message,
	authorID uuid.UUID,
	chatType chat.Type,
) {
	// Convert domain UUID to google UUID for processor
	chatIDGoogle, err := msg.ChatID().ToGoogleUUID()
	if err != nil {
		// UUID conversion error - ignore
		return
	}

	// Determine current entity type based on chat type
	// The tag processor expects "Task", "Bug", "Epic" or empty string
	entityType := chatTypeToEntityType(chatType)

	// Parse and process tags from message content
	processingResult := uc.tagProcessor.ProcessMessage(chatIDGoogle, msg.Content(), entityType)
	if len(processingResult.AppliedTags) == 0 {
		// No successfully applied tags - exit
		return
	}

	// Convert domain UUID to google UUID for executor
	authorIDGoogle, convErr := authorID.ToGoogleUUID()
	if convErr != nil {
		// UUID conversion error - exit
		return
	}

	// Execute commands
	for _, tagApp := range processingResult.AppliedTags {
		_ = uc.tagExecutor.Execute(ctx, tagApp.Command, authorIDGoogle)
		// TODO: send notification about error or create reply with bot
		// For now just ignore the error
	}

	// TODO: format results via tag.Formatter and send reply
}

// chatTypeToEntityType converts chat.Type to entity type string expected by tag processor.
// Returns "Task", "Bug", "Epic" for task-like chats, or empty string for discussions.
func chatTypeToEntityType(chatType chat.Type) string {
	switch chatType {
	case chat.TypeTask:
		return "Task"
	case chat.TypeBug:
		return "Bug"
	case chat.TypeEpic:
		return "Epic"
	case chat.TypeDiscussion:
		return ""
	default:
		return ""
	}
}

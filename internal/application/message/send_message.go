package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	messagedomain "github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/tag"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// ChatRepository определяет интерфейс для доступа к чатам (consumer-side interface)
type ChatRepository interface {
	FindByID(ctx context.Context, chatID uuid.UUID) (*chatapp.ReadModel, error)
}

// SendMessageUseCase обрабатывает отправку сообщения
type SendMessageUseCase struct {
	messageRepo  Repository
	chatRepo     ChatRepository
	eventBus     event.Bus
	tagProcessor *tag.Processor       // Tag processor for parsing tags from message content
	tagExecutor  *tag.CommandExecutor // Tag executor for executing tag commands
}

// NewSendMessageUseCase создает новый SendMessageUseCase
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

// Execute выполняет отправку сообщения
func (uc *SendMessageUseCase) Execute(
	ctx context.Context,
	cmd SendMessageCommand,
) (Result, error) {
	// 1. Валидация
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Проверка доступа к чату
	chatReadModel, err := uc.chatRepo.FindByID(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, ErrChatNotFound
	}

	// Проверяем, что пользователь является участником чата
	if !uc.isParticipant(chatReadModel, cmd.AuthorID) {
		return Result{}, ErrNotChatParticipant
	}

	// 3. Проверка parent message (если это reply)
	if !cmd.ParentMessageID.IsZero() {
		parent, parentErr := uc.messageRepo.FindByID(ctx, cmd.ParentMessageID)
		if parentErr != nil {
			return Result{}, ErrParentNotFound
		}
		// Проверка, что parent в том же чате
		if parent.ChatID() != cmd.ChatID {
			return Result{}, ErrParentInDifferentChat
		}
	}

	// 4. Создание сообщения
	// 2. Создаем сообщение
	msg, err := messagedomain.NewMessage(
		cmd.ChatID,
		cmd.AuthorID,
		cmd.Content,
		cmd.ParentMessageID,
	)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create message: %w", err)
	}

	// 5. Сохранение
	if saveErr := uc.messageRepo.Save(ctx, msg); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save message: %w", saveErr)
	}

	// 6. Публикация события (для WebSocket broadcast)
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
	// Не критично, сообщение уже сохранено
	// TODO: log error
	_ = uc.eventBus.Publish(ctx, evt)

	// 7. Асинхронная обработка тегов (не блокируем ответ)
	if uc.tagProcessor != nil && uc.tagExecutor != nil {
		go uc.processTagsAsync(ctx, msg, cmd.AuthorID)
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

// processTagsAsync обрабатывает теги в содержимом сообщения асинхронно
// Выполняется в горутине для того чтобы не блокировать основной ответ
func (uc *SendMessageUseCase) processTagsAsync(
	ctx context.Context,
	msg *messagedomain.Message,
	authorID uuid.UUID,
) {
	// Convert domain UUID to google UUID for processor
	chatIDGoogle, err := msg.ChatID().ToGoogleUUID()
	if err != nil {
		// UUID conversion error - ignore
		return
	}

	// Parse and process tags from message content
	// currentEntityType is empty because this is a message, not an entity
	processingResult := uc.tagProcessor.ProcessMessage(chatIDGoogle, msg.Content(), "")
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

	// TODO: форматирование результатов через tag.Formatter и отправка reply
}

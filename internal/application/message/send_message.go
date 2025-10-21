package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/tag"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// ChatRepository определяет интерфейс для доступа к чатам (consumer-side interface)
type ChatRepository interface {
	FindByID(ctx context.Context, chatID string) (*chat.ReadModel, error)
}

// SendMessageUseCase обрабатывает отправку сообщения
type SendMessageUseCase struct {
	messageRepo    message.Repository
	chatRepo       ChatRepository
	eventBus       event.Bus
	tagProcessor   *tag.Processor       // Tag processor for parsing tags from message content
	tagExecutor    *tag.CommandExecutor // Tag executor for executing tag commands
}

// NewSendMessageUseCase создает новый SendMessageUseCase
func NewSendMessageUseCase(
	messageRepo message.Repository,
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
	chatReadModel, err := uc.chatRepo.FindByID(ctx, cmd.ChatID.String())
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
	msg, err := message.NewMessage(
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
	evt := message.NewCreated(
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
	if err := shared.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := shared.ValidateRequired("content", cmd.Content); err != nil {
		return ErrEmptyContent
	}
	if len(cmd.Content) > MaxContentLength {
		return ErrContentTooLong
	}
	if err := shared.ValidateUUID("authorID", cmd.AuthorID); err != nil {
		return err
	}
	return nil
}

func (uc *SendMessageUseCase) isParticipant(chatReadModel *chat.ReadModel, userID uuid.UUID) bool {
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
	msg *message.Message,
	authorID uuid.UUID,
) {
	// Конвертируем domain UUID в google UUID для processor
	chatIDGoogle, err := msg.ChatID().ToGoogleUUID()
	if err != nil {
		// Ошибка конвертации UUID - игнорируем
		return
	}

	// Парсинг и обработка тегов из содержимого сообщения
	// currentEntityType пустой, т.к. это сообщение, а не сущность
	processingResult := uc.tagProcessor.ProcessMessage(chatIDGoogle, msg.Content(), "")
	if len(processingResult.AppliedTags) == 0 {
		// Нет успешно применённых тегов - выходим
		return
	}

	// Конвертируем domain UUID в google UUID для executor
	authorIDGoogle, convErr := authorID.ToGoogleUUID()
	if convErr != nil {
		// Ошибка конвертации UUID - выходим
		return
	}

	// Выполняем команды
	for _, tagApp := range processingResult.AppliedTags {
		cmd, ok := tagApp.Command.(tag.Command)
		if !ok {
			// Не команда или неизвестный тип - пропускаем
			continue
		}

		if execErr := uc.tagExecutor.Execute(ctx, cmd, authorIDGoogle); execErr != nil {
			// TODO: отправить notification об ошибке или создать reply с ботом
			// Для теперь просто логируем ошибку (или игнорируем)
		}
	}

	// TODO: форматирование результатов через tag.Formatter и отправка reply
}

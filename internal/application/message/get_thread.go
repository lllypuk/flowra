package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
)

// GetThreadUseCase обрабатывает получение треда (ответов на сообщение)
type GetThreadUseCase struct {
	messageRepo Repository
}

// NewGetThreadUseCase создает новый GetThreadUseCase
func NewGetThreadUseCase(messageRepo Repository) *GetThreadUseCase {
	return &GetThreadUseCase{
		messageRepo: messageRepo,
	}
}

// Execute выполняет получение треда
func (uc *GetThreadUseCase) Execute(
	ctx context.Context,
	query GetThreadQuery,
) (ListResult, error) {
	// Валидация
	if err := uc.validate(query); err != nil {
		return ListResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// Проверяем, что parent message существует
	parentMsg, err := uc.messageRepo.FindByID(ctx, query.ParentMessageID)
	if err != nil {
		return ListResult{}, ErrParentNotFound
	}

	// Загрузка ответов в треде
	messages, err := uc.messageRepo.FindThread(ctx, parentMsg.ID())
	if err != nil {
		return ListResult{}, fmt.Errorf("failed to find thread messages: %w", err)
	}

	return ListResult{
		Value: messages,
	}, nil
}

func (uc *GetThreadUseCase) validate(query GetThreadQuery) error {
	if err := shared.ValidateUUID("parentMessageID", query.ParentMessageID); err != nil {
		return err
	}
	return nil
}

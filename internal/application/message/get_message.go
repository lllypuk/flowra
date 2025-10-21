package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/message"
)

// GetMessageUseCase обрабатывает получение сообщения по ID
type GetMessageUseCase struct {
	messageRepo message.Repository
}

// NewGetMessageUseCase создает новый GetMessageUseCase
func NewGetMessageUseCase(messageRepo message.Repository) *GetMessageUseCase {
	return &GetMessageUseCase{
		messageRepo: messageRepo,
	}
}

// Execute выполняет получение сообщения
func (uc *GetMessageUseCase) Execute(
	ctx context.Context,
	query GetMessageQuery,
) (Result, error) {
	// Валидация
	if err := uc.validate(query); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Загрузка сообщения
	msg, err := uc.messageRepo.FindByID(ctx, query.MessageID)
	if err != nil {
		return Result{}, ErrMessageNotFound
	}

	return Result{
		Value: msg,
	}, nil
}

func (uc *GetMessageUseCase) validate(query GetMessageQuery) error {
	if err := shared.ValidateUUID("messageID", query.MessageID); err != nil {
		return err
	}
	return nil
}

package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// GetMessageUseCase handles retrieval messages по ID
type GetMessageUseCase struct {
	messageRepo Repository
}

// NewGetMessageUseCase creates New GetMessageUseCase
func NewGetMessageUseCase(messageRepo Repository) *GetMessageUseCase {
	return &GetMessageUseCase{
		messageRepo: messageRepo,
	}
}

// Execute performs retrieval messages
func (uc *GetMessageUseCase) Execute(
	ctx context.Context,
	query GetMessageQuery,
) (Result, error) {
	// validation
	if err := uc.validate(query); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Loading message
	msg, err := uc.messageRepo.FindByID(ctx, query.MessageID)
	if err != nil {
		return Result{}, ErrMessageNotFound
	}

	return Result{
		Value: msg,
	}, nil
}

func (uc *GetMessageUseCase) validate(query GetMessageQuery) error {
	if err := appcore.ValidateUUID("messageID", query.MessageID); err != nil {
		return err
	}
	return nil
}

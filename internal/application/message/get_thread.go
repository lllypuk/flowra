package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// GetThreadUseCase handles retrieval treda (response on message)
type GetThreadUseCase struct {
	messageRepo Repository
}

// NewGetThreadUseCase creates New GetThreadUseCase
func NewGetThreadUseCase(messageRepo Repository) *GetThreadUseCase {
	return &GetThreadUseCase{
		messageRepo: messageRepo,
	}
}

// Execute performs retrieval treda
func (uc *GetThreadUseCase) Execute(
	ctx context.Context,
	query GetThreadQuery,
) (ListResult, error) {
	// validation
	if err := uc.validate(query); err != nil {
		return ListResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// Checking, that parent message suschestvuet
	parentMsg, err := uc.messageRepo.FindByID(ctx, query.ParentMessageID)
	if err != nil {
		return ListResult{}, ErrParentNotFound
	}

	// Loading response in thread
	messages, err := uc.messageRepo.FindThread(ctx, parentMsg.ID())
	if err != nil {
		return ListResult{}, fmt.Errorf("failed to find thread messages: %w", err)
	}

	return ListResult{
		Value: messages,
	}, nil
}

func (uc *GetThreadUseCase) validate(query GetThreadQuery) error {
	if err := appcore.ValidateUUID("parentMessageID", query.ParentMessageID); err != nil {
		return err
	}
	return nil
}

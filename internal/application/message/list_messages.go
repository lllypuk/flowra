package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// ListMessagesUseCase handles retrieval list soobscheniy in chate
type ListMessagesUseCase struct {
	messageRepo Repository
}

// NewListMessagesUseCase creates New ListMessagesUseCase
func NewListMessagesUseCase(messageRepo Repository) *ListMessagesUseCase {
	return &ListMessagesUseCase{
		messageRepo: messageRepo,
	}
}

// Execute performs retrieval list soobscheniy
func (uc *ListMessagesUseCase) Execute(
	ctx context.Context,
	query ListMessagesQuery,
) (ListResult, error) {
	// validation
	if err := uc.validate(&query); err != nil {
		return ListResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// podgotovka paginatsii
	pagination := Pagination{
		Limit:  query.Limit,
		Offset: query.Offset,
	}

	// Loading soobscheniy
	messages, err := uc.messageRepo.FindByChatID(ctx, query.ChatID, pagination)
	if err != nil {
		return ListResult{}, fmt.Errorf("failed to find messages: %w", err)
	}

	return ListResult{
		Value: messages,
	}, nil
}

func (uc *ListMessagesUseCase) validate(query *ListMessagesQuery) error {
	if err := appcore.ValidateUUID("chatID", query.ChatID); err != nil {
		return err
	}

	// setting defoltnyh values
	if query.Limit == 0 {
		query.Limit = DefaultLimit
	}
	if query.Limit > MaxLimit {
		query.Limit = MaxLimit
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	return nil
}

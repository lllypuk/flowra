package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// CreateChatUseCase handles the creation of a new chat
type CreateChatUseCase struct {
	chatRepo CommandRepository
}

// NewCreateChatUseCase creates a new CreateChatUseCase
func NewCreateChatUseCase(chatRepo CommandRepository) *CreateChatUseCase {
	return &CreateChatUseCase{
		chatRepo: chatRepo,
	}
}

// Execute executes the chat creation
func (uc *CreateChatUseCase) Execute(ctx context.Context, cmd CreateChatCommand) (Result, error) {
	// Validation
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Create aggregate as Discussion to preserve conversion event trail
	// NewChat() automatically generates ChatCreated and ParticipantAdded events
	chatAggregate, err := chat.NewChat(cmd.WorkspaceID, chat.TypeDiscussion, cmd.IsPublic, cmd.CreatedBy)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create chat: %w", err)
	}

	// Apply type and title
	if err = uc.applyChatTypeAndTitle(chatAggregate, cmd); err != nil {
		return Result{}, err
	}

	// Capture events before saving (for response)
	uncommittedEvents := chatAggregate.GetUncommittedEvents()

	// Save via repository (updates both event store and read model)
	if err = uc.chatRepo.Save(ctx, chatAggregate); err != nil {
		return Result{}, fmt.Errorf("failed to save chat: %w", err)
	}

	return Result{
		Result: appcore.Result[*chat.Chat]{
			Value:   chatAggregate,
			Version: chatAggregate.Version(),
		},
		Events: convertToInterfaceSlice(uncommittedEvents),
	}, nil
}

func (uc *CreateChatUseCase) applyChatTypeAndTitle(chatAggregate *chat.Chat, cmd CreateChatCommand) error {
	// For typed chats (Task/Bug/Epic) convert and set title
	if cmd.Type != chat.TypeDiscussion {
		var err error
		switch cmd.Type {
		case chat.TypeTask:
			err = chatAggregate.ConvertToTask(cmd.Title, cmd.CreatedBy)
		case chat.TypeBug:
			err = chatAggregate.ConvertToBug(cmd.Title, cmd.CreatedBy)
		case chat.TypeEpic:
			err = chatAggregate.ConvertToEpic(cmd.Title, cmd.CreatedBy)
		case chat.TypeDiscussion:
			// Unreachable because of outer if, but needed for exhaustive linter
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to convert to %s: %w", cmd.Type, err)
		}
		return nil
	}

	if cmd.Title != "" {
		// For Discussion chats, set title via Rename if provided
		if err := chatAggregate.Rename(cmd.Title, cmd.CreatedBy); err != nil {
			return fmt.Errorf("failed to set title: %w", err)
		}
	}
	return nil
}

func (uc *CreateChatUseCase) validate(cmd CreateChatCommand) error {
	if err := appcore.ValidateUUID("workspaceID", cmd.WorkspaceID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("createdBy", cmd.CreatedBy); err != nil {
		return err
	}
	if err := appcore.ValidateEnum("type", string(cmd.Type), []string{
		string(chat.TypeDiscussion),
		string(chat.TypeTask),
		string(chat.TypeBug),
		string(chat.TypeEpic),
	}); err != nil {
		return err
	}

	// For typed chats title is required
	if cmd.Type != chat.TypeDiscussion {
		if err := appcore.ValidateRequired("title", cmd.Title); err != nil {
			return ErrTitleRequired
		}
		if err := appcore.ValidateMaxLength("title", cmd.Title, appcore.MaxTitleLength); err != nil {
			return err
		}
	} else if cmd.Title != "" {
		// For Discussion chats title is optional, but if provided - check length
		if err := appcore.ValidateMaxLength("title", cmd.Title, appcore.MaxTitleLength); err != nil {
			return err
		}
	}

	return nil
}

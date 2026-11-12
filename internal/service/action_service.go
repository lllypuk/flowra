package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	messageapp "github.com/lllypuk/flowra/internal/application/message"
	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// commandTimeout bounds one action, lock wait included; it stays below the server write timeout.
const commandTimeout = 20 * time.Second

// ActionMessageSender defines command execution required by ActionService.
type ActionMessageSender interface {
	Execute(ctx context.Context, cmd messageapp.SendMessageCommand) (messageapp.Result, error)
}

// TaskProjectionSync defines projection synchronization required by ActionService.
type TaskProjectionSync interface {
	RebuildOne(ctx context.Context, chatID uuid.UUID) error
}

// ChatCommand is a chat use case run by ActionService.
type ChatCommand[C any] interface {
	Execute(ctx context.Context, cmd C) (chatapp.Result, error)
}

// ActionChatReader reads the chat an action targets.
type ActionChatReader interface {
	FindByID(ctx context.Context, chatID uuid.UUID) (*chatapp.ReadModel, error)
}

// ActionCommandRunner applies a chat command with a definite outcome.
type ActionCommandRunner interface {
	Run(
		ctx context.Context,
		chatID uuid.UUID,
		key appcore.CommandKey,
		exec func(context.Context) (chatapp.Result, error),
	) (chatapp.Outcome, error)
}

// ActionChatCommands are the chat use cases behind the actions.
type ActionChatCommands struct {
	ChangeStatus ChatCommand[chatapp.ChangeStatusCommand]
	SetPriority  ChatCommand[chatapp.SetPriorityCommand]
	AssignUser   ChatCommand[chatapp.AssignUserCommand]
	SetDueDate   ChatCommand[chatapp.SetDueDateCommand]
	Rename       ChatCommand[chatapp.RenameChatCommand]
	Close        ChatCommand[chatapp.CloseChatCommand]
	Reopen       ChatCommand[chatapp.ReopenChatCommand]
}

// ActionServiceOption customizes ActionService behavior.
type ActionServiceOption func(*ActionService)

// WithTaskProjectionSync configures synchronous task read-model rebuild after task-related actions.
func WithTaskProjectionSync(sync TaskProjectionSync) ActionServiceOption {
	return func(s *ActionService) {
		s.taskProjectionSync = sync
	}
}

// WithChatProjectionSync configures the chat read-model rebuild after a replayed command.
func WithChatProjectionSync(sync TaskProjectionSync) ActionServiceOption {
	return func(s *ActionService) {
		s.chatProjectionSync = sync
	}
}

// ActionService applies UI and agent actions to a chat and answers with a definite outcome.
type ActionService struct {
	commands           ActionChatCommands
	runner             ActionCommandRunner
	chats              ActionChatReader
	sendMessageUC      ActionMessageSender
	userRepo           appcore.UserRepository
	taskProjectionSync TaskProjectionSync
	chatProjectionSync TaskProjectionSync
	batcher            *ChangeBatcher
	logger             *slog.Logger
}

// NewActionService creates a new ActionService.
func NewActionService(
	commands ActionChatCommands,
	runner ActionCommandRunner,
	chats ActionChatReader,
	sendMessageUC ActionMessageSender,
	userRepo appcore.UserRepository,
	opts ...ActionServiceOption,
) *ActionService {
	svc := &ActionService{
		commands:      commands,
		runner:        runner,
		chats:         chats,
		sendMessageUC: sendMessageUC,
		userRepo:      userRepo,
		logger:        slog.Default(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(svc)
		}
	}

	svc.batcher = NewChangeBatcher(defaultBatchWindow, svc.flushBatchMessage)
	svc.batcher.logger = svc.logger

	return svc
}

// getActorDisplayName returns the display name for the actor
func (s *ActionService) getActorDisplayName(ctx context.Context, actorID uuid.UUID) string {
	if s.userRepo == nil {
		return ""
	}
	usr, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil || usr == nil {
		return ""
	}
	return usr.FullName
}

// ChangeStatus sets the chat status.
func (s *ActionService) ChangeStatus(
	ctx context.Context,
	chatID uuid.UUID,
	newStatus string,
	actorID uuid.UUID,
	key string,
) (*appcore.ActionResult, error) {
	return s.apply(ctx, chatID, actorID, key, "status:"+newStatus, ChangeTypeStatus, newStatus,
		func(ctx context.Context) (chatapp.Result, error) {
			return s.commands.ChangeStatus.Execute(ctx, chatapp.ChangeStatusCommand{
				ChatID: chatID, Status: newStatus, ChangedBy: actorID,
			})
		})
}

// AssignUser sets or clears the assignee.
func (s *ActionService) AssignUser(
	ctx context.Context,
	chatID uuid.UUID,
	assigneeID *uuid.UUID,
	actorID uuid.UUID,
	key string,
) (*appcore.ActionResult, error) {
	fingerprint, assigneeName := "assignee:none", ""
	if assigneeID != nil {
		usr, err := s.userRepo.GetByID(ctx, *assigneeID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve user: %w", err)
		}
		fingerprint = "assignee:" + assigneeID.String()
		assigneeName = usr.FullName
		if assigneeName == "" {
			assigneeName = usr.Username
		}
	}
	return s.apply(ctx, chatID, actorID, key, fingerprint, ChangeTypeAssignee, assigneeName,
		func(ctx context.Context) (chatapp.Result, error) {
			return s.commands.AssignUser.Execute(ctx, chatapp.AssignUserCommand{
				ChatID: chatID, AssigneeID: assigneeID, AssignedBy: actorID,
			})
		})
}

// SetPriority sets the chat priority.
func (s *ActionService) SetPriority(
	ctx context.Context,
	chatID uuid.UUID,
	priority string,
	actorID uuid.UUID,
	key string,
) (*appcore.ActionResult, error) {
	return s.apply(ctx, chatID, actorID, key, "priority:"+priority, ChangeTypePriority, priority,
		func(ctx context.Context) (chatapp.Result, error) {
			return s.commands.SetPriority.Execute(ctx, chatapp.SetPriorityCommand{
				ChatID: chatID, Priority: priority, SetBy: actorID,
			})
		})
}

// SetDueDate sets or clears the due date.
func (s *ActionService) SetDueDate(
	ctx context.Context,
	chatID uuid.UUID,
	dueDate *time.Time,
	actorID uuid.UUID,
	key string,
) (*appcore.ActionResult, error) {
	fingerprint, formatted := "due:none", ""
	if dueDate != nil {
		fingerprint = "due:" + dueDate.Format(time.DateOnly)
		formatted = dueDate.Format("January 2, 2006")
	}
	return s.apply(ctx, chatID, actorID, key, fingerprint, ChangeTypeDueDate, formatted,
		func(ctx context.Context) (chatapp.Result, error) {
			return s.commands.SetDueDate.Execute(ctx, chatapp.SetDueDateCommand{
				ChatID: chatID, DueDate: dueDate, SetBy: actorID,
			})
		})
}

// Rename changes the chat title.
func (s *ActionService) Rename(
	ctx context.Context,
	chatID uuid.UUID,
	newTitle string,
	actorID uuid.UUID,
	key string,
) (*appcore.ActionResult, error) {
	return s.apply(ctx, chatID, actorID, key, "rename:"+newTitle, ChangeTypeTitle, newTitle,
		func(ctx context.Context) (chatapp.Result, error) {
			return s.commands.Rename.Execute(ctx, chatapp.RenameChatCommand{
				ChatID: chatID, NewTitle: newTitle, RenamedBy: actorID,
			})
		})
}

// Close closes the chat.
func (s *ActionService) Close(
	ctx context.Context,
	chatID uuid.UUID,
	actorID uuid.UUID,
	key string,
) (*appcore.ActionResult, error) {
	res, err := s.apply(ctx, chatID, actorID, key, "close", "", "",
		func(ctx context.Context) (chatapp.Result, error) {
			return s.commands.Close.Execute(ctx, chatapp.CloseChatCommand{ChatID: chatID, ClosedBy: actorID})
		})
	if err == nil && !res.Replayed {
		s.postNote(ctx, chatID, actorID, "closed the chat", "Chat closed")
	}
	return res, err
}

// Reopen reopens a closed chat.
func (s *ActionService) Reopen(
	ctx context.Context,
	chatID uuid.UUID,
	actorID uuid.UUID,
	key string,
) (*appcore.ActionResult, error) {
	res, err := s.apply(ctx, chatID, actorID, key, "reopen", "", "",
		func(ctx context.Context) (chatapp.Result, error) {
			return s.commands.Reopen.Execute(ctx, chatapp.ReopenChatCommand{ChatID: chatID, ReopenedBy: actorID})
		})
	if err == nil && !res.Replayed {
		s.postNote(ctx, chatID, actorID, "reopened the chat", "Chat reopened")
	}
	return res, err
}

// apply runs one action. Once the command is applied, projection sync and the thread note are best effort:
// their failure is logged and the answer stays "applied".
func (s *ActionService) apply(
	ctx context.Context,
	chatID, actorID uuid.UUID,
	key, fingerprint string,
	changeType ChangeType,
	changeValue string,
	exec func(context.Context) (chatapp.Result, error),
) (*appcore.ActionResult, error) {
	// Detached from the request: a client disconnect must not abort a commit and make its outcome unknown.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), commandTimeout)
	defer cancel()

	if err := s.checkParticipant(ctx, chatID, actorID); err != nil {
		return &appcore.ActionResult{Success: false, Error: err.Error()}, err
	}

	cmdKey := appcore.CommandKey{Fingerprint: fingerprint}
	if key != "" {
		cmdKey.ID = actorID.String() + ":" + key
	}
	out, err := s.runner.Run(ctx, chatID, cmdKey, exec)
	if errors.Is(err, errs.ErrOutcomeUnknown) {
		// The commit may have landed without reaching the read models, and a retry without a key writes nothing.
		s.syncProjection(ctx, s.chatProjectionSync, chatID, "chat")
		s.syncProjection(ctx, s.taskProjectionSync, chatID, "task")
	}
	if err != nil {
		return &appcore.ActionResult{Success: false, Error: err.Error()}, err
	}

	if out.Replayed {
		// The first attempt may have committed without reaching the read models.
		s.syncProjection(ctx, s.chatProjectionSync, chatID, "chat")
	}
	s.syncProjection(ctx, s.taskProjectionSync, chatID, "task")

	if changeType != "" && !out.Replayed {
		actorName := s.getActorDisplayName(ctx, actorID)
		if batchErr := s.batcher.AddChange(ctx, actorID, chatID, actorName, changeType, changeValue); batchErr != nil {
			s.logger.WarnContext(ctx, "failed to batch change message", slog.String("error", batchErr.Error()))
		}
	}

	return &appcore.ActionResult{ChatID: chatID, Version: out.Version, Replayed: out.Replayed, Success: true}, nil
}

func (s *ActionService) checkParticipant(ctx context.Context, chatID, actorID uuid.UUID) error {
	if s.chats == nil {
		return nil
	}
	chat, err := s.chats.FindByID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to load chat: %w", err)
	}
	if !slices.ContainsFunc(chat.Participants, func(p chatdomain.Participant) bool { return p.UserID() == actorID }) {
		return fmt.Errorf("%w: not a chat participant", errs.ErrForbidden)
	}
	return nil
}

func (s *ActionService) syncProjection(ctx context.Context, sync TaskProjectionSync, chatID uuid.UUID, name string) {
	if sync == nil {
		return
	}
	if err := sync.RebuildOne(ctx, chatID); err != nil {
		s.logger.ErrorContext(ctx, "failed to sync projection after chat action",
			slog.String("projection", name),
			slog.String("chat_id", chatID.String()),
			slog.String("error", err.Error()),
		)
	}
}

// postNote writes a human-readable line to the thread; it is a record for people, not part of the outcome.
func (s *ActionService) postNote(ctx context.Context, chatID, actorID uuid.UUID, withActor, anonymous string) {
	content := "✅ " + anonymous
	if actorName := s.getActorDisplayName(ctx, actorID); actorName != "" {
		content = fmt.Sprintf("✅ %s %s", actorName, withActor)
	}
	if err := s.flushBatchMessage(ctx, chatID, content, actorID); err != nil {
		s.logger.WarnContext(ctx, "failed to post action note", slog.String("error", err.Error()))
	}
}

// flushBatchMessage is called by the batcher to send a combined message
func (s *ActionService) flushBatchMessage(
	ctx context.Context,
	chatID uuid.UUID,
	content string,
	actorID uuid.UUID,
) error {
	_, err := s.sendMessageUC.Execute(ctx, messageapp.SendMessageCommand{
		ChatID:   chatID,
		AuthorID: actorID,
		Content:  content,
		Type:     message.TypeSystem,
		ActorID:  &actorID,
	})
	return err
}

// Shutdown stops the batcher and cleans up resources
func (s *ActionService) Shutdown() {
	if s.batcher != nil {
		s.batcher.Close()
	}
}

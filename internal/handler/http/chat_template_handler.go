package httphandler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	messageapp "github.com/lllypuk/flowra/internal/application/message"
	taskapp "github.com/lllypuk/flowra/internal/application/task"
	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/tag"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Chat template handler constants.
const (
	defaultChatTemplateListLimit = 50
	roleOwner                    = "owner"
	roleAdmin                    = "admin"
	roleMember                   = "member"
	roleCreator                  = "creator"
)

// ChatTemplateService defines the interface for chat operations needed by templates.
// Declared on the consumer side per project guidelines.
type ChatTemplateService interface {
	// CreateChat creates a new chat.
	CreateChat(ctx context.Context, cmd chatapp.CreateChatCommand) (chatapp.Result, error)

	// GetChat gets a chat by ID.
	GetChat(ctx context.Context, query chatapp.GetChatQuery) (*chatapp.GetChatResult, error)

	// ListChats lists chats in a workspace.
	ListChats(ctx context.Context, query chatapp.ListChatsQuery) (*chatapp.ListChatsResult, error)
}

// MessageTemplateService defines the interface for message operations needed by templates.
// Declared on the consumer side per project guidelines.
type MessageTemplateService interface {
	// ListMessages lists messages in a chat.
	ListMessages(ctx context.Context, query messageapp.ListMessagesQuery) (messageapp.ListResult, error)

	// GetMessage gets a message by ID.
	GetMessage(ctx context.Context, messageID uuid.UUID) (*message.Message, error)
}

// TaskQueryForChatService defines the interface for querying tasks by chat ID.
// Declared on the consumer side per project guidelines.
type TaskQueryForChatService interface {
	// GetTaskByChatID gets a task by its associated chat ID.
	GetTaskByChatID(ctx context.Context, chatID uuid.UUID) (*taskapp.ReadModel, error)
}

// ChatTaskProjectionSync defines projection synchronization required by chat template flows.
type ChatTaskProjectionSync interface {
	RebuildOne(ctx context.Context, chatID uuid.UUID) error
}

// ChatViewData represents chat data for templates.
type ChatViewData struct {
	ID               string
	WorkspaceID      string
	Title            string
	Type             string
	IsPublic         bool
	IsTaskChat       bool
	Status           string
	AssigneeID       string
	Priority         string
	DueDate          *time.Time
	CreatedBy        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ParticipantCount int
	UnreadCount      int
	LastMessage      *LastMessageData
}

// LastMessageData represents the last message in a chat.
type LastMessageData struct {
	Content        string
	AuthorUsername string
	CreatedAt      time.Time
}

// MessageViewData represents message data for templates.
type MessageViewData struct {
	ID              string
	ChatID          string
	Content         string
	CreatedAt       time.Time
	EditedAt        *time.Time
	IsDeleted       bool
	IsSystemMessage bool
	IsBotMessage    bool
	IsGroupStart    bool // first message in a group of consecutive system/bot messages
	IsGroupEnd      bool // last message in a group of consecutive system/bot messages
	CanEdit         bool
	Author          MessageAuthorData
	Tags            []MessageTagData
	Reactions       []MessageReactionData
	Attachments     []AttachmentViewData
}

// MessageAuthorData represents message author data for templates.
type MessageAuthorData struct {
	ID          string
	Username    string
	DisplayName string
	AvatarURL   string
}

// MessageTagData represents a tag in a message.
type MessageTagData struct {
	Key   string
	Value string
}

// MessageReactionData represents reaction data for templates.
type MessageReactionData struct {
	Emoji      string
	Count      int
	HasReacted bool
	Users      []string
}

// AttachmentViewData represents attachment data for templates.
type AttachmentViewData struct {
	FileID   string
	FileName string
	FileSize int64
	MimeType string
	URL      string
	IsImage  bool
}

// ParticipantViewData represents participant data for templates.
type ParticipantViewData struct {
	UserID      string
	Username    string
	DisplayName string
	AvatarURL   string
	Role        string
	JoinedAt    time.Time
}

// TaskViewData represents task-specific data for task/bug/epic chats.
type TaskViewData struct {
	ID         string
	Status     string
	AssigneeID string
	Priority   string
	DueDate    *time.Time
	Severity   string // for bugs only
}

// ChatTemplateHandler provides handlers for rendering chat HTML pages.
type ChatTemplateHandler struct {
	renderer       *TemplateRenderer
	logger         *slog.Logger
	chatService    ChatTemplateService
	messageService MessageTemplateService
	taskService    TaskQueryForChatService
	taskProjector  ChatTaskProjectionSync
	userLookup     UserProfileLookup
	memberService  BoardMemberService
}

// NewChatTemplateHandler creates a new chat template handler.
func NewChatTemplateHandler(
	renderer *TemplateRenderer,
	logger *slog.Logger,
	chatService ChatTemplateService,
	messageService MessageTemplateService,
	taskService TaskQueryForChatService,
) *ChatTemplateHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ChatTemplateHandler{
		renderer:       renderer,
		logger:         logger,
		chatService:    chatService,
		messageService: messageService,
		taskService:    taskService,
	}
}

// SetUserLookup sets the user lookup service for resolving participant profiles.
func (h *ChatTemplateHandler) SetUserLookup(lookup UserProfileLookup) {
	h.userLookup = lookup
}

// SetMemberService sets the member service for loading workspace members.
func (h *ChatTemplateHandler) SetMemberService(svc BoardMemberService) {
	h.memberService = svc
}

// SetTaskProjector sets synchronous task read-model projector for typed chat flows.
func (h *ChatTemplateHandler) SetTaskProjector(projector ChatTaskProjectionSync) {
	h.taskProjector = projector
}

// SetupChatRoutes registers chat-related page and partial routes.
func (h *ChatTemplateHandler) SetupChatRoutes(e *echo.Echo) {
	// Chat pages (protected)
	workspaces := e.Group("/workspaces", RequireAuth)
	workspaces.GET("/:workspace_id/chats", h.ChatLayout)
	workspaces.GET("/:workspace_id/chats/:chat_id", h.ChatView)

	// Chat partials (protected)
	partials := e.Group("/partials", RequireAuth)
	partials.GET("/workspace/:workspace_id/chats", h.ChatListPartial)
	partials.GET("/chats/:chat_id", h.ChatViewPartial)
	partials.GET("/chats/:chat_id/messages", h.MessagesPartial)
	partials.GET("/messages/:message_id", h.SingleMessagePartial)
	partials.GET("/messages/:message_id/edit", h.MessageEditForm)
	partials.GET("/chats/:chat_id/participants", h.ParticipantsPartial)
	partials.GET("/chat/create-form", h.ChatCreateForm)
	partials.POST("/chat/create", h.ChatCreate)
	partials.GET("/chats/search", h.ChatSearchPartial)
}

// ChatLayout renders the main chat page with 3-column layout.
func (h *ChatTemplateHandler) ChatLayout(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	workspaceID, err := uuid.ParseUUID(c.Param("workspace_id"))
	if err != nil {
		return h.renderNotFound(c)
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return h.renderNotFound(c)
	}

	// Get workspace info (using workspace ID directly since we have access)
	workspaceData := WorkspaceViewData{
		ID: workspaceID.String(),
	}

	data := map[string]any{
		"Workspace": workspaceData,
		"Chat":      nil, // No chat selected initially
		"Token":     "",  // TODO: get JWT token for WebSocket auth
	}

	// If chat_id is provided in query, load that chat
	chatIDParam := c.QueryParam("chat_id")
	if chatIDParam != "" {
		chatID, parseErr := uuid.ParseUUID(chatIDParam)
		if parseErr == nil {
			chatData, loadErr := h.loadChatViewData(c.Request().Context(), chatID, userID)
			if loadErr == nil {
				data["Chat"] = chatData
			}
		}
	}

	return h.render(c, "chat/layout.html", "Chats", data)
}

// ChatView renders the chat page with a specific chat selected.
func (h *ChatTemplateHandler) ChatView(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	workspaceID, err := uuid.ParseUUID(c.Param("workspace_id"))
	if err != nil {
		return h.renderNotFound(c)
	}

	chatID, err := uuid.ParseUUID(c.Param("chat_id"))
	if err != nil {
		return h.renderNotFound(c)
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return h.renderNotFound(c)
	}

	// Load chat data
	chatData, err := h.loadChatViewData(c.Request().Context(), chatID, userID)
	if err != nil {
		h.logger.Error("failed to load chat", slog.String("error", err.Error()))
		return h.renderNotFound(c)
	}

	workspaceData := WorkspaceViewData{
		ID: workspaceID.String(),
	}

	data := map[string]any{
		"Workspace": workspaceData,
		"Chat":      chatData,
		"Token":     "", // TODO: get JWT token for WebSocket auth
	}

	// Load task data for task chats
	if chatData.IsTaskChat {
		data["Task"] = h.loadTaskViewData(c.Request().Context(), chatData)
		data["Participants"] = h.loadWorkspaceMembers(c.Request().Context(), workspaceID)
		data["Statuses"] = getChatStatusOptions(chatData.Type)
	}

	return h.render(c, "chat/layout.html", chatData.Title, data)
}

// ChatViewPartial returns just the chat view content for HTMX requests.
func (h *ChatTemplateHandler) ChatViewPartial(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	chatID, err := uuid.ParseUUID(c.Param("chat_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid chat ID")
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return c.String(http.StatusUnauthorized, "Invalid user")
	}

	chatData, err := h.loadChatViewData(c.Request().Context(), chatID, userID)
	if err != nil {
		return c.String(http.StatusNotFound, "Chat not found")
	}

	// If this is not an HTMX request (direct page load), redirect to full page
	if c.Request().Header.Get("Hx-Request") == "" {
		fullURL := "/workspaces/" + chatData.WorkspaceID + "/chats/" + chatData.ID
		return c.Redirect(http.StatusFound, fullURL)
	}

	// Build inner data map
	innerData := map[string]any{
		"Chat": chatData,
	}

	if chatData.IsTaskChat {
		innerData["Task"] = h.loadTaskViewData(c.Request().Context(), chatData)
		if wsID, parseErr := uuid.ParseUUID(chatData.WorkspaceID); parseErr == nil {
			innerData["Participants"] = h.loadWorkspaceMembers(c.Request().Context(), wsID)
		}
		innerData["Statuses"] = getChatStatusOptions(chatData.Type)
	}

	// Wrap in "Data" to match template expectations (template uses .Data.Chat.ID)
	data := map[string]any{
		"Data": innerData,
	}

	return h.renderPartial(c, "chat/view", data)
}

// ChatListPartial returns the chat list as HTML partial for HTMX.
func (h *ChatTemplateHandler) ChatListPartial(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	workspaceID, err := uuid.ParseUUID(c.Param("workspace_id"))
	if err != nil {
		h.logger.Error("invalid workspace_id param", slog.String("param", c.Param("workspace_id")))
		return c.String(http.StatusBadRequest, "Invalid workspace ID")
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return c.String(http.StatusUnauthorized, "Invalid user")
	}

	if h.chatService == nil {
		h.logger.Warn("chatService is nil, returning empty list")
		return h.renderPartial(c, "chat/list", map[string]any{
			"Chats":        []ChatViewData{},
			"ActiveChatID": "",
			"WorkspaceID":  workspaceID.String(),
		})
	}

	query := chatapp.ListChatsQuery{
		WorkspaceID: workspaceID,
		RequestedBy: userID,
		Limit:       defaultChatTemplateListLimit,
		Offset:      0,
	}

	h.logger.Info("listing chats",
		slog.String("workspace_id", workspaceID.String()),
		slog.String("user_id", userID.String()))

	result, err := h.chatService.ListChats(c.Request().Context(), query)
	if err != nil {
		h.logger.Error("failed to list chats",
			slog.String("error", err.Error()),
			slog.String("workspace_id", workspaceID.String()))
		return h.renderPartial(c, "chat/list", map[string]any{
			"Chats":        []ChatViewData{},
			"ActiveChatID": "",
			"WorkspaceID":  workspaceID.String(),
		})
	}

	if result == nil {
		h.logger.Error("ListChats returned nil result")
		return h.renderPartial(c, "chat/list", map[string]any{
			"Chats":        []ChatViewData{},
			"ActiveChatID": "",
			"WorkspaceID":  workspaceID.String(),
		})
	}

	h.logger.Info("found chats", slog.Int("count", len(result.Chats)))

	// Convert to view data
	chatViews := make([]ChatViewData, 0, len(result.Chats))
	for _, chat := range result.Chats {
		chatViews = append(chatViews, ChatViewData{
			ID:          chat.ID.String(),
			WorkspaceID: chat.WorkspaceID.String(),
			Title:       chat.Title,
			Type:        string(chat.Type),
			IsPublic:    chat.IsPublic,
			IsTaskChat:  isTaskType(string(chat.Type)),
			CreatedAt:   chat.CreatedAt,
			UpdatedAt:   chat.CreatedAt, // TODO: add updated_at to domain
			UnreadCount: 0,              // TODO: implement unread count
		})
	}

	// Get active chat ID from query param
	activeChatID := c.QueryParam("active")

	data := map[string]any{
		"Chats":        chatViews,
		"ActiveChatID": activeChatID,
		"WorkspaceID":  workspaceID.String(),
	}

	h.logger.Info("rendering chat/list template",
		slog.Int("chat_count", len(chatViews)),
		slog.String("workspace_id", workspaceID.String()))

	return h.renderPartial(c, "chat/list", data)
}

// MessagesPartial returns messages for a chat as HTML partial.
func (h *ChatTemplateHandler) MessagesPartial(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	chatID, err := uuid.ParseUUID(c.Param("chat_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid chat ID")
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return c.String(http.StatusUnauthorized, "Invalid user")
	}

	if h.messageService == nil {
		return h.renderPartial(c, "messages-list", map[string]any{
			"Messages": []MessageViewData{},
		})
	}

	query := messageapp.ListMessagesQuery{
		ChatID: chatID,
		Limit:  defaultChatTemplateListLimit,
		Offset: 0,
	}

	h.logger.Debug("listing messages for chat",
		slog.String("chat_id", chatID.String()),
		slog.Int("limit", query.Limit),
	)

	result, err := h.messageService.ListMessages(c.Request().Context(), query)
	if err != nil {
		h.logger.Error("failed to list messages",
			slog.String("chat_id", chatID.String()),
			slog.String("error", err.Error()),
		)
		return h.renderPartial(c, "messages-list", map[string]any{
			"Messages": []MessageViewData{},
		})
	}

	h.logger.Debug("messages loaded",
		slog.String("chat_id", chatID.String()),
		slog.Int("count", len(result.Value)),
	)

	// Convert to view data
	messageViews := make([]MessageViewData, 0, len(result.Value))
	for _, msg := range result.Value {
		if msg == nil {
			continue
		}
		if shouldHideSystemTagCommand(msg) {
			continue
		}
		messageViews = append(messageViews, h.convertMessageToView(msg, userID))
	}

	// Apply grouping for consecutive system/bot messages within 5 seconds
	applyMessageGrouping(messageViews)

	h.logger.Debug("messages converted to views",
		slog.String("chat_id", chatID.String()),
		slog.Int("view_count", len(messageViews)),
	)

	data := map[string]any{
		"Messages": messageViews,
	}

	return h.renderPartial(c, "messages-list", data)
}

// SingleMessagePartial returns a single message as HTML partial.
func (h *ChatTemplateHandler) SingleMessagePartial(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	messageID, err := uuid.ParseUUID(c.Param("message_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid message ID")
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return c.String(http.StatusUnauthorized, "Invalid user")
	}

	if h.messageService == nil {
		return c.String(http.StatusNotFound, "Message not found")
	}

	msg, err := h.messageService.GetMessage(c.Request().Context(), messageID)
	if err != nil {
		return c.String(http.StatusNotFound, "Message not found")
	}
	if shouldHideSystemTagCommand(msg) {
		return c.NoContent(http.StatusNoContent)
	}

	messageView := h.convertMessageToView(msg, userID)

	return h.renderPartial(c, "message", messageView)
}

// MessageEditForm returns the message edit form partial.
func (h *ChatTemplateHandler) MessageEditForm(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	messageID, err := uuid.ParseUUID(c.Param("message_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid message ID")
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return c.String(http.StatusUnauthorized, "Invalid user")
	}

	if h.messageService == nil {
		return c.String(http.StatusNotFound, "Message not found")
	}

	msg, err := h.messageService.GetMessage(c.Request().Context(), messageID)
	if err != nil {
		return c.String(http.StatusNotFound, "Message not found")
	}

	// Check if user can edit this message
	if msg.AuthorID() != userID {
		return c.String(http.StatusForbidden, "Cannot edit this message")
	}

	messageView := h.convertMessageToView(msg, userID)

	return h.renderPartial(c, "message_edit", messageView)
}

// ParticipantsPartial returns the participants panel as HTML partial.
func (h *ChatTemplateHandler) ParticipantsPartial(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	chatID, err := uuid.ParseUUID(c.Param("chat_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid chat ID")
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return c.String(http.StatusUnauthorized, "Invalid user")
	}

	participants := h.loadParticipants(c.Request().Context(), chatID, userID)

	// Check if current user can manage participants
	canManage := false
	for _, p := range participants {
		if p.UserID == userID.String() {
			if p.Role == roleCreator || p.Role == roleAdmin || p.Role == roleOwner {
				canManage = true
			}
			break
		}
	}

	data := map[string]any{
		"ChatID":       chatID.String(),
		"Participants": participants,
		"CanManage":    canManage,
	}

	return h.renderPartial(c, "chat/participants", data)
}

// ChatCreateForm returns the create chat form modal.
func (h *ChatTemplateHandler) ChatCreateForm(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	workspaceID := c.QueryParam("workspace_id")
	if workspaceID == "" {
		return c.String(http.StatusBadRequest, "Workspace ID required")
	}

	data := map[string]any{
		"WorkspaceID": workspaceID,
	}

	return h.renderPartial(c, "chat/create-form", data)
}

// ChatCreate handles POST /partials/chat/create and returns HTML partial.
func (h *ChatTemplateHandler) ChatCreate(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return h.modalError(c, http.StatusUnauthorized, "Unauthorized")
	}

	if h.chatService == nil {
		return h.modalError(c, http.StatusServiceUnavailable, "Service unavailable")
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return h.modalError(c, http.StatusBadRequest, "Invalid user ID")
	}

	workspaceIDStr := c.FormValue("workspace_id")
	workspaceID, err := uuid.ParseUUID(workspaceIDStr)
	if err != nil {
		return h.modalError(c, http.StatusBadRequest, "Invalid workspace ID")
	}

	name := c.FormValue("name")
	if name == "" {
		return h.modalError(c, http.StatusBadRequest, "Chat name is required")
	}

	chatType := c.FormValue("type")
	if chatType == "" {
		chatType = "discussion"
	}

	isPublic, _ := strconv.ParseBool(c.FormValue("is_public"))

	// Parse chat type to domain type
	var domainType chatdomain.Type
	switch chatType {
	case chatTypeTask:
		domainType = chatdomain.TypeTask
	case chatTypeBug:
		domainType = chatdomain.TypeBug
	case chatTypeEpic:
		domainType = chatdomain.TypeEpic
	default:
		domainType = chatdomain.TypeDiscussion
	}

	cmd := chatapp.CreateChatCommand{
		WorkspaceID: workspaceID,
		Title:       name,
		Type:        domainType,
		IsPublic:    isPublic,
		CreatedBy:   userID,
	}

	result, err := h.chatService.CreateChat(c.Request().Context(), cmd)
	if err != nil {
		h.logger.Error("failed to create chat", slog.String("error", err.Error()))
		return h.modalError(c, http.StatusInternalServerError, "Failed to create chat")
	}

	if domainType == chatdomain.TypeTask || domainType == chatdomain.TypeBug || domainType == chatdomain.TypeEpic {
		if h.taskProjector == nil {
			h.logger.Error("task projector is not configured for typed chat creation",
				slog.String("chat_id", result.Value.ID().String()),
				slog.String("type", string(domainType)),
			)
			return h.modalError(c, http.StatusServiceUnavailable, "Task projection unavailable")
		}
		if err = h.taskProjector.RebuildOne(c.Request().Context(), result.Value.ID()); err != nil {
			h.logger.Error("failed to sync task projection after typed chat creation",
				slog.String("chat_id", result.Value.ID().String()),
				slog.String("type", string(domainType)),
				slog.String("error", err.Error()),
			)
			return h.modalError(c, http.StatusInternalServerError, "Failed to sync task projection")
		}
	}

	// Redirect to the new chat
	chatURL := "/workspaces/" + workspaceID.String() + "/chats/" + result.Value.ID().String()

	//nolint:canonicalheader // HTMX uses non-canonical header names
	c.Response().Header().Set("HX-Redirect", chatURL)
	return c.NoContent(http.StatusOK)
}

func (h *ChatTemplateHandler) modalError(c echo.Context, statusCode int, message string) error {
	//nolint:canonicalheader // HTMX uses non-canonical header names
	c.Response().Header().Set("HX-Retarget", "#modal-container")
	return c.String(statusCode, `<div class="error">`+message+`</div>`)
}

// ChatSearchPartial returns filtered chat list based on search query.
func (h *ChatTemplateHandler) ChatSearchPartial(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	workspaceID, err := uuid.ParseUUID(c.QueryParam("workspace_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid workspace ID")
	}

	userID, err := uuid.ParseUUID(user.ID)
	if err != nil {
		return c.String(http.StatusUnauthorized, "Invalid user")
	}

	searchQuery := c.QueryParam("q")

	if h.chatService == nil {
		return h.renderPartial(c, "chat/list", map[string]any{
			"Chats":        []ChatViewData{},
			"ActiveChatID": "",
			"WorkspaceID":  workspaceID.String(),
		})
	}

	query := chatapp.ListChatsQuery{
		WorkspaceID: workspaceID,
		RequestedBy: userID,
		Limit:       defaultChatTemplateListLimit,
		Offset:      0,
	}

	result, err := h.chatService.ListChats(c.Request().Context(), query)
	if err != nil {
		h.logger.Error("failed to list chats", slog.String("error", err.Error()))
		return h.renderPartial(c, "chat/list", map[string]any{
			"Chats":        []ChatViewData{},
			"ActiveChatID": "",
			"WorkspaceID":  workspaceID.String(),
		})
	}

	// Filter chats by search query (simple contains match)
	chatViews := make([]ChatViewData, 0)
	for _, chat := range result.Chats {
		// Simple case-insensitive contains filter
		if searchQuery == "" || containsIgnoreCase(chat.Title, searchQuery) {
			chatViews = append(chatViews, ChatViewData{
				ID:          chat.ID.String(),
				WorkspaceID: chat.WorkspaceID.String(),
				Title:       chat.Title,
				Type:        string(chat.Type),
				IsPublic:    chat.IsPublic,
				IsTaskChat:  isTaskType(string(chat.Type)),
				CreatedAt:   chat.CreatedAt,
				UpdatedAt:   chat.CreatedAt,
				UnreadCount: 0,
			})
		}
	}

	data := map[string]any{
		"Chats":        chatViews,
		"ActiveChatID": "",
		"WorkspaceID":  workspaceID.String(),
	}

	return h.renderPartial(c, "chat/list", data)
}

// Helper methods

func (h *ChatTemplateHandler) getUserView(c echo.Context) *UserView {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return nil
	}

	email := middleware.GetEmail(c)
	username := middleware.GetUsername(c)

	// Build display name
	displayName := username
	if displayName == "" {
		displayName = email
	}

	return &UserView{
		ID:          userID.String(),
		Email:       email,
		Username:    username,
		DisplayName: displayName,
		AvatarURL:   "", // TODO: implement avatar
	}
}

func (h *ChatTemplateHandler) render(c echo.Context, template, title string, data any) error {
	user := h.getUserView(c)

	pageData := PageData{
		Title: title,
		User:  user,
		Data:  data,
	}

	return c.Render(http.StatusOK, template, pageData)
}

func (h *ChatTemplateHandler) renderPartial(c echo.Context, template string, data any) error {
	err := c.Render(http.StatusOK, template, data)
	if err != nil {
		h.logger.Error("template render failed",
			slog.String("template", template),
			slog.String("error", err.Error()))
	}
	return err
}

func (h *ChatTemplateHandler) renderNotFound(c echo.Context) error {
	return c.Render(http.StatusNotFound, "404.html", nil)
}

func (h *ChatTemplateHandler) loadChatViewData(
	ctx context.Context,
	chatID, userID uuid.UUID,
) (*ChatViewData, error) {
	if h.chatService == nil {
		return nil, chatapp.ErrChatNotFound
	}

	query := chatapp.GetChatQuery{
		ChatID:      chatID,
		RequestedBy: userID,
	}

	result, err := h.chatService.GetChat(ctx, query)
	if err != nil {
		return nil, err
	}

	if result == nil || result.Chat == nil {
		return nil, chatapp.ErrChatNotFound
	}

	chat := result.Chat
	assigneeID := ""
	if chat.AssignedTo != nil {
		assigneeID = chat.AssignedTo.String()
	}
	priority := ""
	if chat.Priority != nil {
		priority = *chat.Priority
	}
	var dueDate *time.Time
	if chat.DueDate != nil {
		d := *chat.DueDate
		dueDate = &d
	}

	return &ChatViewData{
		ID:               chat.ID.String(),
		WorkspaceID:      chat.WorkspaceID.String(),
		Title:            chat.Title,
		Type:             string(chat.Type),
		IsPublic:         chat.IsPublic,
		IsTaskChat:       isTaskType(string(chat.Type)),
		Status:           getStringValue(chat.Status),
		AssigneeID:       assigneeID,
		Priority:         priority,
		DueDate:          dueDate,
		CreatedBy:        chat.CreatedBy.String(),
		CreatedAt:        chat.CreatedAt,
		UpdatedAt:        chat.CreatedAt,
		ParticipantCount: len(chat.Participants),
		UnreadCount:      0,
	}, nil
}

func (h *ChatTemplateHandler) loadTaskViewData(ctx context.Context, chat *ChatViewData) *TaskViewData {
	if chat == nil || !chat.IsTaskChat {
		return nil
	}

	// Prefer chat read-model fields as source of truth for sidebar values,
	// because task read model can lag behind chat action processing.
	taskView := &TaskViewData{
		ID:         chat.ID,
		Status:     chat.Status,
		Priority:   chat.Priority,
		AssigneeID: chat.AssigneeID,
		DueDate:    chat.DueDate,
	}

	// Load task from task service using chat ID (primarily to get task ID and fallback values)
	if h.taskService == nil {
		return taskView
	}

	chatID, err := uuid.ParseUUID(chat.ID)
	if err != nil {
		return taskView
	}

	task, taskErr := h.taskService.GetTaskByChatID(ctx, chatID)
	if taskErr != nil || task == nil {
		return taskView
	}

	taskView.ID = task.ID.String()
	if taskView.Status == "" {
		taskView.Status = string(task.Status)
	}
	if taskView.Priority == "" {
		taskView.Priority = string(task.Priority)
	}
	if taskView.AssigneeID == "" {
		taskView.AssigneeID = getAssigneeID(task.AssignedTo)
	}
	if taskView.DueDate == nil && task.DueDate != nil {
		taskView.DueDate = task.DueDate
	}

	return taskView
}

func getAssigneeID(assignee *uuid.UUID) string {
	if assignee == nil {
		return ""
	}
	return assignee.String()
}

func (h *ChatTemplateHandler) loadParticipants(
	ctx context.Context,
	chatID uuid.UUID,
	userID ...uuid.UUID,
) []ParticipantViewData {
	if h.chatService == nil {
		return []ParticipantViewData{}
	}

	query := chatapp.GetChatQuery{ChatID: chatID}
	if len(userID) > 0 {
		query.RequestedBy = userID[0]
	}
	result, err := h.chatService.GetChat(ctx, query)
	if err != nil || result == nil || result.Chat == nil {
		h.logger.ErrorContext(
			ctx,
			"failed to load participants",
			slog.String("chat_id", chatID.String()),
			slog.Any("error", err),
		)
		return []ParticipantViewData{}
	}

	participants := make([]ParticipantViewData, 0, len(result.Chat.Participants))
	for _, p := range result.Chat.Participants {
		pv := ParticipantViewData{
			UserID:   p.UserID.String(),
			Role:     string(p.Role),
			JoinedAt: p.JoinedAt,
		}
		if h.userLookup != nil {
			if u := h.userLookup.GetUser(ctx, p.UserID); u != nil {
				pv.Username = u.Username
				pv.DisplayName = u.DisplayName
				pv.AvatarURL = u.AvatarURL
			}
		}
		if pv.Username == "" {
			pv.Username = "user" + p.UserID.String()[:8]
			pv.DisplayName = "User " + p.UserID.String()[:8]
		}
		participants = append(participants, pv)
	}
	return participants
}

const maxWorkspaceMembersForChat = 100

// loadWorkspaceMembers loads all workspace members for use in the task sidebar assignee dropdown.
func (h *ChatTemplateHandler) loadWorkspaceMembers(ctx context.Context, workspaceID uuid.UUID) []MemberViewData {
	if h.memberService == nil {
		return nil
	}
	members, err := h.memberService.ListWorkspaceMembers(ctx, workspaceID, 0, maxWorkspaceMembersForChat)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to load workspace members",
			slog.String("workspace_id", workspaceID.String()),
			slog.String("error", err.Error()),
		)
		return nil
	}
	return members
}

func (h *ChatTemplateHandler) convertMessageToView(msg *message.Message, currentUserID uuid.UUID) MessageViewData {
	if msg == nil {
		return MessageViewData{}
	}

	// Check message type
	isBotMessage := msg.IsBotMessage()
	isSystemMessage := msg.IsSystemMessage()

	// Check if current user can edit this message (bot and system messages cannot be edited)
	canEdit := msg.AuthorID() == currentUserID && !msg.IsDeleted() && !isBotMessage && !isSystemMessage

	// Convert reactions to view data
	reactions := make([]MessageReactionData, 0)
	reactionMap := make(map[string]*MessageReactionData)
	for _, r := range msg.Reactions() {
		if existing, ok := reactionMap[r.EmojiCode()]; ok {
			existing.Count++
			existing.Users = append(existing.Users, r.UserID().String())
			if r.UserID() == currentUserID {
				existing.HasReacted = true
			}
		} else {
			reactionMap[r.EmojiCode()] = &MessageReactionData{
				Emoji:      r.EmojiCode(),
				Count:      1,
				HasReacted: r.UserID() == currentUserID,
				Users:      []string{r.UserID().String()},
			}
		}
	}
	for _, r := range reactionMap {
		reactions = append(reactions, *r)
	}

	// Handle author display based on message type
	authorID := msg.AuthorID().String()
	var username, displayName string

	if isBotMessage {
		// Bot messages show as "Flowra Bot"
		username = "FlowraBot"
		displayName = "Flowra Bot"
	} else if h.userLookup != nil {
		if u := h.userLookup.GetUser(context.Background(), msg.AuthorID()); u != nil {
			username = u.Username
			displayName = u.DisplayName
		}
	}
	if username == "" && !isBotMessage {
		username = authorID[:8]
		displayName = "User " + username
	}

	// Parse tags and get display content
	parsed := parseMessageContent(msg.Content())

	// Convert attachments to view data
	attachments := make([]AttachmentViewData, 0)
	for _, a := range msg.Attachments() {
		attachments = append(attachments, AttachmentViewData{
			FileID:   a.FileID().String(),
			FileName: a.FileName(),
			FileSize: a.FileSize(),
			MimeType: a.MimeType(),
			URL:      fmt.Sprintf("/api/v1/files/%s/%s", a.FileID().String(), a.FileName()),
			IsImage:  strings.HasPrefix(a.MimeType(), "image/"),
		})
	}

	return MessageViewData{
		ID:              msg.ID().String(),
		ChatID:          msg.ChatID().String(),
		Content:         parsed.DisplayText,
		CreatedAt:       msg.CreatedAt(),
		EditedAt:        msg.EditedAt(),
		IsDeleted:       msg.IsDeleted(),
		IsSystemMessage: isSystemMessage,
		IsBotMessage:    isBotMessage,
		CanEdit:         canEdit,
		Author: MessageAuthorData{
			ID:          authorID,
			Username:    username,
			DisplayName: displayName,
			AvatarURL:   "",
		},
		Tags:        parsed.Tags,
		Reactions:   reactions,
		Attachments: attachments,
	}
}

// Utility functions

// parsedContent holds both the display content and parsed tags.
type parsedContent struct {
	DisplayText string
	Tags        []MessageTagData
}

// parseMessageContent parses tags from message content using the tag parser.
// Returns the plain text (for display) and the extracted tags.
func parseMessageContent(content string) parsedContent {
	parser := tag.NewParser()
	result := parser.Parse(content)

	tags := make([]MessageTagData, 0, len(result.Tags))
	for _, pt := range result.Tags {
		tags = append(tags, MessageTagData{
			Key:   pt.Key,
			Value: pt.Value,
		})
	}

	// Determine display text:
	// - If there's plain text, use it
	// - If only tags exist, show the value of the first tag (the main content)
	// - Otherwise fall back to original content
	displayText := result.PlainText
	if displayText == "" && len(tags) > 0 {
		// Use first tag's value as display text (e.g., "#task My Task" shows "My Task")
		displayText = tags[0].Value
	} else if displayText == "" {
		displayText = content
	}

	return parsedContent{
		DisplayText: displayText,
		Tags:        tags,
	}
}

// shouldHideSystemTagCommand hides internal command messages like "#status Done".
// Action responses are rendered by human-readable system/bot messages.
func shouldHideSystemTagCommand(msg *message.Message) bool {
	if msg == nil || !msg.IsSystemMessage() {
		return false
	}
	content := strings.TrimSpace(msg.Content())
	return strings.HasPrefix(content, "#")
}

func isTaskType(chatType string) bool {
	return chatType == chatTypeTask || chatType == chatTypeBug || chatType == chatTypeEpic
}

func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func containsIgnoreCase(s, substr string) bool {
	if substr == "" {
		return true
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// groupingThreshold is the maximum time between messages to be considered in the same group.
const groupingThreshold = 5 * time.Second

// isGroupableMessage returns true if the message can be grouped (system or bot).
func isGroupableMessage(msg *MessageViewData) bool {
	return msg.IsBotMessage || msg.IsSystemMessage
}

// canGroupWith returns true if two messages can be grouped together.
func canGroupWith(current, other *MessageViewData, threshold time.Duration) bool {
	return isGroupableMessage(other) && other.CreatedAt.Sub(current.CreatedAt).Abs() <= threshold
}

// applyMessageGrouping marks consecutive system/bot messages within 5 seconds as grouped.
// Sets IsGroupStart on the first message and IsGroupEnd on the last message of each group.
func applyMessageGrouping(messages []MessageViewData) {
	for i := range messages {
		msg := &messages[i]
		if !isGroupableMessage(msg) {
			continue
		}

		prevInGroup := i > 0 && i-1 < len(messages) && canGroupWith(msg, &messages[i-1], groupingThreshold)
		nextInGroup := i < len(messages)-1 && i+1 < len(messages) &&
			canGroupWith(msg, &messages[i+1], groupingThreshold)

		msg.IsGroupStart = !prevInGroup
		msg.IsGroupEnd = !nextInGroup
	}
}

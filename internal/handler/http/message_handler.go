package httphandler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	messageapp "github.com/lllypuk/flowra/internal/application/message"
	"github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Validation constants for message handler.
const (
	maxMessageContentLength = 10000
	defaultMessageListLimit = 50
	maxMessageListLimit     = 100
)

// Message handler errors.
var (
	ErrMessageNotFound    = errors.New("message not found")
	ErrNotMessageAuthor   = errors.New("only message author can edit")
	ErrMessageEmpty       = errors.New("message content cannot be empty")
	ErrMessageTooLong     = errors.New("message content is too long")
	ErrMessageDeleted     = errors.New("message is deleted")
	ErrParentNotFound     = errors.New("parent message not found")
	ErrNotChatParticipant = errors.New("not a participant of this chat")
)

// SendMessageRequest represents the request to send a message.
type SendMessageRequest struct {
	Content   string     `json:"content"     form:"content"`
	ReplyToID *uuid.UUID `json:"reply_to_id" form:"reply_to_id"`
}

// EditMessageRequest represents the request to edit a message.
type EditMessageRequest struct {
	Content string `json:"content" form:"content"`
}

// MessageResponse represents a message in API responses.
type MessageResponse struct {
	ID          uuid.UUID            `json:"id"`
	ChatID      uuid.UUID            `json:"chat_id"`
	SenderID    uuid.UUID            `json:"sender_id"`
	Content     string               `json:"content"`
	ReplyToID   *uuid.UUID           `json:"reply_to_id,omitempty"`
	CreatedAt   string               `json:"created_at"`
	EditedAt    *string              `json:"edited_at,omitempty"`
	IsDeleted   bool                 `json:"is_deleted"`
	Attachments []AttachmentResponse `json:"attachments,omitempty"`
	Reactions   []ReactionResponse   `json:"reactions,omitempty"`
}

// AttachmentResponse represents a message attachment in API responses.
type AttachmentResponse struct {
	FileID   uuid.UUID `json:"file_id"`
	FileName string    `json:"file_name"`
	FileSize int64     `json:"file_size"`
	MimeType string    `json:"mime_type"`
}

// ReactionResponse represents a message reaction in API responses.
type ReactionResponse struct {
	Emoji string      `json:"emoji"`
	Users []uuid.UUID `json:"users"`
	Count int         `json:"count"`
}

// MessageListResponse represents a list of messages in API responses.
type MessageListResponse struct {
	Messages   []MessageResponse `json:"messages"`
	HasMore    bool              `json:"has_more"`
	NextCursor *string           `json:"next_cursor,omitempty"`
}

// MessageService defines the interface for message operations.
// Declared on the consumer side per project guidelines.
type MessageService interface {
	// SendMessage sends a new message.
	SendMessage(ctx context.Context, cmd messageapp.SendMessageCommand) (messageapp.Result, error)

	// ListMessages lists messages in a chat.
	ListMessages(ctx context.Context, query messageapp.ListMessagesQuery) (messageapp.ListResult, error)

	// EditMessage edits a message.
	EditMessage(ctx context.Context, cmd messageapp.EditMessageCommand) (messageapp.Result, error)

	// DeleteMessage soft-deletes a message.
	DeleteMessage(ctx context.Context, cmd messageapp.DeleteMessageCommand) (messageapp.Result, error)

	// GetMessage gets a message by ID.
	GetMessage(ctx context.Context, messageID uuid.UUID) (*message.Message, error)
}

// MessageHandler handles message-related HTTP requests.
type MessageHandler struct {
	messageService MessageService
}

// NewMessageHandler creates a new MessageHandler.
func NewMessageHandler(messageService MessageService) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
	}
}

// RegisterRoutes registers message routes with the router.
func (h *MessageHandler) RegisterRoutes(r *httpserver.Router) {
	// Message operations (authenticated routes with chat/message ID)
	r.Auth().POST("/chats/:chat_id/messages", h.Send)
	r.Auth().GET("/chats/:chat_id/messages", h.List)
	r.Auth().PUT("/messages/:id", h.Edit)
	r.Auth().DELETE("/messages/:id", h.Delete)
}

// Send handles POST /api/v1/chats/:chat_id/messages.
// Sends a new message to the chat.
func (h *MessageHandler) Send(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatIDStr := c.Param("chat_id")
	chatID, parseErr := uuid.ParseUUID(chatIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	var req SendMessageRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	// Validate request
	if valErr := validateSendMessageRequest(&req); valErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "VALIDATION_ERROR", valErr.Error())
	}

	// Build command
	cmd := messageapp.SendMessageCommand{
		ChatID:   chatID,
		Content:  req.Content,
		AuthorID: userID,
	}
	if req.ReplyToID != nil && !req.ReplyToID.IsZero() {
		cmd.ParentMessageID = *req.ReplyToID
	}

	result, err := h.messageService.SendMessage(c.Request().Context(), cmd)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	resp := ToMessageResponse(result.Value)
	return httpserver.RespondCreated(c, resp)
}

// List handles GET /api/v1/chats/:chat_id/messages.
// Lists messages in a chat with pagination.
func (h *MessageHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatIDStr := c.Param("chat_id")
	chatID, parseErr := uuid.ParseUUID(chatIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	// Parse pagination
	limit, offset := parseMessagePagination(c)

	query := messageapp.ListMessagesQuery{
		ChatID: chatID,
		Limit:  limit,
		Offset: offset,
	}

	result, err := h.messageService.ListMessages(c.Request().Context(), query)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	// Build response
	messages := make([]MessageResponse, 0, len(result.Value))
	for _, msg := range result.Value {
		messages = append(messages, ToMessageResponse(msg))
	}

	// Determine if there are more messages
	hasMore := len(messages) == limit

	// Build cursor for next page
	var nextCursor *string
	if hasMore && len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		cursor := lastMsg.ID.String()
		nextCursor = &cursor
	}

	resp := MessageListResponse{
		Messages:   messages,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}

	return httpserver.RespondOK(c, resp)
}

// Edit handles PUT /api/v1/messages/:id.
// Edits a message.
func (h *MessageHandler) Edit(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	messageIDStr := c.Param("id")
	messageID, parseErr := uuid.ParseUUID(messageIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_MESSAGE_ID", "invalid message ID format")
	}

	var req EditMessageRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	// Validate request
	if valErr := validateEditMessageRequest(&req); valErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "VALIDATION_ERROR", valErr.Error())
	}

	cmd := messageapp.EditMessageCommand{
		MessageID: messageID,
		Content:   req.Content,
		EditorID:  userID,
	}

	result, err := h.messageService.EditMessage(c.Request().Context(), cmd)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	resp := ToMessageResponse(result.Value)
	return httpserver.RespondOK(c, resp)
}

// Delete handles DELETE /api/v1/messages/:id.
// Soft-deletes a message.
func (h *MessageHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	messageIDStr := c.Param("id")
	messageID, parseErr := uuid.ParseUUID(messageIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_MESSAGE_ID", "invalid message ID format")
	}

	cmd := messageapp.DeleteMessageCommand{
		MessageID: messageID,
		DeletedBy: userID,
	}

	_, deleteErr := h.messageService.DeleteMessage(c.Request().Context(), cmd)
	if deleteErr != nil {
		return httpserver.RespondError(c, deleteErr)
	}

	return httpserver.RespondNoContent(c)
}

// Helper functions

func validateSendMessageRequest(req *SendMessageRequest) error {
	if req.Content == "" {
		return ErrMessageEmpty
	}
	if len(req.Content) > maxMessageContentLength {
		return ErrMessageTooLong
	}
	return nil
}

func validateEditMessageRequest(req *EditMessageRequest) error {
	if req.Content == "" {
		return ErrMessageEmpty
	}
	if len(req.Content) > maxMessageContentLength {
		return ErrMessageTooLong
	}
	return nil
}

func parseMessagePagination(c echo.Context) (int, int) {
	limit := defaultMessageListLimit
	offset := 0

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = min(l, maxMessageListLimit)
		}
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Support cursor-based pagination via "before" parameter
	// The cursor is handled in the service layer

	return limit, offset
}

// ToMessageResponse converts a domain Message to MessageResponse.
func ToMessageResponse(msg *message.Message) MessageResponse {
	resp := MessageResponse{
		ID:        msg.ID(),
		ChatID:    msg.ChatID(),
		SenderID:  msg.AuthorID(),
		Content:   msg.Content(),
		CreatedAt: msg.CreatedAt().Format(time.RFC3339),
		IsDeleted: msg.IsDeleted(),
	}

	// Set reply to ID if it's a reply
	if msg.IsReply() {
		parentID := msg.ParentMessageID()
		resp.ReplyToID = &parentID
	}

	// Set edited at if edited
	if msg.IsEdited() {
		editedAt := msg.EditedAt().Format(time.RFC3339)
		resp.EditedAt = &editedAt
	}

	// Add attachments
	attachments := msg.Attachments()
	if len(attachments) > 0 {
		resp.Attachments = make([]AttachmentResponse, 0, len(attachments))
		for _, a := range attachments {
			resp.Attachments = append(resp.Attachments, AttachmentResponse{
				FileID:   a.FileID(),
				FileName: a.FileName(),
				FileSize: a.FileSize(),
				MimeType: a.MimeType(),
			})
		}
	}

	// Add reactions (grouped by emoji)
	reactions := msg.Reactions()
	if len(reactions) > 0 {
		reactionMap := make(map[string][]uuid.UUID)
		for _, r := range reactions {
			reactionMap[r.EmojiCode()] = append(reactionMap[r.EmojiCode()], r.UserID())
		}

		resp.Reactions = make([]ReactionResponse, 0, len(reactionMap))
		for emoji, users := range reactionMap {
			resp.Reactions = append(resp.Reactions, ReactionResponse{
				Emoji: emoji,
				Users: users,
				Count: len(users),
			})
		}
	}

	return resp
}

// MockMessageService is a mock implementation of MessageService for testing.
type MockMessageService struct {
	messages     map[uuid.UUID]*message.Message
	chatMessages map[uuid.UUID][]*message.Message
}

// NewMockMessageService creates a new mock message service.
func NewMockMessageService() *MockMessageService {
	return &MockMessageService{
		messages:     make(map[uuid.UUID]*message.Message),
		chatMessages: make(map[uuid.UUID][]*message.Message),
	}
}

// AddMessage adds a message to the mock service.
func (m *MockMessageService) AddMessage(msg *message.Message) {
	m.messages[msg.ID()] = msg
	m.chatMessages[msg.ChatID()] = append(m.chatMessages[msg.ChatID()], msg)
}

// SendMessage sends a message in the mock service.
func (m *MockMessageService) SendMessage(
	_ context.Context,
	cmd messageapp.SendMessageCommand,
) (messageapp.Result, error) {
	msg, err := message.NewMessage(cmd.ChatID, cmd.AuthorID, cmd.Content, cmd.ParentMessageID)
	if err != nil {
		return messageapp.Result{}, err
	}

	m.messages[msg.ID()] = msg
	m.chatMessages[cmd.ChatID] = append(m.chatMessages[cmd.ChatID], msg)

	return messageapp.Result{Value: msg}, nil
}

// ListMessages lists messages in the mock service.
func (m *MockMessageService) ListMessages(
	_ context.Context,
	query messageapp.ListMessagesQuery,
) (messageapp.ListResult, error) {
	msgs := m.chatMessages[query.ChatID]
	if msgs == nil {
		msgs = []*message.Message{}
	}

	// Apply pagination
	start := min(query.Offset, len(msgs))
	end := min(start+query.Limit, len(msgs))

	return messageapp.ListResult{Value: msgs[start:end]}, nil
}

// EditMessage edits a message in the mock service.
func (m *MockMessageService) EditMessage(
	_ context.Context,
	cmd messageapp.EditMessageCommand,
) (messageapp.Result, error) {
	msg, ok := m.messages[cmd.MessageID]
	if !ok {
		return messageapp.Result{}, messageapp.ErrMessageNotFound
	}

	if err := msg.EditContent(cmd.Content, cmd.EditorID); err != nil {
		return messageapp.Result{}, err
	}

	return messageapp.Result{Value: msg}, nil
}

// DeleteMessage deletes a message in the mock service.
func (m *MockMessageService) DeleteMessage(
	_ context.Context,
	cmd messageapp.DeleteMessageCommand,
) (messageapp.Result, error) {
	msg, ok := m.messages[cmd.MessageID]
	if !ok {
		return messageapp.Result{}, messageapp.ErrMessageNotFound
	}

	if err := msg.Delete(cmd.DeletedBy); err != nil {
		return messageapp.Result{}, err
	}

	return messageapp.Result{Value: msg}, nil
}

// GetMessage gets a message from the mock service.
func (m *MockMessageService) GetMessage(_ context.Context, messageID uuid.UUID) (*message.Message, error) {
	msg, ok := m.messages[messageID]
	if !ok {
		return nil, messageapp.ErrMessageNotFound
	}
	return msg, nil
}

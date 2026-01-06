package httphandler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/application/appcore"
	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Validation constants for chat handler.
const (
	maxChatNameLength      = 100
	defaultChatListLimit   = 20
	maxChatListLimit       = 100
	maxParticipantsPerChat = 100
)

// Chat type string constants for request parsing.
const (
	chatTypeTask = "task"
	chatTypeBug  = "bug"
	chatTypeEpic = "epic"
)

// Chat handler errors.
var (
	ErrChatNotFound          = errors.New("chat not found")
	ErrNotChatMember         = errors.New("not a member of this chat")
	ErrNotChatAdmin          = errors.New("admin access required")
	ErrCannotRemoveCreator   = errors.New("cannot remove chat creator")
	ErrInvalidChatType       = errors.New("invalid chat type")
	ErrParticipantNotFound   = errors.New("participant not found")
	ErrParticipantExists     = errors.New("participant already exists")
	ErrTooManyParticipants   = errors.New("too many participants")
	ErrDirectChatMaxMembers  = errors.New("direct chats cannot have more than 2 participants")
	ErrChatNameRequired      = errors.New("chat name is required")
	ErrChatNameTooLong       = errors.New("chat name is too long")
	ErrInvalidParticipantIDs = errors.New("invalid participant IDs")
)

// CreateChatRequest represents the request to create a chat.
type CreateChatRequest struct {
	Name           string      `json:"name"`
	Type           string      `json:"type"`
	IsPublic       bool        `json:"is_public"`
	ParticipantIDs []uuid.UUID `json:"participant_ids"`
}

// UpdateChatRequest represents the request to update a chat.
type UpdateChatRequest struct {
	Name string `json:"name"`
}

// AddParticipantRequest represents the request to add a participant.
type AddParticipantRequest struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
}

// ChatResponse represents a chat in API responses.
type ChatResponse struct {
	ID           uuid.UUID             `json:"id"`
	WorkspaceID  uuid.UUID             `json:"workspace_id"`
	Name         string                `json:"name"`
	Type         string                `json:"type"`
	IsPublic     bool                  `json:"is_public"`
	CreatedBy    uuid.UUID             `json:"created_by"`
	CreatedAt    string                `json:"created_at"`
	Participants []ParticipantResponse `json:"participants,omitempty"`
	// Task-specific fields
	Status     *string    `json:"status,omitempty"`
	AssignedTo *uuid.UUID `json:"assigned_to,omitempty"`
	Priority   *string    `json:"priority,omitempty"`
	DueDate    *string    `json:"due_date,omitempty"`
	// Bug-specific fields
	Severity *string `json:"severity,omitempty"`
}

// ParticipantResponse represents a chat participant in API responses.
type ParticipantResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt string    `json:"joined_at"`
}

// ChatListResponse represents a list of chats in API responses.
type ChatListResponse struct {
	Chats   []ChatResponse `json:"chats"`
	Total   int            `json:"total"`
	HasMore bool           `json:"has_more"`
}

// ChatService defines the interface for chat operations.
// Declared on the consumer side per project guidelines.
type ChatService interface {
	// CreateChat creates a new chat.
	CreateChat(ctx context.Context, cmd chatapp.CreateChatCommand) (chatapp.Result, error)

	// GetChat gets a chat by ID.
	GetChat(ctx context.Context, query chatapp.GetChatQuery) (*chatapp.GetChatResult, error)

	// ListChats lists chats in a workspace.
	ListChats(ctx context.Context, query chatapp.ListChatsQuery) (*chatapp.ListChatsResult, error)

	// RenameChat renames a chat.
	RenameChat(ctx context.Context, cmd chatapp.RenameChatCommand) (chatapp.Result, error)

	// AddParticipant adds a participant to a chat.
	AddParticipant(ctx context.Context, cmd chatapp.AddParticipantCommand) (chatapp.Result, error)

	// RemoveParticipant removes a participant from a chat.
	RemoveParticipant(ctx context.Context, cmd chatapp.RemoveParticipantCommand) (chatapp.Result, error)

	// DeleteChat deletes a chat (soft delete via event).
	DeleteChat(ctx context.Context, chatID, deletedBy uuid.UUID) error
}

// ChatHandler handles chat-related HTTP requests.
type ChatHandler struct {
	chatService ChatService
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(chatService ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}

// RegisterRoutes registers chat routes with the router.
func (h *ChatHandler) RegisterRoutes(r *httpserver.Router) {
	// Chat CRUD (workspace-scoped routes)
	r.Workspace().POST("/chats", h.Create)
	r.Workspace().GET("/chats", h.List)

	// Chat operations (authenticated routes with chat ID)
	r.Auth().GET("/chats/:id", h.Get)
	r.Auth().PUT("/chats/:id", h.Update)
	r.Auth().DELETE("/chats/:id", h.Delete)

	// Participant management
	r.Auth().POST("/chats/:id/participants", h.AddParticipant)
	r.Auth().DELETE("/chats/:id/participants/:user_id", h.RemoveParticipant)
}

// Create handles POST /api/v1/workspaces/:workspace_id/chats.
// Creates a new chat in the workspace.
func (h *ChatHandler) Create(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	workspaceIDStr := c.Param("workspace_id")
	workspaceID, parseErr := uuid.ParseUUID(workspaceIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "invalid workspace ID format")
	}

	var req CreateChatRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	// Validate request
	if valErr := validateCreateChatRequest(&req); valErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "VALIDATION_ERROR", valErr.Error())
	}

	// Parse chat type
	chatType := parseChatType(req.Type)
	if chatType == "" {
		chatType = chat.TypeDiscussion
	}

	// Create chat command
	cmd := chatapp.CreateChatCommand{
		WorkspaceID: workspaceID,
		Title:       req.Name,
		Type:        chatType,
		IsPublic:    req.IsPublic,
		CreatedBy:   userID,
	}

	result, err := h.chatService.CreateChat(c.Request().Context(), cmd)
	if err != nil {
		return handleChatError(c, err)
	}

	// Add participants if specified
	if len(req.ParticipantIDs) > 0 {
		chatAggregate := result.Value
		for _, participantID := range req.ParticipantIDs {
			if participantID == userID {
				continue // Creator is already added
			}
			addCmd := chatapp.AddParticipantCommand{
				ChatID:  chatAggregate.ID(),
				UserID:  participantID,
				Role:    chat.RoleMember,
				AddedBy: userID,
			}
			// Ignore errors for participant addition - chat is created
			_, _ = h.chatService.AddParticipant(c.Request().Context(), addCmd)
		}
	}

	// Build response
	resp := ToChatResponse(result.Value)
	return httpserver.RespondCreated(c, resp)
}

// Get handles GET /api/v1/chats/:id.
// Gets a chat by ID.
func (h *ChatHandler) Get(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatIDStr := c.Param("id")
	chatID, parseErr := uuid.ParseUUID(chatIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	query := chatapp.GetChatQuery{
		ChatID:      chatID,
		RequestedBy: userID,
	}

	result, err := h.chatService.GetChat(c.Request().Context(), query)
	if err != nil {
		return handleChatError(c, err)
	}

	resp := ToChatResponseFromDTO(result.Chat)
	return httpserver.RespondOK(c, resp)
}

// List handles GET /api/v1/workspaces/:workspace_id/chats.
// Lists chats in a workspace.
func (h *ChatHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	workspaceIDStr := c.Param("workspace_id")
	workspaceID, parseErr := uuid.ParseUUID(workspaceIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "invalid workspace ID format")
	}

	// Parse pagination
	limit, offset := parseChatPagination(c)

	// Parse type filter
	var typeFilter *chat.Type
	if typeStr := c.QueryParam("type"); typeStr != "" {
		t := parseChatType(typeStr)
		if t != "" {
			typeFilter = &t
		}
	}

	query := chatapp.ListChatsQuery{
		WorkspaceID: workspaceID,
		Type:        typeFilter,
		Limit:       limit,
		Offset:      offset,
		RequestedBy: userID,
	}

	result, err := h.chatService.ListChats(c.Request().Context(), query)
	if err != nil {
		return handleChatError(c, err)
	}

	// Build response
	chats := make([]ChatResponse, 0, len(result.Chats))
	for _, ch := range result.Chats {
		chats = append(chats, ToChatResponseFromDTO(&ch))
	}

	resp := ChatListResponse{
		Chats:   chats,
		Total:   result.Total,
		HasMore: result.HasMore,
	}

	return httpserver.RespondOK(c, resp)
}

// Update handles PUT /api/v1/chats/:id.
// Updates a chat (rename).
func (h *ChatHandler) Update(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatIDStr := c.Param("id")
	chatID, parseErr := uuid.ParseUUID(chatIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	var req UpdateChatRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	// Validate
	if req.Name == "" {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "VALIDATION_ERROR", ErrChatNameRequired.Error())
	}
	if len(req.Name) > maxChatNameLength {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "VALIDATION_ERROR", ErrChatNameTooLong.Error())
	}

	cmd := chatapp.RenameChatCommand{
		ChatID:    chatID,
		NewTitle:  req.Name,
		RenamedBy: userID,
	}

	result, err := h.chatService.RenameChat(c.Request().Context(), cmd)
	if err != nil {
		return handleChatError(c, err)
	}

	resp := ToChatResponse(result.Value)
	return httpserver.RespondOK(c, resp)
}

// Delete handles DELETE /api/v1/chats/:id.
// Deletes a chat.
func (h *ChatHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatIDStr := c.Param("id")
	chatID, parseErr := uuid.ParseUUID(chatIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	deleteErr := h.chatService.DeleteChat(c.Request().Context(), chatID, userID)
	if deleteErr != nil {
		return handleChatError(c, deleteErr)
	}

	return httpserver.RespondNoContent(c)
}

// AddParticipant handles POST /api/v1/chats/:id/participants.
// Adds a participant to the chat.
func (h *ChatHandler) AddParticipant(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatIDStr := c.Param("id")
	chatID, parseErr := uuid.ParseUUID(chatIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	var req AddParticipantRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	if req.UserID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "VALIDATION_ERROR", "user_id is required")
	}

	// Parse role
	role := parseParticipantRole(req.Role)
	if role == "" {
		role = chat.RoleMember
	}

	cmd := chatapp.AddParticipantCommand{
		ChatID:  chatID,
		UserID:  req.UserID,
		Role:    role,
		AddedBy: userID,
	}

	result, err := h.chatService.AddParticipant(c.Request().Context(), cmd)
	if err != nil {
		return handleChatError(c, err)
	}

	resp := ToChatResponse(result.Value)
	return httpserver.RespondCreated(c, resp)
}

// RemoveParticipant handles DELETE /api/v1/chats/:id/participants/:user_id.
// Removes a participant from the chat.
func (h *ChatHandler) RemoveParticipant(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatIDStr := c.Param("id")
	chatID, parseErr := uuid.ParseUUID(chatIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	participantIDStr := c.Param("user_id")
	participantID, parseErr2 := uuid.ParseUUID(participantIDStr)
	if parseErr2 != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_USER_ID", "invalid user ID format")
	}

	cmd := chatapp.RemoveParticipantCommand{
		ChatID:    chatID,
		UserID:    participantID,
		RemovedBy: userID,
	}

	_, removeErr := h.chatService.RemoveParticipant(c.Request().Context(), cmd)
	if removeErr != nil {
		return handleChatError(c, removeErr)
	}

	return httpserver.RespondNoContent(c)
}

// Helper functions

func validateCreateChatRequest(req *CreateChatRequest) error {
	// Name validation for non-discussion types
	if req.Type != "" && req.Type != "discussion" {
		if req.Name == "" {
			return ErrChatNameRequired
		}
	}
	if len(req.Name) > maxChatNameLength {
		return ErrChatNameTooLong
	}
	if len(req.ParticipantIDs) > maxParticipantsPerChat {
		return ErrTooManyParticipants
	}
	// Validate type if provided
	if req.Type != "" {
		validTypes := []string{"discussion", chatTypeTask, chatTypeBug, chatTypeEpic, "direct", "group", "channel"}
		valid := false
		for _, t := range validTypes {
			if req.Type == t {
				valid = true
				break
			}
		}
		if !valid {
			return ErrInvalidChatType
		}
	}
	return nil
}

func parseChatType(typeStr string) chat.Type {
	switch typeStr {
	case "discussion":
		return chat.TypeDiscussion
	case chatTypeTask:
		return chat.TypeTask
	case chatTypeBug:
		return chat.TypeBug
	case chatTypeEpic:
		return chat.TypeEpic
	case "direct", "group", "channel":
		// Map legacy types to discussion
		return chat.TypeDiscussion
	default:
		return ""
	}
}

func parseParticipantRole(roleStr string) chat.Role {
	switch roleStr {
	case "admin":
		return chat.RoleAdmin
	case "member":
		return chat.RoleMember
	default:
		return ""
	}
}

func parseChatPagination(c echo.Context) (int, int) {
	limit := defaultChatListLimit
	offset := 0

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > maxChatListLimit {
				limit = maxChatListLimit
			}
		}
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	return limit, offset
}

func handleChatError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, chatapp.ErrChatNotFound):
		return httpserver.RespondErrorWithCode(c, http.StatusNotFound, "CHAT_NOT_FOUND", "chat not found")
	case errors.Is(err, chatapp.ErrUserNotParticipant):
		return httpserver.RespondErrorWithCode(c, http.StatusForbidden, "NOT_MEMBER", "not a member of this chat")
	case errors.Is(err, chatapp.ErrNotAdmin):
		return httpserver.RespondErrorWithCode(c, http.StatusForbidden, "NOT_ADMIN", "admin access required")
	case errors.Is(err, chatapp.ErrCannotRemoveCreator):
		return httpserver.RespondErrorWithCode(
			c, http.StatusForbidden, "CANNOT_REMOVE_CREATOR", "cannot remove chat creator")
	case errors.Is(err, chatapp.ErrUserAlreadyParticipant):
		return httpserver.RespondErrorWithCode(
			c, http.StatusConflict, "PARTICIPANT_EXISTS", "participant already exists")
	case errors.Is(err, chatapp.ErrInvalidChatType):
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_CHAT_TYPE", "invalid chat type")
	case errors.Is(err, chatapp.ErrTitleRequired):
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "TITLE_REQUIRED", "title is required for typed chats")
	case errors.Is(err, chatapp.ErrForbidden):
		return httpserver.RespondErrorWithCode(c, http.StatusForbidden, "FORBIDDEN", "access denied")
	default:
		return httpserver.RespondError(c, err)
	}
}

// ToChatResponse converts a domain Chat to ChatResponse.
func ToChatResponse(ch *chat.Chat) ChatResponse {
	resp := ChatResponse{
		ID:          ch.ID(),
		WorkspaceID: ch.WorkspaceID(),
		Name:        ch.Title(),
		Type:        string(ch.Type()),
		IsPublic:    ch.IsPublic(),
		CreatedBy:   ch.CreatedBy(),
		CreatedAt:   ch.CreatedAt().Format(time.RFC3339),
	}

	// Add participants
	participants := ch.Participants()
	resp.Participants = make([]ParticipantResponse, 0, len(participants))
	for _, p := range participants {
		resp.Participants = append(resp.Participants, ParticipantResponse{
			UserID:   p.UserID(),
			Role:     string(p.Role()),
			JoinedAt: p.JoinedAt().Format(time.RFC3339),
		})
	}

	// Add task-specific fields
	if ch.Type() == chat.TypeTask || ch.Type() == chat.TypeBug || ch.Type() == chat.TypeEpic {
		status := ch.Status()
		resp.Status = &status

		if assignee := ch.AssigneeID(); assignee != nil {
			resp.AssignedTo = assignee
		}

		if priority := ch.Priority(); priority != "" {
			resp.Priority = &priority
		}

		if dueDate := ch.DueDate(); dueDate != nil {
			formatted := dueDate.Format(time.RFC3339)
			resp.DueDate = &formatted
		}
	}

	// Add bug-specific fields
	if ch.Type() == chat.TypeBug {
		if severity := ch.Severity(); severity != "" {
			resp.Severity = &severity
		}
	}

	return resp
}

// ToChatResponseFromDTO converts a chatapp.Chat DTO to ChatResponse.
func ToChatResponseFromDTO(ch *chatapp.Chat) ChatResponse {
	resp := ChatResponse{
		ID:          ch.ID,
		WorkspaceID: ch.WorkspaceID,
		Name:        ch.Title,
		Type:        string(ch.Type),
		IsPublic:    ch.IsPublic,
		CreatedBy:   ch.CreatedBy,
		CreatedAt:   ch.CreatedAt.Format(time.RFC3339),
	}

	// Add participants
	resp.Participants = make([]ParticipantResponse, 0, len(ch.Participants))
	for _, p := range ch.Participants {
		resp.Participants = append(resp.Participants, ParticipantResponse{
			UserID:   p.UserID,
			Role:     string(p.Role),
			JoinedAt: p.JoinedAt.Format(time.RFC3339),
		})
	}

	// Add task-specific fields
	if ch.Status != nil {
		resp.Status = ch.Status
	}
	if ch.AssignedTo != nil {
		resp.AssignedTo = ch.AssignedTo
	}
	if ch.Priority != nil {
		resp.Priority = ch.Priority
	}
	if ch.DueDate != nil {
		formatted := ch.DueDate.Format(time.RFC3339)
		resp.DueDate = &formatted
	}
	if ch.Severity != nil {
		resp.Severity = ch.Severity
	}

	return resp
}

// MockChatService is a mock implementation of ChatService for testing.
type MockChatService struct {
	chats        map[uuid.UUID]*chat.Chat
	participants map[uuid.UUID][]chat.Participant
}

// NewMockChatService creates a new mock chat service.
func NewMockChatService() *MockChatService {
	return &MockChatService{
		chats:        make(map[uuid.UUID]*chat.Chat),
		participants: make(map[uuid.UUID][]chat.Participant),
	}
}

// AddChat adds a chat to the mock service.
func (m *MockChatService) AddChat(ch *chat.Chat) {
	m.chats[ch.ID()] = ch
}

// CreateChat creates a new chat in the mock service.
func (m *MockChatService) CreateChat(_ context.Context, cmd chatapp.CreateChatCommand) (chatapp.Result, error) {
	ch, err := chat.NewChat(cmd.WorkspaceID, cmd.Type, cmd.IsPublic, cmd.CreatedBy)
	if err != nil {
		return chatapp.Result{}, err
	}

	if cmd.Title != "" && cmd.Type != chat.TypeDiscussion {
		switch cmd.Type {
		case chat.TypeTask:
			_ = ch.ConvertToTask(cmd.Title, cmd.CreatedBy)
		case chat.TypeBug:
			_ = ch.ConvertToBug(cmd.Title, cmd.CreatedBy)
		case chat.TypeEpic:
			_ = ch.ConvertToEpic(cmd.Title, cmd.CreatedBy)
		case chat.TypeDiscussion:
			// Already handled above
		}
	}

	m.chats[ch.ID()] = ch
	return chatapp.Result{Result: appcore.Result[*chat.Chat]{Value: ch}}, nil
}

// GetChat gets a chat from the mock service.
func (m *MockChatService) GetChat(_ context.Context, query chatapp.GetChatQuery) (*chatapp.GetChatResult, error) {
	ch, ok := m.chats[query.ChatID]
	if !ok {
		return nil, chatapp.ErrChatNotFound
	}

	// Check access
	if !ch.IsPublic() && !ch.HasParticipant(query.RequestedBy) {
		return nil, chatapp.ErrUserNotParticipant
	}

	// Build DTO
	dto := &chatapp.Chat{
		ID:          ch.ID(),
		WorkspaceID: ch.WorkspaceID(),
		Type:        ch.Type(),
		Title:       ch.Title(),
		IsPublic:    ch.IsPublic(),
		CreatedBy:   ch.CreatedBy(),
		CreatedAt:   ch.CreatedAt(),
		Version:     ch.Version(),
	}

	// Add participants
	for _, p := range ch.Participants() {
		dto.Participants = append(dto.Participants, chatapp.Participant{
			UserID:   p.UserID(),
			Role:     p.Role(),
			JoinedAt: p.JoinedAt(),
		})
	}

	return &chatapp.GetChatResult{
		Chat: dto,
		Permissions: chatapp.Permissions{
			CanRead:   true,
			CanWrite:  ch.HasParticipant(query.RequestedBy),
			CanManage: ch.CreatedBy() == query.RequestedBy || ch.IsParticipantAdmin(query.RequestedBy),
		},
	}, nil
}

// ListChats lists chats from the mock service.
func (m *MockChatService) ListChats(_ context.Context, query chatapp.ListChatsQuery) (*chatapp.ListChatsResult, error) {
	var chats []chatapp.Chat

	for _, ch := range m.chats {
		if ch.WorkspaceID() != query.WorkspaceID {
			continue
		}
		if query.Type != nil && ch.Type() != *query.Type {
			continue
		}
		if !ch.IsPublic() && !ch.HasParticipant(query.RequestedBy) {
			continue
		}

		chats = append(chats, chatapp.Chat{
			ID:          ch.ID(),
			WorkspaceID: ch.WorkspaceID(),
			Type:        ch.Type(),
			Title:       ch.Title(),
			IsPublic:    ch.IsPublic(),
			CreatedBy:   ch.CreatedBy(),
			CreatedAt:   ch.CreatedAt(),
		})
	}

	// Apply pagination
	total := len(chats)
	start := query.Offset
	if start > total {
		start = total
	}
	end := start + query.Limit
	if end > total {
		end = total
	}

	return &chatapp.ListChatsResult{
		Chats:   chats[start:end],
		Total:   total,
		HasMore: end < total,
	}, nil
}

// RenameChat renames a chat in the mock service.
func (m *MockChatService) RenameChat(_ context.Context, cmd chatapp.RenameChatCommand) (chatapp.Result, error) {
	ch, ok := m.chats[cmd.ChatID]
	if !ok {
		return chatapp.Result{}, chatapp.ErrChatNotFound
	}

	if err := ch.Rename(cmd.NewTitle, cmd.RenamedBy); err != nil {
		return chatapp.Result{}, err
	}

	return chatapp.Result{Result: appcore.Result[*chat.Chat]{Value: ch}}, nil
}

// AddParticipant adds a participant to a chat in the mock service.
func (m *MockChatService) AddParticipant(_ context.Context, cmd chatapp.AddParticipantCommand) (chatapp.Result, error) {
	ch, ok := m.chats[cmd.ChatID]
	if !ok {
		return chatapp.Result{}, chatapp.ErrChatNotFound
	}

	if err := ch.AddParticipant(cmd.UserID, cmd.Role); err != nil {
		return chatapp.Result{}, err
	}

	return chatapp.Result{Result: appcore.Result[*chat.Chat]{Value: ch}}, nil
}

// RemoveParticipant removes a participant from a chat in the mock service.
func (m *MockChatService) RemoveParticipant(
	_ context.Context,
	cmd chatapp.RemoveParticipantCommand,
) (chatapp.Result, error) {
	ch, ok := m.chats[cmd.ChatID]
	if !ok {
		return chatapp.Result{}, chatapp.ErrChatNotFound
	}

	if err := ch.RemoveParticipant(cmd.UserID); err != nil {
		return chatapp.Result{}, err
	}

	return chatapp.Result{Result: appcore.Result[*chat.Chat]{Value: ch}}, nil
}

// DeleteChat deletes a chat from the mock service.
func (m *MockChatService) DeleteChat(_ context.Context, chatID, _ uuid.UUID) error {
	if _, ok := m.chats[chatID]; !ok {
		return chatapp.ErrChatNotFound
	}
	delete(m.chats, chatID)
	return nil
}

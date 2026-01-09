//go:build e2e

package e2e

// Common API response wrapper types for E2E tests.
// These types reduce inline struct duplication across test files.

// APIResponse is the generic wrapper for all API responses.
type APIResponse[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// Error represents API error response.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// --- Workspace Types ---

// WorkspaceData represents workspace in API responses.
type WorkspaceData struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     string `json:"owner_id"`
	CreatedAt   string `json:"created_at"`
}

// WorkspaceListData represents workspace list API response.
type WorkspaceListData struct {
	Workspaces []WorkspaceData `json:"workspaces"`
	Total      int             `json:"total"`
}

// MemberData represents workspace member in API responses.
type MemberData struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

// MemberListData represents member list API response.
type MemberListData struct {
	Members []MemberData `json:"members"`
	Total   int          `json:"total"`
}

// --- Chat Types ---

// ParticipantData represents chat participant.
type ParticipantData struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// ChatData represents chat in API responses.
type ChatData struct {
	ID           string            `json:"id"`
	WorkspaceID  string            `json:"workspace_id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	IsPublic     bool              `json:"is_public"`
	CreatedBy    string            `json:"created_by"`
	Participants []ParticipantData `json:"participants"`
}

// ChatListData represents chat list API response.
type ChatListData struct {
	Chats []ChatData `json:"chats"`
	Total int        `json:"total"`
}

// --- Task Types ---

// TaskData represents task in API responses.
type TaskData struct {
	ID          string  `json:"id"`
	ChatID      string  `json:"chat_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	EntityType  string  `json:"entity_type"`
	ReporterID  string  `json:"reporter_id"`
	AssigneeID  *string `json:"assignee_id"`
	DueDate     *string `json:"due_date"`
	CreatedAt   string  `json:"created_at"`
	Version     int     `json:"version"`
}

// TaskListData represents task list API response.
type TaskListData struct {
	Tasks []TaskData `json:"tasks"`
	Total int        `json:"total"`
}

// --- Message Types ---

// MessageData represents message in API responses.
type MessageData struct {
	ID        string  `json:"id"`
	ChatID    string  `json:"chat_id"`
	Content   string  `json:"content"`
	AuthorID  string  `json:"author_id"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt *string `json:"updated_at,omitempty"`
	IsEdited  bool    `json:"is_edited"`
}

// MessageListData represents message list API response.
type MessageListData struct {
	Messages []MessageData `json:"messages"`
	Total    int           `json:"total"`
	HasMore  bool          `json:"has_more"`
}

// --- Auth Types ---

// AuthData represents authentication response.
type AuthData struct {
	Token        string   `json:"token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int      `json:"expires_in"`
	TokenType    string   `json:"token_type"`
	User         UserData `json:"user"`
}

// UserData represents user in API responses.
type UserData struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// UserProfileData represents user profile in API responses.
type UserProfileData struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	AvatarURL string `json:"avatar_url"`
}

// --- Notification Types ---

// NotificationData represents notification in API responses.
type NotificationData struct {
	ID         string  `json:"id"`
	UserID     string  `json:"user_id"`
	Title      string  `json:"title"`
	Message    string  `json:"message"`
	Type       string  `json:"type"`
	ResourceID *string `json:"resource_id,omitempty"`
	IsRead     bool    `json:"is_read"`
	CreatedAt  string  `json:"created_at"`
}

// NotificationListData represents notification list API response.
type NotificationListData struct {
	Notifications []NotificationData `json:"notifications"`
	Total         int                `json:"total"`
	UnreadCount   int                `json:"unread_count"`
}

// --- Type aliases for common responses ---

type WorkspaceResponse = APIResponse[WorkspaceData]
type WorkspaceListResponse = APIResponse[WorkspaceListData]
type MemberListResponse = APIResponse[MemberListData]
type ChatResponse = APIResponse[ChatData]
type ChatListResponse = APIResponse[ChatListData]
type TaskResponse = APIResponse[TaskData]
type TaskListResponse = APIResponse[TaskListData]
type MessageResponse = APIResponse[MessageData]
type MessageListResponse = APIResponse[MessageListData]
type AuthResponse = APIResponse[AuthData]
type UserResponse = APIResponse[UserData]
type UserProfileResponse = APIResponse[UserProfileData]
type NotificationResponse = APIResponse[NotificationData]
type NotificationListResponse = APIResponse[NotificationListData]
type ErrorResponse = APIResponse[any]

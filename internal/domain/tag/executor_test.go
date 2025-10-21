package tag_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/lllypuk/flowra/internal/domain/tag"
	"github.com/lllypuk/flowra/internal/domain/user"
)

// MockUserRepository mocks the UserRepository interface
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username string) (*user.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

// TestCommandExecutor_AssignUser_UserNotFound tests that assign user command fails
// when username cannot be resolved
func TestCommandExecutor_AssignUser_UserNotFound(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)

	// Create empty ChatUseCases - we're only testing user resolution, not UseCase execution
	chatUseCases := &tag.ChatUseCases{}

	executor := tag.NewCommandExecutor(chatUseCases, mockUserRepo)

	ctx := context.Background()
	chatID := uuid.New()
	actorID := uuid.New()

	// Mock user not found
	mockUserRepo.On("FindByUsername", ctx, "nonexistent").
		Return(nil, errors.New("user not found"))

	cmd := tag.AssignUserCommand{
		ChatID:   chatID,
		Username: "@nonexistent",
	}

	// Act
	err := executor.Execute(ctx, cmd, actorID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user @nonexistent not found")
	mockUserRepo.AssertCalled(t, "FindByUsername", ctx, "nonexistent")
}

// TestCommandExecutor_AssignUser_TrimPrefix tests that @-prefix is properly trimmed
func TestCommandExecutor_AssignUser_TrimPrefix(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)

	// Create empty ChatUseCases
	chatUseCases := &tag.ChatUseCases{}

	executor := tag.NewCommandExecutor(chatUseCases, mockUserRepo)

	ctx := context.Background()
	chatID := uuid.New()
	actorID := uuid.New()

	cmd := tag.AssignUserCommand{
		ChatID:   chatID,
		Username: "@testuser",
	}

	// Mock user not found - important to verify the username was trimmed correctly
	mockUserRepo.On("FindByUsername", ctx, "testuser").
		Return(nil, errors.New("user not found"))

	// Act
	err := executor.Execute(ctx, cmd, actorID)

	// Assert - User resolution should be called with trimmed username
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "@testuser not found")
	mockUserRepo.AssertCalled(t, "FindByUsername", ctx, "testuser")
}

// InvalidCommand for testing unknown command type
type InvalidCommand struct{}

func (c InvalidCommand) CommandType() string {
	return "InvalidCommand"
}

// TestCommandExecutor_UnknownCommand tests that unknown command type returns error
func TestCommandExecutor_UnknownCommand(t *testing.T) {
	// Arrange
	chatUseCases := &tag.ChatUseCases{}
	mockUserRepo := new(MockUserRepository)

	executor := tag.NewCommandExecutor(chatUseCases, mockUserRepo)

	ctx := context.Background()
	actorID := uuid.New()

	// Create an invalid command
	var cmd tag.Command = InvalidCommand{}

	// Act
	err := executor.Execute(ctx, cmd, actorID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command type")
}

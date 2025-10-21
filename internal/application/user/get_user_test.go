package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lllypuk/flowra/internal/application/user"
	domainuser "github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

func TestGetUserUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewGetUserUseCase(repo)

	// Создаем пользователя
	existingUser, _ := domainuser.NewUser("external-123", "testuser", "test@example.com", "Test User")
	_ = repo.Save(context.Background(), existingUser)

	query := user.GetUserQuery{
		UserID: existingUser.ID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Value == nil {
		t.Fatal("expected user to be found")
	}

	if result.Value.ID() != existingUser.ID() {
		t.Errorf("expected userID %s, got %s", existingUser.ID(), result.Value.ID())
	}

	if result.Value.Username() != "testuser" {
		t.Errorf("expected username 'testuser', got %s", result.Value.Username())
	}
}

func TestGetUserUseCase_Execute_UserNotFound(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewGetUserUseCase(repo)

	nonExistentUser, _ := domainuser.NewUser("external-123", "test", "test@example.com", "Test")

	query := user.GetUserQuery{
		UserID: nonExistentUser.ID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if !errors.Is(err, user.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got: %v", err)
	}
}

func TestGetUserUseCase_Validate_InvalidUserID(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewGetUserUseCase(repo)

	query := user.GetUserQuery{
		UserID: uuid.UUID(""), // invalid
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid userID")
	}
}

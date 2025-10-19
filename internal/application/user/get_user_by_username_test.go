package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lllypuk/teams-up/internal/application/user"
	domainuser "github.com/lllypuk/teams-up/internal/domain/user"
)

func TestGetUserByUsernameUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewGetUserByUsernameUseCase(repo)

	// Создаем пользователя
	existingUser, _ := domainuser.NewUser("external-123", "testuser", "test@example.com", "Test User")
	_ = repo.Save(context.Background(), existingUser)

	query := user.GetUserByUsernameQuery{
		Username: "testuser",
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

	if result.Value.Username() != "testuser" {
		t.Errorf("expected username 'testuser', got %s", result.Value.Username())
	}

	if result.Value.Email() != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %s", result.Value.Email())
	}
}

func TestGetUserByUsernameUseCase_Execute_UserNotFound(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewGetUserByUsernameUseCase(repo)

	query := user.GetUserByUsernameQuery{
		Username: "nonexistent",
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if !errors.Is(err, user.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got: %v", err)
	}
}

func TestGetUserByUsernameUseCase_Validate_EmptyUsername(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewGetUserByUsernameUseCase(repo)

	query := user.GetUserByUsernameQuery{
		Username: "",
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for empty username")
	}
}

func TestGetUserByUsernameUseCase_Execute_CaseSensitive(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewGetUserByUsernameUseCase(repo)

	// Создаем пользователя с lowercase username
	existingUser, _ := domainuser.NewUser("external-123", "testuser", "test@example.com", "Test User")
	_ = repo.Save(context.Background(), existingUser)

	// Ищем с uppercase (должно не найтись, так как наш мок case-sensitive)
	query := user.GetUserByUsernameQuery{
		Username: "TestUser",
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	// В простом мок-репозитории поиск case-sensitive
	if !errors.Is(err, user.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound for different case, got: %v", err)
	}
}

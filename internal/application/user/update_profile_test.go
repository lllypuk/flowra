package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/flowra/flowra/internal/application/user"
	domainuser "github.com/flowra/flowra/internal/domain/user"
)

func TestUpdateProfileUseCase_Execute_Success_DisplayName(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewUpdateProfileUseCase(repo)

	// Создаем пользователя
	existingUser, _ := domainuser.NewUser("external-123", "testuser", "test@example.com", "Old Name")
	_ = repo.Save(context.Background(), existingUser)

	newDisplayName := "New Display Name"
	cmd := user.UpdateProfileCommand{
		UserID:      existingUser.ID(),
		DisplayName: &newDisplayName,
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Value.DisplayName() != newDisplayName {
		t.Errorf("expected displayName %s, got %s", newDisplayName, result.Value.DisplayName())
	}
}

func TestUpdateProfileUseCase_Execute_Success_Email(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewUpdateProfileUseCase(repo)

	// Создаем пользователя
	existingUser, _ := domainuser.NewUser("external-123", "testuser", "old@example.com", "Test User")
	_ = repo.Save(context.Background(), existingUser)

	newEmail := "new@example.com"
	cmd := user.UpdateProfileCommand{
		UserID: existingUser.ID(),
		Email:  &newEmail,
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Value.Email() != newEmail {
		t.Errorf("expected email %s, got %s", newEmail, result.Value.Email())
	}
}

func TestUpdateProfileUseCase_Execute_Success_Both(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewUpdateProfileUseCase(repo)

	// Создаем пользователя
	existingUser, _ := domainuser.NewUser("external-123", "testuser", "old@example.com", "Old Name")
	_ = repo.Save(context.Background(), existingUser)

	newDisplayName := "New Name"
	newEmail := "new@example.com"
	cmd := user.UpdateProfileCommand{
		UserID:      existingUser.ID(),
		DisplayName: &newDisplayName,
		Email:       &newEmail,
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Value.DisplayName() != newDisplayName {
		t.Errorf("expected displayName %s, got %s", newDisplayName, result.Value.DisplayName())
	}

	if result.Value.Email() != newEmail {
		t.Errorf("expected email %s, got %s", newEmail, result.Value.Email())
	}
}

func TestUpdateProfileUseCase_Execute_UserNotFound(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewUpdateProfileUseCase(repo)

	nonExistentUser, _ := domainuser.NewUser("external-123", "test", "test@example.com", "Test")
	newDisplayName := "New Name"
	cmd := user.UpdateProfileCommand{
		UserID:      nonExistentUser.ID(),
		DisplayName: &newDisplayName,
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if !errors.Is(err, user.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got: %v", err)
	}
}

func TestUpdateProfileUseCase_Execute_EmailAlreadyExists(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewUpdateProfileUseCase(repo)

	// Создаем двух пользователей
	user1, _ := domainuser.NewUser("external-1", "user1", "user1@example.com", "User 1")
	user2, _ := domainuser.NewUser("external-2", "user2", "user2@example.com", "User 2")
	_ = repo.Save(context.Background(), user1)
	_ = repo.Save(context.Background(), user2)

	// Пытаемся изменить email user2 на email user1
	existingEmail := user1.Email()
	cmd := user.UpdateProfileCommand{
		UserID: user2.ID(),
		Email:  &existingEmail,
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if !errors.Is(err, user.ErrEmailAlreadyExists) {
		t.Errorf("expected ErrEmailAlreadyExists, got: %v", err)
	}
}

func TestUpdateProfileUseCase_Validate_NoFieldsProvided(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewUpdateProfileUseCase(repo)

	existingUser, _ := domainuser.NewUser("external-123", "testuser", "test@example.com", "Test User")
	_ = repo.Save(context.Background(), existingUser)

	cmd := user.UpdateProfileCommand{
		UserID: existingUser.ID(),
		// Ни displayName, ни email не указаны
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for no fields provided")
	}
}

func TestUpdateProfileUseCase_Validate_InvalidEmail(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewUpdateProfileUseCase(repo)

	existingUser, _ := domainuser.NewUser("external-123", "testuser", "test@example.com", "Test User")
	_ = repo.Save(context.Background(), existingUser)

	invalidEmail := "invalid-email"
	cmd := user.UpdateProfileCommand{
		UserID: existingUser.ID(),
		Email:  &invalidEmail,
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid email")
	}
}

func TestUpdateProfileUseCase_Validate_EmptyDisplayName(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewUpdateProfileUseCase(repo)

	existingUser, _ := domainuser.NewUser("external-123", "testuser", "test@example.com", "Test User")
	_ = repo.Save(context.Background(), existingUser)

	emptyDisplayName := ""
	cmd := user.UpdateProfileCommand{
		UserID:      existingUser.ID(),
		DisplayName: &emptyDisplayName,
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for empty displayName")
	}
}

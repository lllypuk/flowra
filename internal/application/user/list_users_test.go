package user_test

import (
	"context"
	"testing"

	"github.com/flowra/flowra/internal/application/user"
	domainuser "github.com/flowra/flowra/internal/domain/user"
)

func TestListUsersUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewListUsersUseCase(repo)

	// Создаем несколько пользователей
	user1, _ := domainuser.NewUser("external-1", "user1", "user1@example.com", "User 1")
	user2, _ := domainuser.NewUser("external-2", "user2", "user2@example.com", "User 2")
	user3, _ := domainuser.NewUser("external-3", "user3", "user3@example.com", "User 3")
	_ = repo.Save(context.Background(), user1)
	_ = repo.Save(context.Background(), user2)
	_ = repo.Save(context.Background(), user3)

	query := user.ListUsersQuery{
		Offset: 0,
		Limit:  10,
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result.Users) != 3 {
		t.Errorf("expected 3 users, got %d", len(result.Users))
	}

	if result.TotalCount != 3 {
		t.Errorf("expected totalCount 3, got %d", result.TotalCount)
	}

	if result.Offset != 0 {
		t.Errorf("expected offset 0, got %d", result.Offset)
	}

	if result.Limit != 10 {
		t.Errorf("expected limit 10, got %d", result.Limit)
	}
}

func TestListUsersUseCase_Execute_Pagination(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewListUsersUseCase(repo)

	// Создаем 5 пользователей
	for i := 1; i <= 5; i++ {
		usr, _ := domainuser.NewUser(
			"external-"+string(rune('0'+i)),
			"user"+string(rune('0'+i)),
			"user"+string(rune('0'+i))+"@example.com",
			"User "+string(rune('0'+i)),
		)
		_ = repo.Save(context.Background(), usr)
	}

	// Первая страница
	query1 := user.ListUsersQuery{
		Offset: 0,
		Limit:  2,
	}

	result1, err := useCase.Execute(context.Background(), query1)
	if err != nil {
		t.Fatalf("expected no error for first page, got: %v", err)
	}

	if len(result1.Users) != 2 {
		t.Errorf("expected 2 users on first page, got %d", len(result1.Users))
	}

	if result1.TotalCount != 5 {
		t.Errorf("expected totalCount 5, got %d", result1.TotalCount)
	}

	// Вторая страница
	query2 := user.ListUsersQuery{
		Offset: 2,
		Limit:  2,
	}

	result2, err := useCase.Execute(context.Background(), query2)
	if err != nil {
		t.Fatalf("expected no error for second page, got: %v", err)
	}

	if len(result2.Users) != 2 {
		t.Errorf("expected 2 users on second page, got %d", len(result2.Users))
	}
}

func TestListUsersUseCase_Execute_EmptyList(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewListUsersUseCase(repo)

	query := user.ListUsersQuery{
		Offset: 0,
		Limit:  10,
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result.Users) != 0 {
		t.Errorf("expected 0 users, got %d", len(result.Users))
	}

	if result.TotalCount != 0 {
		t.Errorf("expected totalCount 0, got %d", result.TotalCount)
	}
}

func TestListUsersUseCase_Validate_NegativeOffset(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewListUsersUseCase(repo)

	query := user.ListUsersQuery{
		Offset: -1,
		Limit:  10,
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for negative offset")
	}
}

func TestListUsersUseCase_Validate_ZeroLimit(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewListUsersUseCase(repo)

	query := user.ListUsersQuery{
		Offset: 0,
		Limit:  0,
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for zero limit")
	}
}

func TestListUsersUseCase_Validate_LimitTooLarge(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewListUsersUseCase(repo)

	query := user.ListUsersQuery{
		Offset: 0,
		Limit:  101, // > 100
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for limit > 100")
	}
}

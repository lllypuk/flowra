package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lllypuk/flowra/internal/application/user"
	domainuser "github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

func TestPromoteToAdminUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewPromoteToAdminUseCase(repo)

	// Creating administrator
	admin, _ := domainuser.NewUser("external-admin", "admin", "admin@example.com", "Admin User")
	admin.SetAdmin(true)
	_ = repo.Save(context.Background(), admin)

	// Creating regular user
	regularUser, _ := domainuser.NewUser("external-user", "user", "user@example.com", "Regular User")
	_ = repo.Save(context.Background(), regularUser)

	cmd := user.PromoteToAdminCommand{
		UserID:     regularUser.ID(),
		PromotedBy: admin.ID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !result.Value.IsSystemAdmin() {
		t.Error("expected user to be promoted to admin")
	}
}

func TestPromoteToAdminUseCase_Execute_PromoterNotAdmin(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewPromoteToAdminUseCase(repo)

	// Creating dvuh obychnyh users
	user1, _ := domainuser.NewUser("external-1", "user1", "user1@example.com", "User 1")
	user2, _ := domainuser.NewUser("external-2", "user2", "user2@example.com", "User 2")
	_ = repo.Save(context.Background(), user1)
	_ = repo.Save(context.Background(), user2)

	cmd := user.PromoteToAdminCommand{
		UserID:     user2.ID(),
		PromotedBy: user1.ID(), // user1 not administrator
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if !errors.Is(err, user.ErrNotSystemAdmin) {
		t.Errorf("expected ErrNotSystemAdmin, got: %v", err)
	}
}

func TestPromoteToAdminUseCase_Execute_PromoterNotFound(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewPromoteToAdminUseCase(repo)

	// Creating regular user
	regularUser, _ := domainuser.NewUser("external-user", "user", "user@example.com", "Regular User")
	_ = repo.Save(context.Background(), regularUser)

	nonExistentUser, _ := domainuser.NewUser("external-fake", "fake", "fake@example.com", "Fake")

	cmd := user.PromoteToAdminCommand{
		UserID:     regularUser.ID(),
		PromotedBy: nonExistentUser.ID(), // not suschestvuet
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if !errors.Is(err, user.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got: %v", err)
	}
}

func TestPromoteToAdminUseCase_Execute_TargetUserNotFound(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewPromoteToAdminUseCase(repo)

	// Creating administrator
	admin, _ := domainuser.NewUser("external-admin", "admin", "admin@example.com", "Admin User")
	admin.SetAdmin(true)
	_ = repo.Save(context.Background(), admin)

	nonExistentUser, _ := domainuser.NewUser("external-fake", "fake", "fake@example.com", "Fake")

	cmd := user.PromoteToAdminCommand{
		UserID:     nonExistentUser.ID(), // not suschestvuet
		PromotedBy: admin.ID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if !errors.Is(err, user.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got: %v", err)
	}
}

func TestPromoteToAdminUseCase_Execute_AlreadyAdmin(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewPromoteToAdminUseCase(repo)

	// Creating administrator
	admin1, _ := domainuser.NewUser("external-admin1", "admin1", "admin1@example.com", "Admin 1")
	admin1.SetAdmin(true)
	_ = repo.Save(context.Background(), admin1)

	// Creating esche odnogo administrator
	admin2, _ := domainuser.NewUser("external-admin2", "admin2", "admin2@example.com", "Admin 2")
	admin2.SetAdmin(true)
	_ = repo.Save(context.Background(), admin2)

	cmd := user.PromoteToAdminCommand{
		UserID:     admin2.ID(),
		PromotedBy: admin1.ID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// dolzhno work normalno, dazhe if uzhe administrator
	if !result.Value.IsSystemAdmin() {
		t.Error("expected user to remain admin")
	}
}

func TestPromoteToAdminUseCase_Validate_InvalidUserID(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewPromoteToAdminUseCase(repo)

	admin, _ := domainuser.NewUser("external-admin", "admin", "admin@example.com", "Admin")
	admin.SetAdmin(true)
	_ = repo.Save(context.Background(), admin)

	cmd := user.PromoteToAdminCommand{
		UserID:     uuid.UUID(""), // invalid (zero UUID)
		PromotedBy: admin.ID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid userID")
	}
}

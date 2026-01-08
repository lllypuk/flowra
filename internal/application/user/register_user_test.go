package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lllypuk/flowra/internal/application/user"
	domainuser "github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// mockUserRepository - мок репозитория for testing
type mockUserRepository struct {
	users               map[string]*domainuser.User    // username -> user
	usersByEmail        map[string]*domainuser.User    // email -> user
	usersByExternalID   map[string]*domainuser.User    // keycloakID -> user
	usersByID           map[uuid.UUID]*domainuser.User // id -> user
	saveError           error
	findByUsernameError error
	findByEmailError    error
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users:             make(map[string]*domainuser.User),
		usersByEmail:      make(map[string]*domainuser.User),
		usersByExternalID: make(map[string]*domainuser.User),
		usersByID:         make(map[uuid.UUID]*domainuser.User),
	}
}

func (m *mockUserRepository) FindByID(_ context.Context, id uuid.UUID) (*domainuser.User, error) {
	if usr, ok := m.usersByID[id]; ok {
		return usr, nil
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepository) FindByExternalID(_ context.Context, keycloakID string) (*domainuser.User, error) {
	if usr, ok := m.usersByExternalID[keycloakID]; ok {
		return usr, nil
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepository) FindByUsername(_ context.Context, username string) (*domainuser.User, error) {
	if m.findByUsernameError != nil {
		return nil, m.findByUsernameError
	}
	if usr, ok := m.users[username]; ok {
		return usr, nil
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepository) FindByEmail(_ context.Context, email string) (*domainuser.User, error) {
	if m.findByEmailError != nil {
		return nil, m.findByEmailError
	}
	if usr, ok := m.usersByEmail[email]; ok {
		return usr, nil
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepository) Save(_ context.Context, usr *domainuser.User) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.users[usr.Username()] = usr
	m.usersByEmail[usr.Email()] = usr
	m.usersByExternalID[usr.ExternalID()] = usr
	m.usersByID[usr.ID()] = usr
	return nil
}

func (m *mockUserRepository) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockUserRepository) List(_ context.Context, offset, limit int) ([]*domainuser.User, error) {
	var allUsers []*domainuser.User
	for _, usr := range m.users {
		allUsers = append(allUsers, usr)
	}

	// Применяем offset
	if offset >= len(allUsers) {
		return []*domainuser.User{}, nil
	}

	// Применяем limit
	end := min(offset+limit, len(allUsers))

	return allUsers[offset:end], nil
}

func (m *mockUserRepository) Count(_ context.Context) (int, error) {
	return len(m.users), nil
}

func TestRegisterUserUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewRegisterUserUseCase(repo)
	cmd := user.RegisterUserCommand{
		ExternalID:  "external-123",
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "Test User",
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Value == nil {
		t.Fatal("expected user to be created")
	}

	if result.Value.Username() != cmd.Username {
		t.Errorf("expected username %s, got %s", cmd.Username, result.Value.Username())
	}

	if result.Value.Email() != cmd.Email {
		t.Errorf("expected email %s, got %s", cmd.Email, result.Value.Email())
	}

	if result.Value.ExternalID() != cmd.ExternalID {
		t.Errorf("expected keycloakID %s, got %s", cmd.ExternalID, result.Value.ExternalID())
	}

	// check, that userель savен
	if len(repo.users) != 1 {
		t.Errorf("expected 1 user in repository, got %d", len(repo.users))
	}
}

func TestRegisterUserUseCase_Execute_UsernameAlreadyExists(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewRegisterUserUseCase(repo)

	// Сначала creating user
	existingUser, _ := domainuser.NewUser("external-existing", "testuser", "existing@example.com", "Existing User")
	_ = repo.Save(context.Background(), existingUser)

	// Пытаемся create с тем же username
	cmd := user.RegisterUserCommand{
		ExternalID:  "external-New",
		Username:    "testuser",
		Email:       "New@example.com",
		DisplayName: "New User",
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if !errors.Is(err, user.ErrUsernameAlreadyExists) {
		t.Errorf("expected ErrUsernameAlreadyExists, got: %v", err)
	}
}

func TestRegisterUserUseCase_Execute_EmailAlreadyExists(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewRegisterUserUseCase(repo)

	// Сначала creating user
	existingUser, _ := domainuser.NewUser("external-existing", "existinguser", "test@example.com", "Existing User")
	_ = repo.Save(context.Background(), existingUser)

	// Пытаемся create с тем же email
	cmd := user.RegisterUserCommand{
		ExternalID:  "external-New",
		Username:    "newuser",
		Email:       "test@example.com",
		DisplayName: "New User",
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if !errors.Is(err, user.ErrEmailAlreadyExists) {
		t.Errorf("expected ErrEmailAlreadyExists, got: %v", err)
	}
}

func TestRegisterUserUseCase_Validate_MissingExternalID(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewRegisterUserUseCase(repo)
	cmd := user.RegisterUserCommand{
		ExternalID:  "",
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "Test User",
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing keycloakID")
	}
}

func TestRegisterUserUseCase_Validate_InvalidEmail(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	useCase := user.NewRegisterUserUseCase(repo)
	cmd := user.RegisterUserCommand{
		ExternalID:  "external-123",
		Username:    "testuser",
		Email:       "invalid-email",
		DisplayName: "Test User",
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid email")
	}
}

func TestRegisterUserUseCase_Execute_SaveError(t *testing.T) {
	// Arrange
	repo := newMockUserRepository()
	repo.saveError = errors.New("database error")
	useCase := user.NewRegisterUserUseCase(repo)
	cmd := user.RegisterUserCommand{
		ExternalID:  "external-123",
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "Test User",
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error from save operation")
	}
}

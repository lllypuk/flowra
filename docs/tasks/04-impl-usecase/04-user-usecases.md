# Task 04: User Domain Use Cases

**Дата:** 2025-10-19
**Статус:** 📝 Pending
**Зависимости:** Task 01 (Architecture)
**Оценка:** 3-4 часа

## Цель

Реализовать Use Cases для User entity. User domain относительно простой, так как основная аутентификация происходит через Keycloak.

## Контекст

**User entity:**
- ID, Username, Email, DisplayName
- System Admin flag
- Keycloak integration (паролей нет в домене)
- Простая CRUD модель (без Event Sourcing)

## Use Cases для реализации

### Command Use Cases

| UseCase | Операция | Приоритет | Оценка |
|---------|----------|-----------|--------|
| RegisterUserUseCase | Регистрация (синхронизация с Keycloak) | Критичный | 1.5 ч |
| UpdateProfileUseCase | Обновление профиля | Высокий | 1 ч |
| PromoteToAdminUseCase | Повышение до admin | Средний | 0.5 ч |

### Query Use Cases

| UseCase | Операция | Приоритет | Оценка |
|---------|----------|-----------|--------|
| GetUserUseCase | Получение по ID | Критичный | 0.5 ч |
| GetUserByUsernameUseCase | Поиск по username | Высокий | 0.5 ч |
| ListUsersUseCase | Список пользователей | Средний | 1 ч |

## Структура файлов

```
internal/application/user/
├── commands.go
├── queries.go
├── results.go
├── errors.go
│
├── register_user.go
├── update_profile.go
├── promote_to_admin.go
│
├── get_user.go
├── get_user_by_username.go
├── list_users.go
│
└── *_test.go
```

## Commands

```go
package user

import "github.com/google/uuid"

// RegisterUserCommand - регистрация пользователя
type RegisterUserCommand struct {
    KeycloakID  string        // ID из Keycloak
    Username    string
    Email       string
    DisplayName string
}

func (c RegisterUserCommand) CommandName() string { return "RegisterUser" }

// UpdateProfileCommand - обновление профиля
type UpdateProfileCommand struct {
    UserID      uuid.UUID
    DisplayName *string       // опционально
    Email       *string       // опционально
}

func (c UpdateProfileCommand) CommandName() string { return "UpdateProfile" }

// PromoteToAdminCommand - повышение до admin
type PromoteToAdminCommand struct {
    UserID     uuid.UUID
    PromotedBy uuid.UUID     // должен быть system admin
}

func (c PromoteToAdminCommand) CommandName() string { return "PromoteToAdmin" }
```

## RegisterUserUseCase (пример)

```go
package user

import (
    "context"
    "fmt"

    "github.com/lllypuk/teams-up/internal/application/shared"
    "github.com/lllypuk/teams-up/internal/domain/user"
)

type RegisterUserUseCase struct {
    userRepo user.Repository
}

func NewRegisterUserUseCase(userRepo user.Repository) *RegisterUserUseCase {
    return &RegisterUserUseCase{userRepo: userRepo}
}

func (uc *RegisterUserUseCase) Execute(
    ctx context.Context,
    cmd RegisterUserCommand,
) (UserResult, error) {
    // Валидация
    if err := uc.validate(cmd); err != nil {
        return UserResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // Проверка уникальности username
    existing, _ := uc.userRepo.FindByUsername(ctx, cmd.Username)
    if existing != nil {
        return UserResult{}, ErrUsernameAlreadyExists
    }

    // Создание пользователя
    usr := user.NewUser(
        cmd.KeycloakID,
        cmd.Username,
        cmd.Email,
        cmd.DisplayName,
    )

    // Сохранение
    if err := uc.userRepo.Save(ctx, usr); err != nil {
        return UserResult{}, fmt.Errorf("failed to save user: %w", err)
    }

    return UserResult{
        Result: shared.Result[*user.User]{
            Value: usr,
        },
    }, nil
}

func (uc *RegisterUserUseCase) validate(cmd RegisterUserCommand) error {
    if err := shared.ValidateRequired("keycloakID", cmd.KeycloakID); err != nil {
        return err
    }
    if err := shared.ValidateRequired("username", cmd.Username); err != nil {
        return err
    }
    if err := shared.ValidateRequired("email", cmd.Email); err != nil {
        return err
    }
    // TODO: email format validation
    return nil
}
```

## Keycloak Integration

RegisterUserUseCase вызывается при первом логине через Keycloak:

```go
// В auth middleware
func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // 1. Валидация JWT от Keycloak
        claims := validateToken(c)

        // 2. Получение или создание пользователя
        user, err := m.userRepo.FindByKeycloakID(c.Request().Context(), claims.Subject)
        if err != nil {
            // Пользователь не существует - регистрируем
            cmd := RegisterUserCommand{
                KeycloakID:  claims.Subject,
                Username:    claims.PreferredUsername,
                Email:       claims.Email,
                DisplayName: claims.Name,
            }
            result, err := m.registerUserUseCase.Execute(c.Request().Context(), cmd)
            if err != nil {
                return echo.ErrInternalServerError
            }
            user = result.Value
        }

        // 3. Добавление в контекст
        ctx := shared.WithUserID(c.Request().Context(), user.ID())
        c.SetRequest(c.Request().WithContext(ctx))

        return next(c)
    }
}
```

## Checklist

- [ ] Создать `commands.go`, `queries.go`, `results.go`, `errors.go`
- [ ] RegisterUserUseCase + tests
- [ ] UpdateProfileUseCase + tests
- [ ] PromoteToAdminUseCase + tests
- [ ] GetUserUseCase + tests
- [ ] GetUserByUsernameUseCase + tests
- [ ] ListUsersUseCase + tests
- [ ] Integration с Keycloak middleware

## Следующие шаги

- **Task 05**: Workspace UseCases

# Стандарты кодирования New Teams Up

## Обзор

Этот документ определяет стандарты кодирования для проекта New Teams Up. Соблюдение этих стандартов обеспечивает консистентность, читаемость и поддерживаемость кода.

## Общие принципы

### Clean Code
- Код должен быть самодокументируемым
- Предпочитайте ясность производительности (если это не критично)
- Используйте говорящие имена для переменных, функций и типов
- Функции должны делать одну вещь хорошо
- Избегайте глубокой вложенности (max 3 уровня)

### SOLID принципы
- **Single Responsibility Principle**: Класс/функция имеет одну причину для изменения
- **Open/Closed Principle**: Открыт для расширения, закрыт для модификации
- **Liskov Substitution Principle**: Подтипы должны быть заменяемы своими базовыми типами
- **Interface Segregation Principle**: Клиенты не должны зависеть от неиспользуемых интерфейсов
- **Dependency Inversion Principle**: Зависимости должны быть от абстракций, не от конкретных реализаций

## Go Стандарты

### Общие правила

#### Форматирование
- Используйте `gofmt` или `goimports` для автоматического форматирования
- Длина строки не более 120 символов
- Используйте табы для отступов
- Пустая строка между логическими блоками кода

```go
// Хорошо
func ProcessUser(ctx context.Context, userID int) (*User, error) {
    user, err := userRepo.FindByID(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("finding user: %w", err)
    }

    if user.IsActive {
        return user, nil
    }

    return nil, ErrUserInactive
}

// Плохо
func ProcessUser(ctx context.Context,userID int)(*User,error){
user,err:=userRepo.FindByID(ctx,userID)
if err!=nil{return nil,fmt.Errorf("finding user: %w",err)}
if user.IsActive{return user,nil}
return nil,ErrUserInactive
}
```

#### Именование

**Переменные и функции**: camelCase
```go
var userCount int
var isUserActive bool

func getUserProfile() {}
func calculateTotalScore() {}
```

**Константы**: UPPER_CASE или PascalCase для экспортируемых
```go
const (
    MaxRetryCount = 3
    defaultTimeout = 30 * time.Second
)
```

**Типы**: PascalCase
```go
type UserService struct {}
type HTTPClient interface {}
```

**Пакеты**: короткие, строчные, без underscore
```go
package auth
package userservice
```

#### Комментарии

**Публичные элементы**: обязательны комментарии
```go
// UserService handles user-related operations.
type UserService struct {
    repo UserRepository
}

// CreateUser creates a new user in the system.
// It validates the input and returns the created user or an error.
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
    // implementation
}
```

**Приватные элементы**: комментарии желательны для сложной логики
```go
// validateUserInput checks if the user input is valid
func validateUserInput(req CreateUserRequest) error {
    // validation logic
}
```

#### Обработка ошибок

**Wrap ошибки с контекстом**:
```go
user, err := repo.FindByID(ctx, id)
if err != nil {
    return nil, fmt.Errorf("finding user by ID %d: %w", id, err)
}
```

**Создание кастомных ошибок**:
```go
var (
    ErrUserNotFound = errors.New("user not found")
    ErrInvalidInput = errors.New("invalid input")
)

type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field %s: %s", e.Field, e.Message)
}
```

**Не игнорируйте ошибки**:
```go
// Хорошо
if err := someOperation(); err != nil {
    log.Error("operation failed", "error", err)
    return err
}

// Плохо
someOperation() // ошибка игнорируется
```

#### Работа с контекстом

**Первый параметр функции должен быть context.Context**:
```go
func ProcessRequest(ctx context.Context, req Request) (*Response, error) {
    // implementation
}
```

**Передавайте контекст во все внешние вызовы**:
```go
func (s *Service) ProcessUser(ctx context.Context, userID int) error {
    user, err := s.repo.GetUser(ctx, userID) // передаем контекст
    if err != nil {
        return err
    }
    
    return s.notifier.Send(ctx, user) // и здесь тоже
}
```

#### Интерфейсы

**Определяйте интерфейсы на стороне потребителя**:
```go
// В пакете service
type UserRepository interface {
    GetUser(ctx context.Context, id int) (*User, error)
    SaveUser(ctx context.Context, user *User) error
}

type UserService struct {
    repo UserRepository // зависимость от интерфейса
}
```

**Интерфейсы должны быть небольшими**:
```go
// Хорошо
type Reader interface {
    Read([]byte) (int, error)
}

type Writer interface {
    Write([]byte) (int, error)
}

// Плохо - слишком много методов
type FileManager interface {
    Read([]byte) (int, error)
    Write([]byte) (int, error)
    Close() error
    Seek(int64, int) (int64, error)
    Stat() (FileInfo, error)
    // ... еще 10 методов
}
```

### Структуры и методы

#### Структуры
```go
// Хорошо - группировка связанных полей
type User struct {
    // Identity
    ID    int    `json:"id" db:"id"`
    Email string `json:"email" db:"email"`
    
    // Profile
    FirstName string    `json:"first_name" db:"first_name"`
    LastName  string    `json:"last_name" db:"last_name"`
    Bio       *string   `json:"bio" db:"bio"` // nullable поля как указатели
    
    // Metadata
    CreatedAt time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
    DeletedAt *time.Time `json:"-" db:"deleted_at"`
}
```

#### Методы
```go
// Методы должны быть привязаны к соответствующему типу
func (u *User) IsActive() bool {
    return u.DeletedAt == nil
}

func (u *User) FullName() string {
    return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

// Конструкторы
func NewUser(email, firstName, lastName string) *User {
    return &User{
        Email:     email,
        FirstName: firstName,
        LastName:  lastName,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
}
```

### Пакеты и импорты

#### Организация импортов
```go
import (
    // Стандартная библиотека
    "context"
    "fmt"
    "time"
    
    // Внешние зависимости
    "github.com/gin-gonic/gin"
    "github.com/jmoiron/sqlx"
    
    // Внутренние пакеты
    "github.com/your-org/new-teams-up/internal/domain"
    "github.com/your-org/new-teams-up/pkg/logger"
)
```

#### Структура пакетов
```
internal/
├── domain/          # Доменные модели
├── application/     # Use cases
├── infrastructure/  # Внешние зависимости
└── presentation/    # HTTP handlers
```

### Тестирование

#### Именование тестов
```go
func TestUserService_CreateUser_Success(t *testing.T) {}
func TestUserService_CreateUser_InvalidEmail(t *testing.T) {}
func TestUserService_CreateUser_EmailAlreadyExists(t *testing.T) {}
```

#### Структура тестов (AAA pattern)
```go
func TestUserService_CreateUser_Success(t *testing.T) {
    // Arrange
    repo := &mockUserRepository{}
    service := NewUserService(repo)
    req := CreateUserRequest{
        Email:     "test@example.com",
        FirstName: "John",
        LastName:  "Doe",
    }
    
    // Act
    user, err := service.CreateUser(context.Background(), req)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, req.Email, user.Email)
    assert.Equal(t, req.FirstName, user.FirstName)
    assert.Equal(t, req.LastName, user.LastName)
}
```

#### Table-driven тесты для множественных сценариев
```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "user@example.com", false},
        {"empty email", "", true},
        {"invalid format", "not-an-email", true},
        {"missing domain", "user@", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateEmail(tt.email)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

## Database Standards

### Миграции

#### Именование файлов
```
001_create_users_table.up.sql
001_create_users_table.down.sql
002_add_user_profiles.up.sql
002_add_user_profiles.down.sql
```

#### Структура миграций
```sql
-- 001_create_users_table.up.sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
```

### Именование в БД

#### Таблицы: snake_case, множественное число
```sql
users
user_profiles
team_memberships
project_tasks
```

#### Колонки: snake_case
```sql
first_name
created_at
is_active
team_id
```

#### Индексы: idx_table_column(s)
```sql
idx_users_email
idx_team_memberships_user_id_team_id
```

#### Внешние ключи: fk_table_referenced_table
```sql
fk_user_profiles_users
fk_team_memberships_teams
```

## API Standards

### REST API

#### URL структура
```
GET    /api/v1/users           # Получить список пользователей
POST   /api/v1/users           # Создать пользователя
GET    /api/v1/users/{id}      # Получить пользователя
PUT    /api/v1/users/{id}      # Обновить пользователя
DELETE /api/v1/users/{id}      # Удалить пользователя

GET    /api/v1/users/{id}/teams # Получить команды пользователя
```

#### HTTP статус коды
```
200 OK                  # Успешный запрос
201 Created            # Ресурс создан
204 No Content         # Успешно, без содержимого
400 Bad Request        # Неверный запрос
401 Unauthorized       # Не авторизован
403 Forbidden          # Доступ запрещен
404 Not Found          # Ресурс не найден
409 Conflict           # Конфликт (например, email уже существует)
422 Unprocessable Entity # Ошибка валидации
500 Internal Server Error # Внутренняя ошибка сервера
```

#### JSON структуры

**Успешные ответы**:
```json
{
  "data": {
    "id": 1,
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe"
  },
  "meta": {
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

**Списки с пагинацией**:
```json
{
  "data": [...],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

**Ошибки**:
```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "Validation failed for user input",
    "details": [
      {
        "field": "email",
        "message": "Email is required"
      }
    ]
  },
  "meta": {
    "timestamp": "2024-01-15T10:30:00Z",
    "request_id": "req-123"
  }
}
```

## Безопасность

### Input Validation
```go
type CreateUserRequest struct {
    Email     string `json:"email" validate:"required,email"`
    FirstName string `json:"first_name" validate:"required,min=2,max=50"`
    LastName  string `json:"last_name" validate:"required,min=2,max=50"`
}

func (r CreateUserRequest) Validate() error {
    validator := validator.New()
    return validator.Struct(r)
}
```

### SQL Injection Prevention
```go
// Хорошо - используйте параметризованные запросы
func (r *repository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
    query := `SELECT id, email, first_name FROM users WHERE email = $1`
    var user User
    err := r.db.GetContext(ctx, &user, query, email)
    return &user, err
}

// Плохо - уязвимо к SQL injection
func (r *repository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
    query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)
    // ...
}
```

### Sensitive Data
```go
type User struct {
    ID           int    `json:"id"`
    Email        string `json:"email"`
    FirstName    string `json:"first_name"`
    PasswordHash string `json:"-"` // исключаем из JSON
}
```

## Логирование

### Структурированные логи
```go
import "go.uber.org/zap"

logger.Info("user created",
    zap.Int("user_id", user.ID),
    zap.String("email", user.Email),
    zap.Duration("duration", time.Since(start)),
)

logger.Error("failed to create user",
    zap.Error(err),
    zap.String("email", req.Email),
    zap.String("operation", "create_user"),
)
```

### Уровни логирования
- **DEBUG**: Детальная информация для отладки
- **INFO**: Общая информация о работе приложения
- **WARN**: Предупреждения о потенциальных проблемах
- **ERROR**: Ошибки, которые не останавливают приложение
- **FATAL**: Критические ошибки, останавливающие приложение

## Performance Guidelines

### Database
- Используйте индексы для часто запрашиваемых колонок
- Избегайте N+1 проблем (используйте JOIN или batch loading)
- Используйте connection pooling
- Используйте prepared statements

### Caching
```go
// Пример кэширования результатов
func (s *UserService) GetUser(ctx context.Context, id int) (*User, error) {
    // Попробуем получить из кэша
    if cached, found := s.cache.Get(fmt.Sprintf("user:%d", id)); found {
        return cached.(*User), nil
    }
    
    // Если нет в кэше, получаем из БД
    user, err := s.repo.GetUser(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // Сохраняем в кэш
    s.cache.Set(fmt.Sprintf("user:%d", id), user, 5*time.Minute)
    
    return user, nil
}
```

## Code Review Checklist

### Общее
- [ ] Код соответствует стандартам проекта
- [ ] Имена переменных и функций понятны
- [ ] Нет дублирования кода
- [ ] Функции не слишком большие (< 50 строк)
- [ ] Есть необходимые комментарии

### Go специфичное
- [ ] Используется `gofmt`
- [ ] Обрабатываются все ошибки
- [ ] Используется context.Context где нужно
- [ ] Нет race conditions
- [ ] Ресурсы освобождаются (defer close())

### Тесты
- [ ] Написаны тесты для нового кода
- [ ] Покрытие тестами достаточное (>80%)
- [ ] Тесты проходят успешно
- [ ] Тестируются error cases

### Безопасность
- [ ] Input validation реализована
- [ ] Нет hardcoded секретов
- [ ] SQL injection защита
- [ ] Правильная обработка sensitive data

## Инструменты

### Обязательные
- `gofmt` - форматирование кода
- `go vet` - статический анализ
- `golangci-lint` - линтер
- `go test` - тестирование

### Рекомендуемые
- `staticcheck` - расширенный статический анализ
- `govulncheck` - поиск уязвимостей
- `goimports` - управление импортами
- `godoc` - генерация документации

### Настройка .golangci.yml
```yaml
linters-settings:
  gocyclo:
    min-complexity: 15
  goconst:
    min-len: 3
    min-occurrences: 3
  misspell:
    locale: US

linters:
  enable:
    - gofmt
    - goimports
    - gosimple
    - govet
    - ineffassign
    - misspell
    - staticcheck
    - unused
```

---

*Последнее обновление: [Текущая дата]*  
*Версия: 1.0*  
*Поддерживается: Development Team*
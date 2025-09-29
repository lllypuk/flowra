# Development Guidelines для Claude

## Обзор

Этот документ содержит специфические руководства для AI-ассистента Claude при работе с проектом New Teams Up. Он дополняет основные стандарты кодирования и обеспечивает консистентный подход к разработке.

## Принципы работы с Claude

### Контекст-ориентированный подход
- Всегда учитывай архитектуру проекта (Clean Architecture + DDD)
- Понимай текущую фазу разработки (Phase 0: Foundation)
- Придерживайся установленных паттернов и конвенций
- Используй существующие абстракции и интерфейсы

### Качество кода
- Пиши код, готовый к production
- Следуй принципам SOLID
- Создавай тесты для нового кода
- Документируй публичные API

## Структурные паттерны

### Domain Layer
При создании доменных моделей:

```go
// Правильно - богатая доменная модель
type User struct {
    id       UserID
    email    Email
    profile  UserProfile
    skills   []Skill
    createdAt time.Time
    updatedAt time.Time
}

func (u *User) AddSkill(skill Skill) error {
    if u.HasSkill(skill.Type) {
        return ErrSkillAlreadyExists
    }
    u.skills = append(u.skills, skill)
    return nil
}

func (u *User) HasSkill(skillType SkillType) bool {
    for _, skill := range u.skills {
        if skill.Type == skillType {
            return true
        }
    }
    return false
}
```

### Application Layer
Use cases должны быть простыми и фокусироваться на оркестрации:

```go
type CreateUserUseCase struct {
    userRepo UserRepository
    eventBus EventBus
    logger   Logger
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, cmd CreateUserCommand) (*UserDTO, error) {
    // 1. Валидация входных данных
    if err := cmd.Validate(); err != nil {
        return nil, fmt.Errorf("validating command: %w", err)
    }

    // 2. Проверка бизнес-правил
    exists, err := uc.userRepo.ExistsByEmail(ctx, cmd.Email)
    if err != nil {
        return nil, fmt.Errorf("checking email existence: %w", err)
    }
    if exists {
        return nil, ErrEmailAlreadyExists
    }

    // 3. Создание доменного объекта
    user, err := NewUser(cmd.Email, cmd.Profile)
    if err != nil {
        return nil, fmt.Errorf("creating user: %w", err)
    }

    // 4. Сохранение
    if err := uc.userRepo.Save(ctx, user); err != nil {
        return nil, fmt.Errorf("saving user: %w", err)
    }

    // 5. Публикация события
    event := UserCreatedEvent{
        UserID:    user.ID(),
        Email:     user.Email().String(),
        CreatedAt: time.Now(),
    }
    uc.eventBus.Publish(ctx, event)

    // 6. Логирование
    uc.logger.Info("user created",
        zap.String("user_id", user.ID().String()),
        zap.String("email", user.Email().String()),
    )

    return ToUserDTO(user), nil
}
```

### Infrastructure Layer
Всегда реализуй интерфейсы из domain/application слоев:

```go
type PostgreSQLUserRepository struct {
    db     *sqlx.DB
    logger Logger
}

func NewPostgreSQLUserRepository(db *sqlx.DB, logger Logger) *PostgreSQLUserRepository {
    return &PostgreSQLUserRepository{
        db:     db,
        logger: logger,
    }
}

func (r *PostgreSQLUserRepository) Save(ctx context.Context, user *User) error {
    query := `
        INSERT INTO users (id, email, first_name, last_name, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (id) DO UPDATE SET
            email = EXCLUDED.email,
            first_name = EXCLUDED.first_name,
            last_name = EXCLUDED.last_name,
            updated_at = EXCLUDED.updated_at`

    _, err := r.db.ExecContext(ctx, query,
        user.ID(),
        user.Email().String(),
        user.Profile().FirstName,
        user.Profile().LastName,
        user.CreatedAt(),
        user.UpdatedAt(),
    )

    if err != nil {
        r.logger.Error("failed to save user",
            zap.String("user_id", user.ID().String()),
            zap.Error(err),
        )
        return fmt.Errorf("executing save query: %w", err)
    }

    return nil
}
```

## Naming Conventions

### Go-специфичные правила
- **Packages**: короткие, lowercase, без underscores (`user`, `team`, `match`)
- **Types**: PascalCase (`UserService`, `TeamRepository`)
- **Functions**: camelCase для private, PascalCase для public
- **Constants**: PascalCase или UPPER_CASE для package-level
- **Interfaces**: часто с суффиксом `-er` (`UserCreator`, `TeamManager`)

### Доменно-специфичные соглашения
- **Entities**: существительные (`User`, `Team`, `Project`)
- **Value Objects**: описательные (`Email`, `UserProfile`, `SkillLevel`)
- **Services**: `{Entity}Service` (`UserService`, `MatchService`)
- **Repositories**: `{Entity}Repository` (`UserRepository`)
- **Use Cases**: `{Verb}{Entity}UseCase` (`CreateUserUseCase`)
- **Events**: `{Entity}{Action}Event` (`UserCreatedEvent`, `TeamMemberAddedEvent`)

## Error Handling Patterns

### Доменные ошибки
```go
// Определяй ошибки как constants в domain слое
var (
    ErrUserNotFound        = errors.New("user not found")
    ErrEmailAlreadyExists  = errors.New("email already exists")
    ErrInvalidEmail       = errors.New("invalid email format")
    ErrTeamMemberLimit    = errors.New("team member limit exceeded")
)

// Для более сложных ошибок используй кастомные типы
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Code    string `json:"code"`
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}
```

### Обработка ошибок в слоях
```go
// Application layer - добавляет контекст
func (s *UserService) GetUser(ctx context.Context, id UserID) (*User, error) {
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, ErrUserNotFound) {
            return nil, err // передаем доменную ошибку как есть
        }
        return nil, fmt.Errorf("finding user %s: %w", id, err)
    }
    return user, nil
}

// Presentation layer - конвертирует в HTTP ответы
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    userID := UserID(mux.Vars(r)["id"])

    user, err := h.userService.GetUser(r.Context(), userID)
    if err != nil {
        if errors.Is(err, ErrUserNotFound) {
            h.respondWithError(w, http.StatusNotFound, "USER_NOT_FOUND", err.Error())
            return
        }
        h.respondWithError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
        return
    }

    h.respondWithData(w, http.StatusOK, ToUserDTO(user))
}
```

## Testing Guidelines

### Test Structure
Используй AAA pattern (Arrange, Act, Assert):

```go
func TestUserService_CreateUser_Success(t *testing.T) {
    // Arrange
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mocks.NewMockUserRepository(ctrl)
    mockEventBus := mocks.NewMockEventBus(ctrl)
    service := NewUserService(mockRepo, mockEventBus)

    cmd := CreateUserCommand{
        Email:     "test@example.com",
        FirstName: "John",
        LastName:  "Doe",
    }

    mockRepo.EXPECT().
        ExistsByEmail(gomock.Any(), Email("test@example.com")).
        Return(false, nil)

    mockRepo.EXPECT().
        Save(gomock.Any(), gomock.Any()).
        Return(nil)

    mockEventBus.EXPECT().
        Publish(gomock.Any(), gomock.Any()).
        Return(nil)

    // Act
    result, err := service.CreateUser(context.Background(), cmd)

    // Assert
    require.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "test@example.com", result.Email)
    assert.Equal(t, "John", result.FirstName)
    assert.Equal(t, "Doe", result.LastName)
}
```

### Test Data Builders
```go
type UserBuilder struct {
    user *User
}

func NewUserBuilder() *UserBuilder {
    user, _ := NewUser(
        Email("test@example.com"),
        UserProfile{
            FirstName: "Test",
            LastName:  "User",
        },
    )
    return &UserBuilder{user: user}
}

func (b *UserBuilder) WithEmail(email string) *UserBuilder {
    b.user.email = Email(email)
    return b
}

func (b *UserBuilder) WithName(firstName, lastName string) *UserBuilder {
    b.user.profile.FirstName = firstName
    b.user.profile.LastName = lastName
    return b
}

func (b *UserBuilder) Build() *User {
    return b.user
}

// Usage
user := NewUserBuilder().
    WithEmail("custom@example.com").
    WithName("Custom", "User").
    Build()
```

## API Design

### Request/Response DTOs
```go
// Request DTOs - всегда валидируются
type CreateUserRequest struct {
    Email     string `json:"email" validate:"required,email"`
    FirstName string `json:"first_name" validate:"required,min=2,max=50"`
    LastName  string `json:"last_name" validate:"required,min=2,max=50"`
}

func (r CreateUserRequest) Validate() error {
    validator := validator.New()
    return validator.Struct(r)
}

// Response DTOs - только данные для клиента
type UserResponse struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    FirstName string    `json:"first_name"`
    LastName  string    `json:"last_name"`
    CreatedAt time.Time `json:"created_at"`
}

// Mappers - отдельные функции для конвертации
func ToUserResponse(user *User) UserResponse {
    return UserResponse{
        ID:        user.ID().String(),
        Email:     user.Email().String(),
        FirstName: user.Profile().FirstName,
        LastName:  user.Profile().LastName,
        CreatedAt: user.CreatedAt(),
    }
}
```

### Handler Pattern
```go
type UserHandler struct {
    userService UserService
    logger      Logger
}

func NewUserHandler(userService UserService, logger Logger) *UserHandler {
    return &UserHandler{
        userService: userService,
        logger:      logger,
    }
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondWithError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON format")
        return
    }

    if err := req.Validate(); err != nil {
        h.respondWithValidationError(w, err)
        return
    }

    cmd := CreateUserCommand{
        Email:     req.Email,
        FirstName: req.FirstName,
        LastName:  req.LastName,
    }

    user, err := h.userService.CreateUser(r.Context(), cmd)
    if err != nil {
        h.handleServiceError(w, err)
        return
    }

    h.respondWithData(w, http.StatusCreated, ToUserResponse(user))
}
```

## Database Patterns

### Migration Structure
```sql
-- 001_create_users_table.up.sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_created_at ON users(created_at);
```

### Repository Query Patterns
```go
func (r *PostgreSQLUserRepository) FindByEmail(ctx context.Context, email Email) (*User, error) {
    query := `
        SELECT id, email, first_name, last_name, created_at, updated_at
        FROM users
        WHERE email = $1 AND deleted_at IS NULL`

    var row struct {
        ID        string    `db:"id"`
        Email     string    `db:"email"`
        FirstName string    `db:"first_name"`
        LastName  string    `db:"last_name"`
        CreatedAt time.Time `db:"created_at"`
        UpdatedAt time.Time `db:"updated_at"`
    }

    err := r.db.GetContext(ctx, &row, query, email.String())
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrUserNotFound
        }
        return nil, fmt.Errorf("querying user by email: %w", err)
    }

    return r.toDomainUser(row), nil
}
```

## Configuration Management

### Environment-based Config
```go
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Redis    RedisConfig    `mapstructure:"redis"`
    JWT      JWTConfig      `mapstructure:"jwt"`
    Logger   LoggerConfig   `mapstructure:"logger"`
}

type ServerConfig struct {
    Host         string        `mapstructure:"host" default:"localhost"`
    Port         int           `mapstructure:"port" default:"8080"`
    ReadTimeout  time.Duration `mapstructure:"read_timeout" default:"30s"`
    WriteTimeout time.Duration `mapstructure:"write_timeout" default:"30s"`
}

func LoadConfig() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("./configs")
    viper.AddConfigPath(".")

    viper.AutomaticEnv()
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

    if err := viper.ReadInConfig(); err != nil {
        return nil, fmt.Errorf("reading config: %w", err)
    }

    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("unmarshaling config: %w", err)
    }

    return &config, nil
}
```

## Logging Standards

### Structured Logging
```go
import "go.uber.org/zap"

// Application level
logger.Info("user created successfully",
    zap.String("user_id", user.ID().String()),
    zap.String("email", user.Email().String()),
    zap.Duration("duration", time.Since(start)),
)

// Error with context
logger.Error("failed to create user",
    zap.Error(err),
    zap.String("email", req.Email),
    zap.String("operation", "create_user"),
    zap.String("trace_id", traceID),
)

// Performance monitoring
logger.Warn("slow database query detected",
    zap.Duration("duration", queryTime),
    zap.String("query", "SELECT * FROM users"),
    zap.String("table", "users"),
)
```

## Performance Considerations

### Database Optimization
- Всегда используй индексы для часто запрашиваемых полей
- Избегай N+1 проблем (используй JOIN или batch loading)
- Используй prepared statements для repeated queries
- Реализуй pagination для больших datasets

### Caching Strategy
```go
func (s *UserService) GetUser(ctx context.Context, id UserID) (*User, error) {
    // Try cache first
    cacheKey := fmt.Sprintf("user:%s", id)
    if cached, found := s.cache.Get(cacheKey); found {
        return cached.(*User), nil
    }

    // Fallback to database
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // Cache for future requests
    s.cache.Set(cacheKey, user, 5*time.Minute)

    return user, nil
}
```

## Security Guidelines

### Input Validation
- Всегда валидируй входные данные на уровне presentation
- Используй whitelist подход для валидации
- Санитизируй данные перед сохранением
- Используй prepared statements для SQL queries

### Authentication & Authorization
```go
func (h *UserHandler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := extractTokenFromHeader(r.Header.Get("Authorization"))
        if token == "" {
            h.respondWithError(w, http.StatusUnauthorized, "MISSING_TOKEN", "Authorization token required")
            return
        }

        claims, err := h.jwtService.ValidateToken(token)
        if err != nil {
            h.respondWithError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid token")
            return
        }

        ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
        next(w, r.WithContext(ctx))
    }
}
```

## Documentation Standards

### Code Comments
```go
// UserService handles all user-related business operations.
// It coordinates between the domain layer and infrastructure layer
// while maintaining business rules and invariants.
type UserService struct {
    repo     UserRepository
    eventBus EventBus
    logger   Logger
}

// CreateUser creates a new user in the system.
// It validates the input, checks business rules, and publishes
// a UserCreatedEvent upon successful creation.
//
// Returns ErrEmailAlreadyExists if the email is already in use.
func (s *UserService) CreateUser(ctx context.Context, cmd CreateUserCommand) (*User, error) {
    // implementation
}
```

### API Documentation
Используй OpenAPI/Swagger спецификации для документирования API endpoints.

## Monitoring & Observability

### Metrics
```go
// Prometheus metrics
var (
    userCreationCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "users_created_total",
            Help: "Total number of users created",
        },
        []string{"status"},
    )

    userCreationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "user_creation_duration_seconds",
            Help: "Duration of user creation operations",
        },
        []string{"status"},
    )
)

func (s *UserService) CreateUser(ctx context.Context, cmd CreateUserCommand) (*User, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start).Seconds()
        userCreationDuration.WithLabelValues("success").Observe(duration)
        userCreationCounter.WithLabelValues("success").Inc()
    }()

    // implementation
}
```

---

*Следуй этим принципам для поддержания высокого качества кода и архитектурной консистентности.*

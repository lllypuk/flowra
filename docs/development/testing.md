# Стратегия тестирования New Teams Up

## Обзор

Этот документ описывает комплексную стратегию тестирования для проекта New Teams Up, включая типы тестов, инструменты, практики и процессы обеспечения качества.

## Принципы тестирования

### Пирамида тестирования
```
        E2E Tests
       /         \
    Integration Tests
   /                 \
  Unit Tests (Foundation)
```

- **70% Unit Tests** - Быстрые, изолированные, тестируют отдельные компоненты
- **20% Integration Tests** - Тестируют взаимодействие между компонентами
- **10% E2E Tests** - Тестируют пользовательские сценарии целиком

### Основные принципы

- **Test-Driven Development (TDD)** где это целесообразно
- **Fail Fast** - быстрое обнаружение проблем
- **Independent Tests** - тесты не должны зависеть друг от друга
- **Repeatable** - стабильные результаты в любой среде
- **Clear Test Names** - имя теста объясняет что тестируется

## Типы тестирования

### 1. Unit Tests

**Цель**: Тестирование отдельных функций, методов и компонентов в изоляции.

**Покрытие**: Минимум 80% для всего кода, 95% для критичной бизнес-логики.

**Пример структуры**:
```go
func TestUserService_CreateUser_Success(t *testing.T) {
    // Arrange
    mockRepo := &mocks.UserRepository{}
    mockEventBus := &mocks.EventBus{}
    service := NewUserService(mockRepo, mockEventBus)
    
    req := CreateUserRequest{
        Email:     "test@example.com",
        FirstName: "John",
        LastName:  "Doe",
    }
    
    expectedUser := &User{
        ID:        1,
        Email:     req.Email,
        FirstName: req.FirstName,
        LastName:  req.LastName,
    }
    
    mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*User")).Return(nil)
    mockEventBus.On("Publish", mock.AnythingOfType("UserCreatedEvent")).Return(nil)
    
    // Act
    result, err := service.CreateUser(context.Background(), req)
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, expectedUser.Email, result.Email)
    assert.Equal(t, expectedUser.FirstName, result.FirstName)
    assert.Equal(t, expectedUser.LastName, result.LastName)
    mockRepo.AssertExpectations(t)
    mockEventBus.AssertExpectations(t)
}
```

**Table-driven тесты для множественных сценариев**:
```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name      string
        email     string
        wantValid bool
        wantError string
    }{
        {
            name:      "valid_email",
            email:     "user@example.com",
            wantValid: true,
        },
        {
            name:      "empty_email",
            email:     "",
            wantValid: false,
            wantError: "email is required",
        },
        {
            name:      "invalid_format",
            email:     "not-an-email",
            wantValid: false,
            wantError: "invalid email format",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateEmail(tt.email)
            if tt.wantValid {
                assert.NoError(t, err)
            } else {
                assert.Error(t, err)
                if tt.wantError != "" {
                    assert.Contains(t, err.Error(), tt.wantError)
                }
            }
        })
    }
}
```

### 2. Integration Tests

**Цель**: Тестирование взаимодействия между компонентами системы.

**Теги для сборки**:
```go
//go:build integration

package integration

import (
    "testing"
    "github.com/stretchr/testify/suite"
)

type UserServiceIntegrationTestSuite struct {
    suite.Suite
    db      *sql.DB
    service *UserService
}

func (s *UserServiceIntegrationTestSuite) SetupSuite() {
    // Настройка тестовой БД
    s.db = setupTestDB()
    s.service = NewUserService(NewUserRepository(s.db))
}

func (s *UserServiceIntegrationTestSuite) TearDownSuite() {
    s.db.Close()
}

func (s *UserServiceIntegrationTestSuite) SetupTest() {
    // Очистка данных перед каждым тестом
    clearTestData(s.db)
}

func (s *UserServiceIntegrationTestSuite) TestCreateUser_DatabaseIntegration() {
    // Arrange
    req := CreateUserRequest{
        Email:     "integration@example.com",
        FirstName: "Integration",
        LastName:  "Test",
    }
    
    // Act
    user, err := s.service.CreateUser(context.Background(), req)
    
    // Assert
    s.Require().NoError(err)
    s.Equal(req.Email, user.Email)
    
    // Verify in database
    var count int
    err = s.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", req.Email).Scan(&count)
    s.Require().NoError(err)
    s.Equal(1, count)
}

func TestUserServiceIntegrationTestSuite(t *testing.T) {
    suite.Run(t, new(UserServiceIntegrationTestSuite))
}
```

### 3. API Tests

**Цель**: Тестирование HTTP API endpoints.

```go
func TestUserAPI_CreateUser_Success(t *testing.T) {
    // Arrange
    app := setupTestApp()
    defer app.Close()
    
    payload := `{
        "email": "api@example.com",
        "first_name": "API",
        "last_name": "Test"
    }`
    
    // Act
    resp, err := http.Post(
        app.URL+"/api/v1/users",
        "application/json",
        strings.NewReader(payload),
    )
    
    // Assert
    require.NoError(t, err)
    defer resp.Body.Close()
    
    assert.Equal(t, http.StatusCreated, resp.Status)
    
    var response struct {
        Data struct {
            ID        int    `json:"id"`
            Email     string `json:"email"`
            FirstName string `json:"first_name"`
            LastName  string `json:"last_name"`
        } `json:"data"`
    }
    
    err = json.NewDecoder(resp.Body).Decode(&response)
    require.NoError(t, err)
    
    assert.Equal(t, "api@example.com", response.Data.Email)
    assert.Equal(t, "API", response.Data.FirstName)
    assert.Equal(t, "Test", response.Data.LastName)
}
```

### 4. End-to-End Tests

**Цель**: Тестирование полных пользовательских сценариев.

**Инструменты**: Playwright, Selenium WebDriver, или API-based E2E тесты.

```go
func TestE2E_UserRegistrationFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }
    
    // Arrange
    app := setupFullTestApp()
    defer app.Close()
    
    client := &http.Client{Timeout: 30 * time.Second}
    
    // Act & Assert: Complete user registration flow
    
    // Step 1: Register user
    registerResp := registerUser(t, client, app.URL, "e2e@example.com", "password123")
    assert.Equal(t, http.StatusCreated, registerResp.StatusCode)
    
    // Step 2: Verify email (simulate)
    verifyEmail(t, client, app.URL, "verification-token")
    
    // Step 3: Login
    loginResp := loginUser(t, client, app.URL, "e2e@example.com", "password123")
    assert.Equal(t, http.StatusOK, loginResp.StatusCode)
    
    token := extractAuthToken(t, loginResp)
    
    // Step 4: Create team
    teamResp := createTeam(t, client, app.URL, token, "E2E Test Team")
    assert.Equal(t, http.StatusCreated, teamResp.StatusCode)
    
    // Step 5: Invite member
    inviteResp := inviteMember(t, client, app.URL, token, "member@example.com")
    assert.Equal(t, http.StatusOK, inviteResp.StatusCode)
}
```

### 5. Performance Tests

**Цель**: Проверка производительности критичных компонентов.

```go
func BenchmarkUserService_CreateUser(b *testing.B) {
    service := setupBenchmarkService()
    req := CreateUserRequest{
        Email:     "bench@example.com",
        FirstName: "Benchmark",
        LastName:  "Test",
    }
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        req.Email = fmt.Sprintf("bench%d@example.com", i)
        _, err := service.CreateUser(context.Background(), req)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkUserRepository_FindByID(b *testing.B) {
    repo := setupBenchmarkRepository()
    userID := createTestUser(repo)
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, err := repo.FindByID(context.Background(), userID)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### 6. Load Tests

**Инструменты**: k6, Apache JMeter

**Пример k6 скрипта**:
```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '2m', target: 100 }, // Ramp up to 100 users
    { duration: '5m', target: 100 }, // Stay at 100 users
    { duration: '2m', target: 200 }, // Ramp up to 200 users
    { duration: '5m', target: 200 }, // Stay at 200 users
    { duration: '2m', target: 0 },   // Ramp down to 0 users
  ],
};

export default function () {
  let response = http.post('http://localhost:8080/api/v1/users', JSON.stringify({
    email: `load-test-${Math.random()}@example.com`,
    first_name: 'Load',
    last_name: 'Test'
  }), {
    headers: { 'Content-Type': 'application/json' },
  });
  
  check(response, {
    'status is 201': (r) => r.status === 201,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  sleep(1);
}
```

## Инструменты и библиотеки

### Go Testing Libraries

```go
// go.mod dependencies
require (
    github.com/stretchr/testify v1.8.4
    github.com/golang/mock v1.6.0
    github.com/DATA-DOG/go-sqlmock v1.5.0
    github.com/testcontainers/testcontainers-go v0.23.0
)
```

### Основные библиотеки

**testify**: Основная библиотека для assertions
```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/suite"
)
```

**gomock**: Генерация моков
```bash
go install github.com/golang/mock/mockgen@latest
//go:generate mockgen -source=interfaces.go -destination=mocks/mock_interfaces.go
```

**testcontainers**: Интеграционные тесты с реальными зависимостями
```go
func setupTestDB(t *testing.T) *sql.DB {
    ctx := context.Background()
    
    postgresContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:14"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    require.NoError(t, err)
    
    connStr, err := postgresContainer.ConnectionString(ctx)
    require.NoError(t, err)
    
    db, err := sql.Open("postgres", connStr)
    require.NoError(t, err)
    
    return db
}
```

## Моки и тестовые дублеры

### Типы тестовых дублеров

1. **Dummy** - объекты, которые передаются, но не используются
2. **Fake** - рабочие реализации с упрощениями
3. **Stubs** - предоставляют готовые ответы на вызовы
4. **Spies** - записывают информацию о том, как они были вызваны
5. **Mocks** - предварительно запрограммированные с ожиданиями

### Создание моков

**Интерфейс для мокирования**:
```go
type UserRepository interface {
    Save(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id int) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
}
```

**Генерация мока**:
```bash
mockgen -source=repository.go -destination=mocks/mock_repository.go
```

**Использование мока в тестах**:
```go
func TestUserService_CreateUser_RepositoryError(t *testing.T) {
    // Arrange
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    mockRepo := mocks.NewMockUserRepository(ctrl)
    service := NewUserService(mockRepo)
    
    req := CreateUserRequest{Email: "test@example.com"}
    expectedError := errors.New("database connection failed")
    
    mockRepo.EXPECT().
        Save(gomock.Any(), gomock.Any()).
        Return(expectedError)
    
    // Act
    _, err := service.CreateUser(context.Background(), req)
    
    // Assert
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "database connection failed")
}
```

### In-Memory тестовые реализации

```go
type InMemoryUserRepository struct {
    users map[int]*User
    mutex sync.RWMutex
    nextID int
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
    return &InMemoryUserRepository{
        users: make(map[int]*User),
        nextID: 1,
    }
}

func (r *InMemoryUserRepository) Save(ctx context.Context, user *User) error {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    
    if user.ID == 0 {
        user.ID = r.nextID
        r.nextID++
    }
    
    r.users[user.ID] = user
    return nil
}

func (r *InMemoryUserRepository) FindByID(ctx context.Context, id int) (*User, error) {
    r.mutex.RLock()
    defer r.mutex.RUnlock()
    
    user, exists := r.users[id]
    if !exists {
        return nil, ErrUserNotFound
    }
    
    return user, nil
}
```

## Тестовые данные

### Фабрики тестовых данных

```go
package testdata

import (
    "time"
    "github.com/your-org/new-teams-up/internal/domain"
)

type UserFactory struct {
    counter int
}

func NewUserFactory() *UserFactory {
    return &UserFactory{counter: 0}
}

func (f *UserFactory) Create(opts ...UserOption) *domain.User {
    f.counter++
    
    user := &domain.User{
        ID:        f.counter,
        Email:     fmt.Sprintf("user%d@example.com", f.counter),
        FirstName: "Test",
        LastName:  "User",
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    
    for _, opt := range opts {
        opt(user)
    }
    
    return user
}

type UserOption func(*domain.User)

func WithEmail(email string) UserOption {
    return func(u *domain.User) {
        u.Email = email
    }
}

func WithName(firstName, lastName string) UserOption {
    return func(u *domain.User) {
        u.FirstName = firstName
        u.LastName = lastName
    }
}

// Использование
func TestSomething(t *testing.T) {
    factory := NewUserFactory()
    
    user1 := factory.Create()
    user2 := factory.Create(
        WithEmail("custom@example.com"),
        WithName("John", "Doe"),
    )
}
```

### Fixtures

```go
// testdata/fixtures.go
func LoadUserFixtures() []*domain.User {
    return []*domain.User{
        {
            ID:        1,
            Email:     "admin@example.com",
            FirstName: "Admin",
            LastName:  "User",
            CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
        },
        {
            ID:        2,
            Email:     "user@example.com",
            FirstName: "Regular",
            LastName:  "User",
            CreatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
        },
    }
}
```

## CI/CD интеграция

### GitHub Actions

```yaml
# .github/workflows/test.yml
name: Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:14
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: test_db
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432
      
      redis:
        image: redis:6
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 6379:6379
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-
    
    - name: Download dependencies
      run: go mod download
    
    - name: Run linter
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest
    
    - name: Run unit tests
      run: make test-unit
    
    - name: Run integration tests
      run: make test-integration
      env:
        DB_HOST: localhost
        DB_PORT: 5432
        DB_NAME: test_db
        DB_USER: postgres
        DB_PASSWORD: postgres
        REDIS_HOST: localhost
        REDIS_PORT: 6379
    
    - name: Generate coverage report
      run: make coverage
    
    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
        flags: unittests
        name: codecov-umbrella
```

### Test Reports

```yaml
    - name: Publish Test Results
      uses: dorny/test-reporter@v1
      if: success() || failure()
      with:
        name: Go Tests
        path: '*.xml'
        reporter: java-junit
```

## Makefile команды

```makefile
# Makefile
.PHONY: test test-unit test-integration test-e2e coverage lint

# Все тесты
test:
	go test -v ./...

# Unit тесты
test-unit:
	go test -v -short ./...

# Integration тесты
test-integration:
	go test -v -tags=integration ./...

# E2E тесты
test-e2e:
	go test -v -tags=e2e ./tests/e2e/...

# Покрытие кода
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

# Тесты с race detection
test-race:
	go test -race -v ./...

# Benchmarks
bench:
	go test -bench=. -benchmem ./...

# Линтер
lint:
	golangci-lint run

# Генерация моков
generate-mocks:
	go generate ./...

# Очистка
clean-test:
	rm -f coverage.out coverage.html
	go clean -testcache
```

## Метрики качества

### Покрытие кода

- **Unit tests**: минимум 80%
- **Critical business logic**: минимум 95%
- **Integration paths**: минимум 70%

### Performance метрики

- **API response time**: < 200ms для 95% запросов
- **Database queries**: < 100ms для простых запросов
- **Memory usage**: отсутствие memory leaks

### Качественные метрики

- **Flaky tests**: < 1%
- **Test execution time**: все тесты < 10 минут
- **Code duplication**: < 5%

## Best Practices

### Написание тестов

1. **Следуйте AAA паттерну**: Arrange, Act, Assert
2. **Один assert на тест**: каждый тест должен проверять одну вещь
3. **Независимые тесты**: тесты не должны зависеть друг от друга
4. **Описательные имена**: имя должно объяснять что тестируется
5. **Тестируйте поведение, не реализацию**: фокус на входы и выходы

### Структура тестов

```go
func TestServiceName_MethodName_Scenario(t *testing.T) {
    // Arrange - настройка данных и моков
    
    // Act - выполнение тестируемого действия
    
    // Assert - проверка результатов
}
```

### Организация тестовых файлов

```
internal/
├── service/
│   ├── user_service.go
│   ├── user_service_test.go      # Unit тесты
│   └── user_service_integration_test.go  # Integration тесты
├── repository/
│   ├── user_repository.go
│   └── user_repository_test.go
└── testdata/
    ├── fixtures/
    ├── factories/
    └── helpers/
```

## Troubleshooting

### Частые проблемы

**Flaky тесты**:
- Используйте детерминированные данные
- Избегайте sleep в пользу синхронизации
- Изолируйте внешние зависимости

**Медленные тесты**:
- Используйте in-memory databases для unit тестов
- Параллелизируйте тесты где возможно
- Оптимизируйте setup/teardown

**Race conditions**:
- Всегда запускайте тесты с `-race`
- Используйте proper synchronization
- Тестируйте concurrent scenarios

## Заключение

Качественная стратегия тестирования - это основа надежного и поддерживаемого кода. Следование описанным практикам поможет обеспечить высокое качество продукта и уверенность в изменениях.

### Ключевые принципы:
- Пишите тесты как часть разработки, не после
- Поддерживайте высокое покрытие кода тестами
- Регулярно рефакторите тесты вместе с продакшн кодом
- Автоматизируйте выполнение тестов в CI/CD
- Используйте метрики для мониторинга качества тестов

---

*Последнее обновление: [Текущая дата]*  
*Версия: 1.0*  
*Поддерживается: QA Team*
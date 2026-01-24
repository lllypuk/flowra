# Flowra Testing Strategy

## Overview

This document describes a comprehensive testing strategy for the Flowra project, including test types, tools, practices,
and quality assurance processes.

## Testing Principles

### Testing Pyramid

```
        E2E Tests
       /         \
    Integration Tests
   /                 \
  Unit Tests (Foundation)
```

- **70% Unit Tests** - Fast, isolated, testing individual components
- **20% Integration Tests** - Testing interactions between components
- **10% E2E Tests** - Testing complete user scenarios

### Core Principles

- **Test-Driven Development (TDD)** where appropriate
- **Fail Fast** - quick problem detection
- **Independent Tests** - tests should not depend on each other
- **Repeatable** - stable results in any environment
- **Clear Test Names** - test name explains what is being tested

## Testing Types

### 1. Unit Tests

**Goal**: Testing individual functions, methods, and components in isolation.

**Coverage**: Minimum 80% for all code, 95% for critical business logic.

**Example structure**:

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

**Table-driven tests for multiple scenarios**:

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
t.Run(tt.name, func (t *testing.T) {
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

**Goal**: Testing interactions between system components.

**Build tags**:

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
	// Setup test DB
	s.db = setupTestDB()
	s.service = NewUserService(NewUserRepository(s.db))
}

func (s *UserServiceIntegrationTestSuite) TearDownSuite() {
	s.db.Close()
}

func (s *UserServiceIntegrationTestSuite) SetupTest() {
	// Clear data before each test
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

**Goal**: Testing HTTP API endpoints.

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

**Goal**: Testing complete user scenarios.

**Tools**: Playwright, Selenium WebDriver, or API-based E2E tests.

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
registerResp := registerUser(t, client, app.URL, "e2e@example.com", "test123")
assert.Equal(t, http.StatusCreated, registerResp.StatusCode)

// Step 2: Verify email (simulate)
verifyEmail(t, client, app.URL, "verification-token")

// Step 3: Login
loginResp := loginUser(t, client, app.URL, "e2e@example.com", "test123")
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

**Goal**: Checking performance of critical components.

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

**Tools**: k6, Apache JMeter

**Example k6 script**:

```javascript
import http from 'k6/http';
import {check, sleep} from 'k6';

export let options = {
    stages: [
        {duration: '2m', target: 100}, // Ramp up to 100 users
        {duration: '5m', target: 100}, // Stay at 100 users
        {duration: '2m', target: 200}, // Ramp up to 200 users
        {duration: '5m', target: 200}, // Stay at 200 users
        {duration: '2m', target: 0},   // Ramp down to 0 users
    ],
};

export default function () {
    let response = http.post('http://localhost:8080/api/v1/users', JSON.stringify({
        email: `load-test-${Math.random()}@example.com`,
        first_name: 'Load',
        last_name: 'Test'
    }), {
        headers: {'Content-Type': 'application/json'},
    });

    check(response, {
        'status is 201': (r) => r.status === 201,
        'response time < 500ms': (r) => r.timings.duration < 500,
    });

    sleep(1);
}
```

## Tools and Libraries

### Go Testing Libraries

```go
// go.mod dependencies
require (
github.com/stretchr/testify v1.8.4
github.com/golang/mock v1.6.0
github.com/DATA-DOG/go -sqlmock v1.5.0
github.com/testcontainers/testcontainers-go v0.23.0
)
```

### Core Libraries

**testify**: Main library for assertions

```go
import (
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
"github.com/stretchr/testify/mock"
"github.com/stretchr/testify/suite"
)
```

**gomock**: Mock generation

```bash
go install github.com/golang/mock/mockgen@latest
//go:generate mockgen -source=interfaces.go -destination=mocks/mock_interfaces.go
```

**testcontainers**: Integration tests with real dependencies

```go
func setupTestDB(t *testing.T) *mongo.Client {
ctx := context.Background()

mongoContainer, err := mongodb.RunContainer(ctx,
testcontainers.WithImage("mongo:6.0"),
mongodb.WithUsername("admin"),
mongodb.WithPassword("admin123"),
)
require.NoError(t, err)

uri, err := mongoContainer.ConnectionString(ctx)
require.NoError(t, err)

client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
require.NoError(t, err)

return client
}
```

## Mocks and Test Doubles

### Types of Test Doubles

1. **Dummy** - objects that are passed but not used
2. **Fake** - working implementations with simplifications
3. **Stubs** - provide canned answers to calls
4. **Spies** - record information about how they were called
5. **Mocks** - pre-programmed with expectations

### Creating Mocks

**Interface for mocking**:

```go
type UserRepository interface {
Save(ctx context.Context, user *User) error
FindByID(ctx context.Context, id int) (*User, error)
FindByEmail(ctx context.Context, email string) (*User, error)
}
```

**Mock generation**:

```bash
mockgen -source=repository.go -destination=mocks/mock_repository.go
```

**Using mock in tests**:

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

### In-Memory Test Implementations

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

## Test Data

### Test Data Factories

```go
package testdata

import (
	"time"
	"github.com/your-org/new-flowra/internal/domain"
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

// Usage
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

## CI/CD Integration

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
      mongodb:
        image: mongo:6.0
        env:
          MONGO_INITDB_ROOT_USERNAME: admin
          MONGO_INITDB_ROOT_PASSWORD: admin123
        options: >-
          --health-cmd "mongosh --eval 'db.adminCommand(\"ping\")'"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 27017:27017

      redis:
        image: redis:7
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
          MONGODB_URI: mongodb://admin:admin123@localhost:27017
          MONGODB_DATABASE: test_db
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

## Makefile Commands

```makefile
# Makefile
.PHONY: test test-unit test-integration test-e2e coverage lint

# All tests
test:
	go test -v ./...

# Unit tests
test-unit:
	go test -v -short ./...

# Integration tests
test-integration:
	go test -v -tags=integration ./...

# E2E tests
test-e2e:
	go test -v -tags=e2e ./tests/e2e/...

# Code coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

# Tests with race detection
test-race:
	go test -race -v ./...

# Benchmarks
bench:
	go test -bench=. -benchmem ./...

# Linter
lint:
	golangci-lint run

# Generate mocks
generate-mocks:
	go generate ./...

# Cleanup
clean-test:
	rm -f coverage.out coverage.html
	go clean -testcache
```

## Quality Metrics

### Code Coverage

- **Unit tests**: minimum 80%
- **Critical business logic**: minimum 95%
- **Integration paths**: minimum 70%

### Performance Metrics

- **API response time**: < 200ms for 95% of requests
- **Database queries**: < 100ms for simple queries
- **Memory usage**: no memory leaks

### Quality Metrics

- **Flaky tests**: < 1%
- **Test execution time**: all tests < 10 minutes
- **Code duplication**: < 5%

## Best Practices

### Writing Tests

1. **Follow AAA pattern**: Arrange, Act, Assert
2. **One assert per test**: each test should check one thing
3. **Independent tests**: tests should not depend on each other
4. **Descriptive names**: name should explain what is being tested
5. **Test behavior, not implementation**: focus on inputs and outputs

### Test Structure

```go
func TestServiceName_MethodName_Scenario(t *testing.T) {
// Arrange - setup data and mocks

// Act - execute tested action

// Assert - check results
}
```

### Test File Organization

```
internal/
├── service/
│   ├── user_service.go
│   ├── user_service_test.go      # Unit tests
│   └── user_service_integration_test.go  # Integration tests
├── repository/
│   ├── user_repository.go
│   └── user_repository_test.go
└── testdata/
    ├── fixtures/
    ├── factories/
    └── helpers/
```

## Troubleshooting

### Common Problems

**Flaky tests**:

- Use deterministic data
- Avoid sleep in favor of synchronization
- Isolate external dependencies

**Slow tests**:

- Use in-memory databases for unit tests
- Parallelize tests where possible
- Optimize setup/teardown

**Race conditions**:

- Always run tests with `-race`
- Use proper synchronization
- Test concurrent scenarios

## Conclusion

A quality testing strategy is the foundation of reliable and maintainable code. Following the described practices will
help ensure high product quality and confidence in changes.

### Key Principles:

- Write tests as part of development, not after
- Maintain high code coverage with tests
- Regularly refactor tests along with production code
- Automate test execution in CI/CD
- Use metrics to monitor test quality

---

*Last updated: [Current date]*  
*Version: 1.0*  
*Maintained by: QA Team*

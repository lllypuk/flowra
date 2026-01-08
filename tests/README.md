# Testing Guide

This directory contains all test utilities, mocks, and integration tests for the flowra project.

## Structure

```
tests/
├── README.md              # This file
├── mocks/                 # Mock implementations for testing
│   ├── user_repository.go
│   └── ...
├── testutil/              # Testing utilities
│   ├── db.go             # Helpers for database operations
│   └── fixtures.go       # Test data (builders)
└── integration/           # Integration tests
    └── usecase/
        └── ...
```

## Test Types

### 1. Unit Tests

**Location**: Next to the tested code (e.g., `create_task_test.go` next to `create_task.go`)

**Characteristics**:
- Fast (< 1 second per test)
- Use in-memory Event Store
- Use mock repositories
- No external dependencies

**Run**:
```bash
# All unit tests
make test-unit

# Specific package
go test ./internal/usecase/task/
```

### 2. Integration Tests

**Location**: `tests/integration/`

**Characteristics**:
- Slower (1-5 seconds per test)
- Use real PostgreSQL
- Require build tag `integration`
- Each test creates its own isolated schema

**Run**:
```bash
# All integration tests
make test-integration

# Manually
TEST_DATABASE_URL="postgresql://postgres:postgres123@localhost:5432/test_db?sslmode=disable" \
  go test -tags=integration ./tests/integration/...
```

### 3. Coverage

**Check coverage**:
```bash
# Generate HTML report
make test-coverage

# Check threshold (80%)
make test-coverage-check

# View coverage in terminal
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Test Utilities

### InMemoryEventStore

In-memory implementation of EventStore for fast unit tests.

```go
import "github.com/lllypuk/flowra/internal/infrastructure/eventstore"

eventStore := eventstore.NewInMemoryEventStore()
// Use in tests
```

### MockUserRepository

Mock implementation of UserRepository for testing.

```go
import "github.com/lllypuk/flowra/tests/mocks"

repo := mocks.NewMockUserRepository()
repo.AddUser(userID, "testuser", "Test User")
exists, _ := repo.Exists(ctx, userID) // true
```

### Test Fixtures (Builders)

Builders for creating test commands with fluent API.

```go
import "github.com/lllypuk/flowra/tests/testutil"

// Basic command with default values
cmd := testutil.CreateTaskCommandFixture()

// Command with custom values
cmd := testutil.BuildCreateTaskCommand(
    testutil.WithTitle("Custom Task"),
    testutil.WithPriority(task.PriorityHigh),
    testutil.WithAssignee(assigneeID),
    testutil.WithDueDate(tomorrow),
)
```

Available builders:
- `CreateTaskCommandFixture()` / `BuildCreateTaskCommand(...)`
- `ChangeStatusCommandFixture(taskID)` / `BuildChangeStatusCommand(taskID, ...)`
- `AssignTaskCommandFixture(taskID, assigneeID)` / `BuildAssignTaskCommand(...)`
- `ChangePriorityCommandFixture(taskID)` / `BuildChangePriorityCommand(taskID, ...)`
- `SetDueDateCommandFixture(taskID)` / `BuildSetDueDateCommand(taskID, ...)`

### Database Helpers (Integration Tests)

Helpers for working with test database in integration tests.

```go
//go:build integration

import "github.com/lllypuk/flowra/tests/testutil"

func TestSomething_Integration(t *testing.T) {
    db := testutil.SetupTestDatabase(t)
    defer testutil.TeardownTestDatabase(t, db)
    
    // Use db to create eventStore, etc.
}
```

## Best Practices

### Table-Driven Tests

Use table-driven approach for multiple scenarios:

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expectedErr error
    }{
        {name: "valid input", input: "test", expectedErr: nil},
        {name: "empty input", input: "", expectedErr: ErrEmptyInput},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)
            if tt.expectedErr == nil {
                assert.NoError(t, err)
            } else {
                assert.ErrorIs(t, err, tt.expectedErr)
            }
        })
    }
}
```

### Test Naming Convention

```go
// Pattern: Test<FunctionName>_<Scenario>

// Success cases
TestCreateTaskUseCase_Success
TestCreateTaskUseCase_WithDefaults

// Errors
TestCreateTaskUseCase_ValidationErrors
TestCreateTaskUseCase_EmptyTitle
TestCreateTaskUseCase_TaskNotFound

// Edge cases
TestCreateTaskUseCase_Idempotent
TestCreateTaskUseCase_ConcurrentUpdate
```

### Assertions

Use `testify/assert` and `testify/require`:

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// require - stops test on error (for critical checks)
require.NoError(t, err)
require.NotNil(t, result)

// assert - continues execution (for multiple checks)
assert.Equal(t, expected, actual)
assert.Len(t, events, 1)
assert.True(t, condition)
```

## Makefile Targets

```bash
make test                  # All tests
make test-unit            # Only unit tests
make test-integration     # Only integration tests
make test-coverage        # Generate HTML report
make test-coverage-check  # Check threshold (80%)
make test-verbose         # Tests with verbose output
make test-clean           # Clean cache and coverage files
```

## Environment Variables

### Integration Tests

- `TEST_DATABASE_URL` - URL of test PostgreSQL database
  - Example: `postgresql://postgres:postgres123@localhost:5432/test_db?sslmode=disable`
  - If not set, integration tests will be skipped

## CI/CD (Future)

When CI is configured, it will automatically run:
1. Unit tests on every push
2. Coverage check (minimum 80%)
3. Integration tests on PR
4. Linting

## Coverage Goals

```
internal/usecase/task/          > 80%
internal/domain/task/           > 90%
internal/infrastructure/        > 70%
```

## Troubleshooting

### Integration tests do not run

Check:
1. PostgreSQL is running: `docker-compose ps postgres`
2. `TEST_DATABASE_URL` environment variable is set
3. Build tag `integration` is specified: `-tags=integration`

### Coverage is low

1. Check which packages are not covered: `go tool cover -func=coverage.out`
2. Add tests for uncovered code
3. Use HTML report for visualization: `make test-coverage` → `coverage.html`

### Tests are slow

1. Make sure you are using unit tests, not integration tests
2. Use `InMemoryEventStore` instead of database
3. Avoid `time.Sleep()` in tests
4. Use mocks for external dependencies

## References

- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Testify Documentation](https://github.com/stretchr/testify)
# Flowra Coding Standards

## Overview

This document defines the coding standards for the Flowra project. Following these standards ensures consistency, readability, and maintainability of the code.

## General Principles

### Clean Code
- Code should be self-documenting
- Prefer clarity over performance (unless performance is critical)
- Use descriptive names for variables, functions, and types
- Functions should do one thing well
- Avoid deep nesting (max 3 levels)

### SOLID Principles
- **Single Responsibility Principle**: A class/function has one reason to change
- **Open/Closed Principle**: Open for extension, closed for modification
- **Liskov Substitution Principle**: Subtypes must be substitutable for their base types
- **Interface Segregation Principle**: Clients should not depend on unused interfaces
- **Dependency Inversion Principle**: Dependencies should be on abstractions, not concrete implementations

## Go Standards

### General Rules

#### Formatting
- Use `gofmt` or `goimports` for automatic formatting
- Line length no more than 120 characters
- Use tabs for indentation
- Empty line between logical code blocks

```go
// Good
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

// Bad
func ProcessUser(ctx context.Context,userID int)(*User,error){
user,err:=userRepo.FindByID(ctx,userID)
if err!=nil{return nil,fmt.Errorf("finding user: %w",err)}
if user.IsActive{return user,nil}
return nil,ErrUserInactive
}
```

#### Naming

**Variables and functions**: camelCase
```go
var userCount int
var isUserActive bool

func getUserProfile() {}
func calculateTotalScore() {}
```

**Constants**: UPPER_CASE or PascalCase for exported
```go
const (
    MaxRetryCount = 3
    defaultTimeout = 30 * time.Second
)
```

**Types**: PascalCase
```go
type UserService struct {}
type HTTPClient interface {}
```

**Packages**: short, lowercase, no underscores
```go
package auth
package userservice
```

#### Comments

**Public elements**: comments are required
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

**Private elements**: comments are recommended for complex logic
```go
// validateUserInput checks if the user input is valid
func validateUserInput(req CreateUserRequest) error {
    // validation logic
}
```

#### Error Handling

**Wrap errors with context**:
```go
user, err := repo.FindByID(ctx, id)
if err != nil {
    return nil, fmt.Errorf("finding user by ID %d: %w", id, err)
}
```

**Creating custom errors**:
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

**Don't ignore errors**:
```go
// Good
if err := someOperation(); err != nil {
    log.Error("operation failed", "error", err)
    return err
}

// Bad
someOperation() // error is ignored
```

#### Working with Context

**First function parameter should be context.Context**:
```go
func ProcessRequest(ctx context.Context, req Request) (*Response, error) {
    // implementation
}
```

**Pass context to all external calls**:
```go
func (s *Service) ProcessUser(ctx context.Context, userID int) error {
    user, err := s.repo.GetUser(ctx, userID) // pass context
    if err != nil {
        return err
    }
    
    return s.notifier.Send(ctx, user) // and here too
}
```

#### Interfaces

**Define interfaces on the consumer side**:
```go
// In the service package
type UserRepository interface {
    GetUser(ctx context.Context, id int) (*User, error)
    SaveUser(ctx context.Context, user *User) error
}

type UserService struct {
    repo UserRepository // dependency on interface
}
```

**Interfaces should be small**:
```go
// Good
type Reader interface {
    Read([]byte) (int, error)
}

type Writer interface {
    Write([]byte) (int, error)
}

// Bad - too many methods
type FileManager interface {
    Read([]byte) (int, error)
    Write([]byte) (int, error)
    Close() error
    Seek(int64, int) (int64, error)
    Stat() (FileInfo, error)
    // ... 10 more methods
}
```

### Structs and Methods

#### Structs
```go
// Good - grouping related fields
type User struct {
    // Identity
    ID    int    `json:"id" db:"id"`
    Email string `json:"email" db:"email"`
    
    // Profile
    FirstName string    `json:"first_name" db:"first_name"`
    LastName  string    `json:"last_name" db:"last_name"`
    Bio       *string   `json:"bio" db:"bio"` // nullable fields as pointers
    
    // Metadata
    CreatedAt time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
    DeletedAt *time.Time `json:"-" db:"deleted_at"`
}
```

#### Methods
```go
// Methods should be attached to the appropriate type
func (u *User) IsActive() bool {
    return u.DeletedAt == nil
}

func (u *User) FullName() string {
    return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

// Constructors
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

### Packages and Imports

#### Import Organization
```go
import (
    // Standard library
    "context"
    "fmt"
    "time"
    
    // External dependencies
    "github.com/gin-gonic/gin"
    "github.com/jmoiron/sqlx"
    
    // Internal packages
    "github.com/your-org/new-flowra/internal/domain"
    "github.com/your-org/new-flowra/pkg/logger"
)
```

#### Package Structure
```
internal/
├── domain/          # Domain models
├── application/     # Use cases
├── infrastructure/  # External dependencies
└── presentation/    # HTTP handlers
```

### Testing

#### Test Naming
```go
func TestUserService_CreateUser_Success(t *testing.T) {}
func TestUserService_CreateUser_InvalidEmail(t *testing.T) {}
func TestUserService_CreateUser_EmailAlreadyExists(t *testing.T) {}
```

#### Test Structure (AAA pattern)
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

#### Table-driven tests for multiple scenarios
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

### Migrations

#### File Naming
```
001_create_users_table.up.sql
001_create_users_table.down.sql
002_add_user_profiles.up.sql
002_add_user_profiles.down.sql
```

#### Migration Structure
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

### Database Naming

#### Tables: snake_case, plural
```sql
users
user_profiles
team_memberships
project_tasks
```

#### Columns: snake_case
```sql
first_name
created_at
is_active
team_id
```

#### Indexes: idx_table_column(s)
```sql
idx_users_email
idx_team_memberships_user_id_team_id
```

#### Foreign Keys: fk_table_referenced_table
```sql
fk_user_profiles_users
fk_team_memberships_teams
```

## API Standards

### REST API

#### URL Structure
```
GET    /api/v1/users           # Get list of users
POST   /api/v1/users           # Create user
GET    /api/v1/users/{id}      # Get user
PUT    /api/v1/users/{id}      # Update user
DELETE /api/v1/users/{id}      # Delete user

GET    /api/v1/users/{id}/teams # Get user's teams
```

#### HTTP Status Codes
```
200 OK                  # Successful request
201 Created            # Resource created
204 No Content         # Successful, no content
400 Bad Request        # Invalid request
401 Unauthorized       # Not authenticated
403 Forbidden          # Access denied
404 Not Found          # Resource not found
409 Conflict           # Conflict (e.g., email already exists)
422 Unprocessable Entity # Validation error
500 Internal Server Error # Internal server error
```

#### JSON Structures

**Successful responses**:
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

**Lists with pagination**:
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

**Errors**:
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

## Security

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
// Good - use parameterized queries
func (r *repository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
    query := `SELECT id, email, first_name FROM users WHERE email = $1`
    var user User
    err := r.db.GetContext(ctx, &user, query, email)
    return &user, err
}

// Bad - vulnerable to SQL injection
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
    PasswordHash string `json:"-"` // exclude from JSON
}
```

## Logging

### Structured Logs
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

### Logging Levels
- **DEBUG**: Detailed information for debugging
- **INFO**: General information about application operation
- **WARN**: Warnings about potential issues
- **ERROR**: Errors that don't stop the application
- **FATAL**: Critical errors that stop the application

## Performance Guidelines

### Database
- Use indexes for frequently queried columns
- Avoid N+1 problems (use JOIN or batch loading)
- Use connection pooling
- Use prepared statements

### Caching
```go
// Example of result caching
func (s *UserService) GetUser(ctx context.Context, id int) (*User, error) {
    // Try to get from cache
    if cached, found := s.cache.Get(fmt.Sprintf("user:%d", id)); found {
        return cached.(*User), nil
    }
    
    // If not in cache, get from DB
    user, err := s.repo.GetUser(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // Save to cache
    s.cache.Set(fmt.Sprintf("user:%d", id), user, 5*time.Minute)
    
    return user, nil
}
```

## Code Review Checklist

### General
- [ ] Code follows project standards
- [ ] Variable and function names are clear
- [ ] No code duplication
- [ ] Functions are not too large (< 50 lines)
- [ ] Necessary comments are present

### Go Specific
- [ ] Uses `gofmt`
- [ ] All errors are handled
- [ ] Uses context.Context where needed
- [ ] No race conditions
- [ ] Resources are released (defer close())

### Tests
- [ ] Tests are written for new code
- [ ] Test coverage is sufficient (>80%)
- [ ] Tests pass successfully
- [ ] Error cases are tested

### Security
- [ ] Input validation is implemented
- [ ] No hardcoded secrets
- [ ] SQL injection protection
- [ ] Proper handling of sensitive data

## Tools

### Required
- `gofmt` - code formatting
- `go vet` - static analysis
- `golangci-lint` - linter
- `go test` - testing

### Recommended
- `staticcheck` - extended static analysis
- `govulncheck` - vulnerability scanning
- `goimports` - import management
- `godoc` - documentation generation

### .golangci.yml Configuration
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

*Last updated: [Current date]*  
*Version: 1.0*  
*Maintained by: Development Team*
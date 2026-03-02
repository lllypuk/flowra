# Frontend E2E Tests

Browser-based end-to-end tests for the Flowra frontend using Playwright.

## Prerequisites

1. **Install Playwright browsers:**
   ```bash
   make playwright-install
   ```

2. **Start the application server and infrastructure:**
   
   **Option A: Full-stack runtime (recommended)**
   ```bash
   make dev
   ```
   
   **Option B: Manual split setup**
   ```bash
   # Start infrastructure
   docker-compose up -d mongodb redis keycloak
   
   # Start worker (required for outbox/projection freshness)
   FLOWRA_DEV_MODE=fullstack go run ./cmd/worker

   # Start API server
   FLOWRA_DEV_MODE=fullstack go run ./cmd/api
   ```

3. **Verify server is running:**
   ```bash
   curl http://localhost:8080/health
   ```

## Running Tests

### Run all frontend E2E tests
```bash
make test-e2e-frontend
```

### Run board + sidebar smoke regression only
```bash
make test-e2e-frontend-smoke
```

### Run with visible browser (for debugging)
```bash
HEADLESS=false make test-e2e-frontend
```

### Run specific test
```bash
go test -tags=e2e -v -run TestFrontend_LoginPage_Renders ./tests/e2e/frontend/
```

## Test Behavior

- Tests will automatically **skip** if the server is not available at `http://localhost:8080`
- Tests require a fully functional backend with Keycloak for SSO authentication
- Default browser mode is **headless** (invisible)
- Set `HEADLESS=false` environment variable to see the browser during test execution

## Test Coverage

Frontend E2E tests cover:
- ✅ Authentication (Login/Logout via Keycloak SSO)
- ✅ Workspace management (Create, Edit, Delete)
- ✅ Chat functionality (Create chats, Send messages)
- ✅ Kanban board (View, Drag & Drop)
- ✅ Notifications (Dropdown, Mark as read)
- ✅ Accessibility (Keyboard navigation, Skip links, ARIA)
- ✅ Responsive design (Mobile, Tablet layouts)

Manual smoke checklist for critical board + sidebar regression:
- [`SMOKE_CHECKLIST.md`](./SMOKE_CHECKLIST.md)

## Troubleshooting

### Tests are skipped
```
WARNING: Server is not available at http://localhost:8080
```
**Solution:** Start full stack with `make dev` (preferred) or run API+worker manually.

### Playwright not installed
```
Failed to start Playwright
```
**Solution:** Run `make playwright-install`

### Keycloak login fails
- Ensure Keycloak is running: `docker-compose ps keycloak`
- Check Keycloak is configured with test user credentials
- Verify Keycloak is accessible at `http://localhost:8090`

## Configuration

Test configuration is in `frontend_test.go`:
```go
const (
    baseURL          = "http://localhost:8080"
    defaultTimeout   = 30 * time.Second
    keycloakUser     = "testuser"
    keycloakPassword = "test123"
)
```

Override base URL with environment variable:
```bash
E2E_BASE_URL=http://localhost:3000 make test-e2e-frontend
```

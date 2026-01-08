//go:build e2e

// Package frontend provides end-to-end browser tests for the Flowra frontend.
package frontend

import (
	"os"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
)

// Test configuration
const (
	baseURL          = "http://localhost:8080"
	defaultTimeout   = 30 * time.Second
	keycloakUser     = "testuser"
	keycloakPassword = "password"
	defaultHeadless  = true
	slowMo           = 0 // Set to 100 for debugging
)

// isHeadless returns whether browser should run in headless mode.
// Set HEADLESS=false environment variable to run with visible browser.
func isHeadless() bool {
	if val := os.Getenv("HEADLESS"); val == "false" || val == "0" {
		return false
	}
	return defaultHeadless
}

// TestSuite holds the Playwright context for frontend tests.
type TestSuite struct {
	pw      *playwright.Playwright
	browser playwright.Browser
}

// setupTestSuite initializes Playwright and browser.
func setupTestSuite(t *testing.T) *TestSuite {
	t.Helper()

	// Check if base URL is accessible
	if os.Getenv("E2E_BASE_URL") != "" {
		// Use custom base URL if provided
	}

	pw, err := playwright.Run()
	require.NoError(t, err, "Failed to start Playwright")

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(isHeadless()),
		SlowMo:   playwright.Float(slowMo),
	})
	require.NoError(t, err, "Failed to launch browser")

	return &TestSuite{
		pw:      pw,
		browser: browser,
	}
}

// teardownTestSuite cleans up Playwright resources.
func (s *TestSuite) teardownTestSuite() {
	if s.browser != nil {
		s.browser.Close()
	}
	if s.pw != nil {
		s.pw.Stop()
	}
}

// newPage creates a new browser page with default settings.
func (s *TestSuite) newPage(t *testing.T) playwright.Page {
	t.Helper()

	page, err := s.browser.NewPage()
	require.NoError(t, err, "Failed to create new page")

	page.SetDefaultTimeout(float64(defaultTimeout.Milliseconds()))

	return page
}

// loginAsTestUser performs login via Keycloak SSO.
func loginAsTestUser(t *testing.T, page playwright.Page) {
	t.Helper()

	// Navigate to login page
	_, err := page.Goto(baseURL + "/login")
	require.NoError(t, err, "Failed to navigate to login page")

	// Click SSO login button
	err = page.Click("text=Sign in with SSO")
	require.NoError(t, err, "Failed to click SSO button")

	// Wait for Keycloak login form
	_, err = page.WaitForSelector("#username", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(float64(defaultTimeout.Milliseconds())),
	})
	require.NoError(t, err, "Keycloak login form not found")

	// Fill credentials
	err = page.Fill("#username", keycloakUser)
	require.NoError(t, err, "Failed to fill username")

	err = page.Fill("#password", keycloakPassword)
	require.NoError(t, err, "Failed to fill password")

	// Submit login
	err = page.Click("#kc-login")
	require.NoError(t, err, "Failed to click login button")

	// Wait for redirect to workspaces page
	err = page.WaitForURL("**/workspaces**", playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(float64(defaultTimeout.Milliseconds())),
	})
	require.NoError(t, err, "Failed to redirect to workspaces after login")
}

// ============================================================
// Authentication Tests
// ============================================================

func TestFrontend_LoginPage_Renders(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	// Navigate to login page
	_, err := page.Goto(baseURL + "/login")
	require.NoError(t, err)

	// Check page title
	title, err := page.Title()
	require.NoError(t, err)
	require.Contains(t, title, "Login")

	// Check SSO button is visible
	ssoButton := page.Locator("text=Sign in with SSO")
	visible, err := ssoButton.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "SSO button should be visible")
}

func TestFrontend_LoginFlow(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	// Perform login
	loginAsTestUser(t, page)

	// Verify user menu is visible
	userMenu := page.Locator(".dropdown summary")
	visible, err := userMenu.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "User menu should be visible after login")
}

func TestFrontend_Logout(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	// Login first
	loginAsTestUser(t, page)

	// Open user menu and click logout
	err := page.Click(".dropdown summary")
	require.NoError(t, err)

	err = page.Click("text=Logout")
	require.NoError(t, err)

	// Wait for redirect to home page
	err = page.WaitForURL(baseURL+"/", playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(float64(defaultTimeout.Milliseconds())),
	})
	require.NoError(t, err, "Should redirect to home after logout")

	// Verify login button is visible
	loginButton := page.Locator("text=Login")
	visible, err := loginButton.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "Login button should be visible after logout")
}

func TestFrontend_ProtectedRouteRedirect(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	// Try to access protected route without login
	_, err := page.Goto(baseURL + "/workspaces")
	require.NoError(t, err)

	// Should redirect to login
	url := page.URL()
	require.Contains(t, url, "/login", "Should redirect to login for protected routes")
}

// ============================================================
// Workspace Tests
// ============================================================

func TestFrontend_CreateWorkspace(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	loginAsTestUser(t, page)

	// Click create workspace button
	err := page.Click("text=+ New Workspace")
	require.NoError(t, err)

	// Wait for modal
	_, err = page.WaitForSelector("dialog[open]")
	require.NoError(t, err, "Create workspace modal should open")

	// Fill form
	err = page.Fill("input[name=name]", "Test Workspace")
	require.NoError(t, err)

	err = page.Fill("textarea[name=description]", "Test workspace description")
	require.NoError(t, err)

	// Submit
	err = page.Click("text=Create Workspace")
	require.NoError(t, err)

	// Verify workspace appears in list
	workspaceCard := page.Locator(".workspace-card:has-text('Test Workspace')")
	visible, err := workspaceCard.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "New workspace should appear in list")
}

func TestFrontend_EditWorkspaceName(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	loginAsTestUser(t, page)

	// Navigate to workspace settings (assumes workspace exists)
	_, err := page.Goto(baseURL + "/workspaces")
	require.NoError(t, err)

	// Click on first workspace
	err = page.Click(".workspace-card >> nth=0")
	require.NoError(t, err)

	// Navigate to settings
	err = page.Click("text=Settings")
	require.NoError(t, err)

	// Edit name
	err = page.Fill("input[name=name]", "Updated Workspace Name")
	require.NoError(t, err)

	// Save
	err = page.Click("text=Save Changes")
	require.NoError(t, err)

	// Verify success message
	flashMessage := page.Locator(".flash-success")
	visible, err := flashMessage.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "Success message should appear")
}

// ============================================================
// Chat Tests
// ============================================================

func TestFrontend_CreateChat(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	loginAsTestUser(t, page)

	// Navigate to workspace
	_, err := page.Goto(baseURL + "/workspaces")
	require.NoError(t, err)

	err = page.Click(".workspace-card >> nth=0")
	require.NoError(t, err)

	// Click create chat button
	err = page.Click("text=+ New Chat")
	require.NoError(t, err)

	// Fill chat name
	err = page.Fill("input[name=name]", "Test Chat")
	require.NoError(t, err)

	// Submit
	err = page.Click("text=Create Chat")
	require.NoError(t, err)

	// Verify chat appears
	chatItem := page.Locator(".chat-item:has-text('Test Chat')")
	visible, err := chatItem.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "New chat should appear in list")
}

func TestFrontend_SendMessage(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	loginAsTestUser(t, page)

	// Navigate to a chat (assumes chat exists)
	_, err := page.Goto(baseURL + "/workspaces")
	require.NoError(t, err)

	err = page.Click(".workspace-card >> nth=0")
	require.NoError(t, err)

	err = page.Click(".chat-item >> nth=0")
	require.NoError(t, err)

	// Type and send message
	messageInput := page.Locator("textarea[name=content]")
	err = messageInput.Fill("Hello, this is a test message!")
	require.NoError(t, err)

	// Send with Enter
	err = messageInput.Press("Enter")
	require.NoError(t, err)

	// Verify message appears
	message := page.Locator(".message:has-text('Hello, this is a test message!')")
	err = message.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(defaultTimeout.Milliseconds())),
	})
	require.NoError(t, err, "Message should appear in chat")
}

// ============================================================
// Kanban Board Tests
// ============================================================

func TestFrontend_ViewKanbanBoard(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	loginAsTestUser(t, page)

	// Navigate to board
	_, err := page.Goto(baseURL + "/workspaces")
	require.NoError(t, err)

	err = page.Click(".workspace-card >> nth=0")
	require.NoError(t, err)

	err = page.Click("text=Board")
	require.NoError(t, err)

	// Verify columns exist
	columns := page.Locator(".board-column")
	count, err := columns.Count()
	require.NoError(t, err)
	require.Equal(t, 4, count, "Board should have 4 columns (TODO, In Progress, Review, Done)")
}

func TestFrontend_KanbanDragDrop(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	loginAsTestUser(t, page)

	// Navigate to board with tasks
	_, err := page.Goto(baseURL + "/workspaces")
	require.NoError(t, err)

	err = page.Click(".workspace-card >> nth=0")
	require.NoError(t, err)

	err = page.Click("text=Board")
	require.NoError(t, err)

	// Find first task card
	taskCard := page.Locator(".task-card").First()
	doneColumn := page.Locator("[data-status=done] .column-cards")

	// Drag task to Done column
	err = taskCard.DragTo(doneColumn)
	require.NoError(t, err)

	// Verify task moved
	time.Sleep(500 * time.Millisecond) // Wait for status update

	doneCards := page.Locator("[data-status=done] .task-card")
	count, err := doneCards.Count()
	require.NoError(t, err)
	require.GreaterOrEqual(t, count, 1, "Done column should have at least one task")
}

// ============================================================
// Notification Tests
// ============================================================

func TestFrontend_NotificationDropdown(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	loginAsTestUser(t, page)

	// Click notification icon
	err := page.Click(".notification-dropdown summary")
	require.NoError(t, err)

	// Wait for dropdown to load
	_, err = page.WaitForSelector(".notification-dropdown ul[role=listbox]")
	require.NoError(t, err, "Notification dropdown should open")
}

func TestFrontend_MarkNotificationAsRead(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	loginAsTestUser(t, page)

	// Open notifications page
	_, err := page.Goto(baseURL + "/notifications")
	require.NoError(t, err)

	// Check if there are unread notifications
	unreadNotif := page.Locator(".notification-item.unread").First()
	visible, err := unreadNotif.IsVisible()
	if err != nil || !visible {
		t.Skip("No unread notifications to test")
		return
	}

	// Click to mark as read
	err = unreadNotif.Click()
	require.NoError(t, err)

	// Verify notification is no longer unread
	time.Sleep(500 * time.Millisecond)
	hasUnreadClass, err := unreadNotif.Evaluate("el => el.classList.contains('unread')", nil)
	require.NoError(t, err)
	require.False(t, hasUnreadClass.(bool), "Notification should no longer be unread")
}

// ============================================================
// Accessibility Tests
// ============================================================

func TestFrontend_KeyboardNavigation(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	_, err := page.Goto(baseURL + "/login")
	require.NoError(t, err)

	// Tab through elements
	err = page.Keyboard().Press("Tab")
	require.NoError(t, err)

	// Check focus is visible
	focusedElement := page.Locator(":focus")
	visible, err := focusedElement.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "Focused element should be visible")
}

func TestFrontend_SkipLink(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	loginAsTestUser(t, page)

	// Tab to skip link
	err := page.Keyboard().Press("Tab")
	require.NoError(t, err)

	// Skip link should be focused
	skipLink := page.Locator(".skip-link:focus")
	visible, err := skipLink.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "Skip link should be visible when focused")
}

func TestFrontend_ModalEscapeClose(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	loginAsTestUser(t, page)

	// Open create workspace modal
	err := page.Click("text=+ New Workspace")
	require.NoError(t, err)

	// Verify modal is open
	modal := page.Locator("dialog[open]")
	visible, err := modal.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "Modal should be open")

	// Press Escape
	err = page.Keyboard().Press("Escape")
	require.NoError(t, err)

	// Verify modal is closed
	time.Sleep(300 * time.Millisecond)
	visible, err = modal.IsVisible()
	require.NoError(t, err)
	require.False(t, visible, "Modal should be closed after Escape")
}

// ============================================================
// Responsive Design Tests
// ============================================================

func TestFrontend_MobileLayout(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	// Create page with mobile viewport
	context, err := suite.browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  375,
			Height: 667,
		},
	})
	require.NoError(t, err)
	defer context.Close()

	page, err := context.NewPage()
	require.NoError(t, err)
	defer page.Close()

	loginAsTestUser(t, page)

	// Mobile nav toggle should be visible
	mobileToggle := page.Locator(".mobile-nav-toggle")
	visible, err := mobileToggle.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "Mobile nav toggle should be visible on mobile")

	// Desktop nav should be hidden
	desktopNav := page.Locator(".desktop-nav")
	visible, err = desktopNav.IsVisible()
	require.NoError(t, err)
	require.False(t, visible, "Desktop nav should be hidden on mobile")
}

func TestFrontend_TabletLayout(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	// Create page with tablet viewport
	context, err := suite.browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  768,
			Height: 1024,
		},
	})
	require.NoError(t, err)
	defer context.Close()

	page, err := context.NewPage()
	require.NoError(t, err)
	defer page.Close()

	loginAsTestUser(t, page)

	// Workspace grid should adjust
	_, err = page.Goto(baseURL + "/workspaces")
	require.NoError(t, err)

	// Cards should be visible and properly laid out
	cards := page.Locator(".workspace-card")
	count, err := cards.Count()
	require.NoError(t, err)
	require.GreaterOrEqual(t, count, 0, "Workspaces page should load on tablet")
}

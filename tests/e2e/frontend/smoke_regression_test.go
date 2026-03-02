//go:build e2e

package frontend

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
)

func TestFrontend_BoardSidebarSmokeRegression(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.teardownTestSuite()

	page := suite.newPage(t)
	defer page.Close()

	loginAsTestUser(t, page)

	workspaceName := fmt.Sprintf("PR12 Smoke %d", time.Now().UnixNano())
	chatTitle := fmt.Sprintf("PR12-Task-%d", time.Now().UnixNano())
	workspaceSelector := withExactText(".workspace-card", workspaceName)
	chatItemSelector := withExactText("a.chat-item", chatTitle)
	taskCardSelector := withExactText(".task-card", chatTitle)

	_, err := page.Goto(baseURL + "/workspaces")
	require.NoError(t, err)

	err = page.Click("button:has-text('+ New Workspace')")
	require.NoError(t, err)

	_, err = page.WaitForSelector("#create-workspace-modal", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(float64(defaultTimeout.Milliseconds())),
	})
	require.NoError(t, err)

	err = page.Fill("#name", workspaceName)
	require.NoError(t, err)
	err = page.Fill("#description", "PR-12 smoke regression workspace")
	require.NoError(t, err)
	err = page.Click("#create-workspace-modal button:has-text('Create Workspace')")
	require.NoError(t, err)

	waitForLocatorCountAtLeast(t, page.Locator(workspaceSelector), 1, defaultTimeout, "new workspace card to appear")

	err = page.Click(workspaceSelector)
	require.NoError(t, err)
	err = page.WaitForURL(regexp.MustCompile(`/workspaces/[0-9a-f-]+$`), playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(float64(defaultTimeout.Milliseconds())),
	})
	require.NoError(t, err)

	err = page.Click("a:has-text('Chats')")
	require.NoError(t, err)
	err = page.WaitForURL(regexp.MustCompile(`/workspaces/[0-9a-f-]+/chats`), playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(float64(defaultTimeout.Milliseconds())),
	})
	require.NoError(t, err)

	waitForLocatorCountAtLeast(t, page.Locator(".chat-list"), 1, defaultTimeout, "chat list to load")

	err = page.Click("button:has-text('+ New Chat')")
	require.NoError(t, err)

	_, err = page.WaitForSelector("#create-chat-modal", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(float64(defaultTimeout.Milliseconds())),
	})
	require.NoError(t, err)

	err = page.Fill("#chat-name", chatTitle)
	require.NoError(t, err)
	selectOptionByValue(t, page.Locator("#chat-type"), "task")
	err = page.Click("#create-chat-modal button:has-text('Create Chat')")
	require.NoError(t, err)

	err = page.WaitForURL(regexp.MustCompile(`/workspaces/[0-9a-f-]+/chats/[0-9a-f-]+`), playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(float64(defaultTimeout.Milliseconds())),
	})
	require.NoError(t, err)

	chatURL := page.URL()
	workspaceID := workspaceIDFromChatURL(t, chatURL)
	boardURL := fmt.Sprintf("%s/workspaces/%s/board", baseURL, workspaceID)

	waitForLocatorCountAtLeast(t, page.Locator(chatItemSelector), 1, defaultTimeout, "new chat item in sidebar")
	assertLocatorCountEqual(t, page.Locator(chatItemSelector), 1, "chat sidebar duplicates for new chat")

	systemMessages := page.Locator(".message.system-message")
	systemMessageCountBefore, err := systemMessages.Count()
	require.NoError(t, err)

	selectOptionByValue(t, page.Locator("#task-status"), "In Progress")
	waitForInputValue(t, page.Locator("#task-status"), "In Progress", 10*time.Second, "status select after change")
	waitForTextContains(t, page.Locator("#chat-header-status"), "In Progress", 10*time.Second, "chat header status")
	waitForLocatorCountAtLeast(t, systemMessages, systemMessageCountBefore+1, 10*time.Second, "status system message")

	selectOptionByValue(t, page.Locator("#task-priority"), "High")
	waitForInputValue(t, page.Locator("#task-priority"), "High", 10*time.Second, "priority select after change")
	waitForLocatorCountAtLeast(t, systemMessages, systemMessageCountBefore+2, 10*time.Second, "priority system message")

	assigneeSelect := page.Locator("#task-assignee")
	waitForLocatorCountAtLeast(t, assigneeSelect.Locator("option"), 2, defaultTimeout, "assignee options")

	assigneeValue, err := assigneeSelect.Locator("option").Nth(1).GetAttribute("value")
	require.NoError(t, err)
	require.NotEmpty(t, assigneeValue)

	selectOptionByValue(t, assigneeSelect, assigneeValue)
	waitForInputValue(t, assigneeSelect, assigneeValue, 10*time.Second, "assignee after assign")

	selectOptionByValue(t, assigneeSelect, "")
	waitForInputValue(t, assigneeSelect, "", 10*time.Second, "assignee after unassign")

	dueDate := time.Now().UTC().Add(72 * time.Hour).Format("2006-01-02")
	dueDateInput := page.Locator("#task-due-date")
	err = dueDateInput.Fill(dueDate)
	require.NoError(t, err)
	err = dueDateInput.DispatchEvent("change", nil)
	require.NoError(t, err)
	waitForInputValue(t, dueDateInput, dueDate, 10*time.Second, "due date after set")
	time.Sleep(700 * time.Millisecond) // Sidebar due-date submit uses 500ms debounce.

	_, err = page.Goto(boardURL)
	require.NoError(t, err)

	waitForLocatorCountAtLeast(t, page.Locator(".board-column"), 4, defaultTimeout, "board columns")
	waitForLocatorCountAtLeast(t, page.Locator(taskCardSelector), 1, defaultTimeout, "task card on board")
	assertLocatorCountEqual(t, page.Locator(taskCardSelector), 1, "board duplicates after sidebar mutations")

	inProgressTaskCardSelector := ".column-cards[data-status='in_progress'] " + taskCardSelector
	waitForLocatorCountAtLeast(
		t,
		page.Locator(inProgressTaskCardSelector),
		1,
		10*time.Second,
		"task card in in_progress column after status mutation",
	)

	inProgressCard := page.Locator(inProgressTaskCardSelector).First()
	priorityValue, err := inProgressCard.GetAttribute("data-priority")
	require.NoError(t, err)
	require.Equal(t, "high", priorityValue)
	assertLocatorCountEqual(t, inProgressCard.Locator(".card-assignee"), 0, "card assignee after unassign")

	err = inProgressCard.DragTo(page.Locator(".column-cards[data-status='done']"))
	require.NoError(t, err)

	doneTaskCardSelector := ".column-cards[data-status='done'] " + taskCardSelector
	waitForLocatorCountAtLeast(
		t,
		page.Locator(doneTaskCardSelector),
		1,
		10*time.Second,
		"task card in done column after drag-and-drop",
	)
	waitForLocatorCountEqual(
		t,
		page.Locator(inProgressTaskCardSelector),
		0,
		10*time.Second,
		"in_progress column card count after drag-and-drop",
	)
	assertLocatorCountEqual(t, page.Locator(taskCardSelector), 1, "board duplicates after drag-and-drop")

	_, err = page.Goto(chatURL)
	require.NoError(t, err)

	waitForTextContains(t, page.Locator("#chat-header-status"), "Done", 10*time.Second, "chat header status after board move")
	waitForInputValue(t, page.Locator("#task-status"), "Done", 10*time.Second, "status select after board move")
	waitForInputValue(t, page.Locator("#task-priority"), "High", 10*time.Second, "priority after board move")
	waitForInputValue(t, page.Locator("#task-assignee"), "", 10*time.Second, "assignee after board move")
	waitForInputValue(t, page.Locator("#task-due-date"), dueDate, 10*time.Second, "due date after board move")
	assertLocatorCountEqual(t, page.Locator(chatItemSelector), 1, "chat sidebar duplicates after board move")

	_, err = page.Reload()
	require.NoError(t, err)

	waitForTextContains(t, page.Locator("#chat-header-status"), "Done", 10*time.Second, "chat header status after reload")
	waitForInputValue(t, page.Locator("#task-status"), "Done", 10*time.Second, "status after reload")
	waitForInputValue(t, page.Locator("#task-priority"), "High", 10*time.Second, "priority after reload")
	waitForInputValue(t, page.Locator("#task-assignee"), "", 10*time.Second, "assignee after reload")
	waitForInputValue(t, page.Locator("#task-due-date"), dueDate, 10*time.Second, "due date after reload")
	assertLocatorCountEqual(t, page.Locator(chatItemSelector), 1, "chat sidebar duplicates after reload")

	_, err = page.Goto(boardURL)
	require.NoError(t, err)
	waitForLocatorCountAtLeast(t, page.Locator(".board-column"), 4, defaultTimeout, "board columns after return")
	_, err = page.Reload()
	require.NoError(t, err)

	waitForLocatorCountAtLeast(
		t,
		page.Locator(doneTaskCardSelector),
		1,
		10*time.Second,
		"done column card after board reload",
	)
	assertLocatorCountEqual(t, page.Locator(taskCardSelector), 1, "board duplicates after reload")

	doneCard := page.Locator(doneTaskCardSelector).First()
	priorityValue, err = doneCard.GetAttribute("data-priority")
	require.NoError(t, err)
	require.Equal(t, "high", priorityValue)
	assertLocatorCountEqual(t, doneCard.Locator(".card-assignee"), 0, "done card assignee after reload")
	assertLocatorCountEqual(t, doneCard.Locator(".card-due"), 1, "done card due date after reload")
}

func withExactText(baseSelector, text string) string {
	return fmt.Sprintf(`%s:has-text("%s")`, baseSelector, escapeTextForSelector(text))
}

func escapeTextForSelector(raw string) string {
	return strings.ReplaceAll(raw, `"`, `\"`)
}

func workspaceIDFromChatURL(t *testing.T, chatURL string) string {
	t.Helper()

	matches := regexp.MustCompile(`/workspaces/([^/]+)/chats/`).FindStringSubmatch(chatURL)
	require.Len(t, matches, 2, "chat URL must contain workspace ID")

	return matches[1]
}

func selectOptionByValue(t *testing.T, selectEl playwright.Locator, value string) {
	t.Helper()

	values := []string{value}
	_, err := selectEl.SelectOption(playwright.SelectOptionValues{
		Values: &values,
	})
	require.NoError(t, err)
}

func waitForLocatorCountAtLeast(
	t *testing.T,
	locator playwright.Locator,
	minCount int,
	timeout time.Duration,
	description string,
) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	lastCount := 0
	var lastErr error

	for time.Now().Before(deadline) {
		lastCount, lastErr = locator.Count()
		if lastErr == nil && lastCount >= minCount {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.NoError(t, lastErr)
	require.GreaterOrEqual(t, lastCount, minCount, description)
}

func waitForLocatorCountEqual(
	t *testing.T,
	locator playwright.Locator,
	expectedCount int,
	timeout time.Duration,
	description string,
) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	lastCount := -1
	var lastErr error

	for time.Now().Before(deadline) {
		lastCount, lastErr = locator.Count()
		if lastErr == nil && lastCount == expectedCount {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.NoError(t, lastErr)
	require.Equal(t, expectedCount, lastCount, description)
}

func assertLocatorCountEqual(t *testing.T, locator playwright.Locator, expectedCount int, description string) {
	t.Helper()

	count, err := locator.Count()
	require.NoError(t, err)
	require.Equal(t, expectedCount, count, description)
}

func waitForInputValue(
	t *testing.T,
	locator playwright.Locator,
	expected string,
	timeout time.Duration,
	description string,
) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	lastValue := ""
	var lastErr error

	for time.Now().Before(deadline) {
		lastValue, lastErr = locator.InputValue()
		if lastErr == nil && lastValue == expected {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.NoError(t, lastErr)
	require.Equal(t, expected, lastValue, description)
}

func waitForTextContains(
	t *testing.T,
	locator playwright.Locator,
	expected string,
	timeout time.Duration,
	description string,
) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	lastText := ""
	var lastErr error

	for time.Now().Before(deadline) {
		lastText, lastErr = locator.TextContent()
		if lastErr == nil && strings.Contains(lastText, expected) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.NoError(t, lastErr)
	require.Contains(t, lastText, expected, description)
}

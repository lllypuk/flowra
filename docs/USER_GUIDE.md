# Flowra User Guide

Welcome to Flowra! This guide will help you get started with team collaboration and task management.

## Getting Started

### 1. Login

Flowra uses Single Sign-On (SSO) for authentication:

1. Go to the login page
2. Click **Sign in with SSO**
3. Enter your organization credentials in Keycloak
4. You'll be redirected to your workspaces

### 2. Create or Join a Workspace

A workspace is a shared space for your team:

- Click **+ New Workspace** to create one
- Enter a name and optional description
- Invite team members by their email

### 3. Start Chatting

Workspaces contain chat channels for team communication:

- Click **+ New Chat** to create a chat channel
- Select participants
- Start messaging!

## Features

### Chat

Flowra chat supports:

- **Markdown formatting** - Use `**bold**`, `*italic*`, and `` `code` ``
- **Tags** - Type `#` to see available commands:
  - `#createTask` - Create a new task from your message
  - `#bug` - Mark as a bug report
  - `#epic` - Create an epic
- **Mentions** - Use `@username` to notify team members
- **Real-time updates** - Messages appear instantly for all participants

**Keyboard Shortcuts:**
- `Ctrl+Enter` or `Cmd+Enter` - Send message
- `Escape` - Close any open dialogs

### Kanban Board

View and manage tasks on the visual board:

- **Columns:** TODO → In Progress → Review → Done
- **Drag and drop** cards between columns to change status
- **Click a card** to see full task details
- **Filter** by type, assignee, or priority

**Card Information:**
- Task title
- Type indicator (Task, Bug, Epic)
- Priority border (Critical=red, High=orange, Medium=blue, Low=green)
- Assignee avatar
- Due date (highlighted if overdue)

### Task Details

Click any task card to open the sidebar:

- **Edit title** - Click the title to edit inline
- **Change status** - Use the dropdown
- **Set priority** - Low, Medium, High, Critical
- **Assign** - Select team member from dropdown
- **Due date** - Pick a date from the calendar
- **Description** - Add detailed information
- **Activity** - View the full history of changes

### Notifications

Stay informed about important updates:

- **Bell icon** shows unread count
- Click to see recent notifications
- Notification types:
  - Mentions in chat
  - Task assignments
  - Status changes
  - Comments on your tasks
- Click a notification to go directly to the relevant item

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+K` / `Cmd+K` | Quick search |
| `Ctrl+Enter` / `Cmd+Enter` | Submit form / Send message |
| `Escape` | Close modal or dropdown |
| `?` | Show keyboard shortcuts help |
| `Tab` | Navigate between elements |

## Accessibility

Flowra is designed to be accessible:

- **Keyboard navigation** - All features are accessible via keyboard
- **Screen reader support** - Proper ARIA labels throughout
- **Skip links** - Press Tab on any page to access skip links
- **High contrast** - Respects system high contrast settings
- **Reduced motion** - Animations are reduced when system preference is set

## Mobile Access

Flowra works on mobile devices:

- **Responsive layout** - Adapts to screen size
- **Touch-friendly** - Large touch targets (44px minimum)
- **Hamburger menu** - Mobile navigation menu
- **Swipe gestures** - Horizontal scroll for kanban board

## Tips & Best Practices

1. **Use tags consistently** - Agree on tag usage with your team
2. **Set due dates** - Keep track of deadlines
3. **Assign tasks** - Make ownership clear
4. **Check notifications** - Stay updated on team activity
5. **Use keyboard shortcuts** - Work faster with shortcuts

## Getting Help

If you encounter any issues:

1. Check this user guide
2. Press `?` for keyboard shortcuts
3. Contact your workspace administrator
4. Report issues at https://github.com/anthropics/claude-code/issues

## Browser Support

Flowra works best on:

| Browser | Version |
|---------|---------|
| Chrome | 90+ |
| Firefox | 88+ |
| Safari | 14+ |
| Edge | 90+ |
| Mobile Chrome | Latest |
| Mobile Safari | iOS 14+ |

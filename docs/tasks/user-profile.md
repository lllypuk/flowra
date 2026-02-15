# User Profile & Settings Page

**Priority:** 1 (Core Gap)
**Status:** Complete

## Context

The backend has `GET /api/v1/users/me`, `PUT /api/v1/users/me`, and `GET /api/v1/users/:id` endpoints, but there is no frontend page for viewing or editing the user profile. The navbar has a user menu dropdown, but "Settings" currently has no dedicated page.

## Available API

- `GET /api/v1/users/me` — Returns current user profile (display name, email, avatar, username)
- `PUT /api/v1/users/me` — Updates display name, email, avatar
- `GET /api/v1/users/:id` — View another user's profile

## Deliverables

### User Settings Page (`/settings` or `/profile`)
- [x] Create `web/templates/user/settings.html` template
- [x] Add route handler for GET `/settings`
- [x] Display current profile info (avatar, display name, email, username)
- [x] Edit display name via inline edit or form (HTMX PUT to API)
- [x] Edit email via inline edit or form
- [x] Avatar display (placeholder with initials if no avatar URL)
- [x] Wire "Settings" link in navbar dropdown to this page

### User Profile View (`/users/:id`)
- [x] Create `web/templates/user/profile.html` template — read-only view of another user
- [x] Add route handler for GET `/users/:id`
- [x] Show display name, username, avatar
- [x] Link to this page from member lists, chat participant lists, assignee names

### Integration Points
- [x] Navbar user menu: link to settings page (already existed)
- [x] Workspace member rows: link username to profile
- [x] Chat message headers: clickable username
- [x] Task sidebar assignee: uses dropdown (no change needed)

## Technical Notes

- Follow existing HTMX partial pattern (e.g., workspace settings page)
- Use `base.html` layout
- Pico CSS form elements for inputs
- Flash messages for save success/error
- No file upload for avatar yet (just URL field or initials placeholder)

## Implementation Summary

Created two new pages:
1. **User Settings** (`/settings`) - Allows current user to view and edit their profile
2. **User Profile** (`/users/:id`) - Read-only view of any user's profile

Added clickable username links in:
- Workspace member rows (`@username` → `/users/:id`)
- Chat message headers (`@username` → `/users/:id`)

Created new template functions:
- `avatarInitials` - Extracts initials from UserResponse object for avatar display

All changes follow existing patterns and integrate seamlessly with the HTMX-based frontend.

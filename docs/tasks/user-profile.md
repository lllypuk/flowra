# User Profile & Settings Page

**Priority:** 1 (Core Gap)
**Status:** Pending

## Context

The backend has `GET /api/v1/users/me`, `PUT /api/v1/users/me`, and `GET /api/v1/users/:id` endpoints, but there is no frontend page for viewing or editing the user profile. The navbar has a user menu dropdown, but "Settings" currently has no dedicated page.

## Available API

- `GET /api/v1/users/me` — Returns current user profile (display name, email, avatar, username)
- `PUT /api/v1/users/me` — Updates display name, email, avatar
- `GET /api/v1/users/:id` — View another user's profile

## Deliverables

### User Settings Page (`/settings` or `/profile`)
- [ ] Create `web/templates/user/settings.html` template
- [ ] Add route handler for GET `/settings`
- [ ] Display current profile info (avatar, display name, email, username)
- [ ] Edit display name via inline edit or form (HTMX PUT to API)
- [ ] Edit email via inline edit or form
- [ ] Avatar display (placeholder with initials if no avatar URL)
- [ ] Wire "Settings" link in navbar dropdown to this page

### User Profile View (`/users/:id`)
- [ ] Create `web/templates/user/profile.html` template — read-only view of another user
- [ ] Add route handler for GET `/users/:id`
- [ ] Show display name, username, avatar
- [ ] Link to this page from member lists, chat participant lists, assignee names

### Integration Points
- [ ] Navbar user menu: link to settings page
- [ ] Workspace member rows: link username to profile
- [ ] Chat message headers: clickable username
- [ ] Task sidebar assignee: clickable username

## Technical Notes

- Follow existing HTMX partial pattern (e.g., workspace settings page)
- Use `base.html` layout
- Pico CSS form elements for inputs
- Flash messages for save success/error
- No file upload for avatar yet (just URL field or initials placeholder)

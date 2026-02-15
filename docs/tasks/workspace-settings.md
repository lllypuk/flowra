# Workspace Settings & Member Management

**Priority:** 2 (Feature Completeness)
**Status:** Pending

## Context

Workspace settings page exists with name/description editing and a danger zone (delete). Member management page shows members table with role badges. However, several admin operations are incomplete in the UI.

## Available API

- `PUT /api/v1/workspaces/:id` — Update name, description
- `DELETE /api/v1/workspaces/:id` — Delete workspace (owner only)
- `POST /api/v1/workspaces/:id/members` — Add member
- `DELETE /api/v1/workspaces/:id/members/:user_id` — Remove member
- `PUT /api/v1/workspaces/:id/members/:user_id/role` — Update member role

## Deliverables

### Member Invitation Flow
- [ ] "Invite Member" button opens modal with user search
- [ ] Search users by username or email (may need a user search API endpoint)
- [ ] Select user and role, submit invitation
- [ ] Flash message on success, update member list via HTMX swap
- [ ] Handle errors (user already member, user not found)

### Member Role Management
- [ ] Role dropdown on each member row (Owner, Admin, Member)
- [ ] Change role via `PUT /api/v1/workspaces/:id/members/:user_id/role`
- [ ] Only show role editing to workspace owner
- [ ] Confirmation for role changes
- [ ] HTMX swap to update row in place

### Member Removal
- [ ] "Remove" button per member row (admin+ only)
- [ ] Confirmation dialog before removal
- [ ] `DELETE /api/v1/workspaces/:id/members/:user_id`
- [ ] Remove row from table via HTMX swap
- [ ] "Leave workspace" option for self-removal

### Settings Page Polish
- [ ] Workspace avatar/icon selection (optional)
- [ ] Show workspace creation date, owner info
- [ ] Transfer ownership option (change another member to owner)
- [ ] Confirmation dialogs for destructive actions

## Technical Notes

- Existing partials: `WorkspaceMembersPartial`, `WorkspaceMembersOptionsPartial`, `UpdateMemberRolePartial`, `WorkspaceInviteForm`
- These HTMX partials are already registered in routes — may just need template implementation
- Follow Pico CSS table patterns for member list
- Use `data-confirm` pattern for destructive actions

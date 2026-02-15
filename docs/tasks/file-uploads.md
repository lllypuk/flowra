# File Uploads & Attachments

**Priority:** 3 (Quality of Life)
**Status:** Pending

## Context

No file upload functionality exists in the frontend or backend. Messages and tasks currently support text only. This is a significant feature that requires both backend and frontend work.

## Scope Assessment

**Backend work needed first:**
- File upload API endpoint (multipart form)
- File storage (local filesystem or S3-compatible)
- File metadata in message/task models
- File serving endpoint with auth check

**This task covers frontend only — backend must be implemented first.**

## Deliverables (Frontend)

### Message Attachments
- [ ] File attachment button in message form (paperclip icon)
- [ ] File picker dialog (accept common types: images, docs, PDFs)
- [ ] Upload progress indicator
- [ ] Preview attached files before sending
- [ ] Remove attachment before sending
- [ ] Display attachments in sent messages (image preview, file icon + name for others)
- [ ] Click to download/open attachment

### Task Attachments
- [ ] File upload area in task sidebar
- [ ] Drag-and-drop support
- [ ] List of attached files with download links
- [ ] Remove attachment (with confirmation)

### Image Handling
- [ ] Inline image preview in messages (thumbnail with click-to-expand)
- [ ] Lightbox for full-size image viewing
- [ ] Image paste from clipboard

## Prerequisites

- [ ] Backend file upload API must be implemented first
- [ ] File storage configuration must be decided
- [ ] Max file size limits defined

## Technical Notes

- Use `FormData` API for multipart uploads
- Consider chunk upload for large files
- Image thumbnails can be generated server-side or CSS-resized
- Drag-and-drop uses standard HTML5 DnD events

# File Uploads & Attachments

**Priority:** 3 (Quality of Life)
**Status:** Complete ✅

## Context

File upload functionality for messages and tasks. Includes backend file storage, upload/download API, and frontend UI with image preview, lightbox, drag-and-drop, and clipboard paste.

## Scope Assessment

**Backend work needed first:**
- File upload API endpoint (multipart form)
- File storage (local filesystem or S3-compatible)
- File metadata in message/task models
- File serving endpoint with auth check

**This task covers frontend only — backend must be implemented first.**

## Deliverables (Frontend)

### Message Attachments
- [x] File attachment button in message form (paperclip icon)
- [x] File picker dialog (accept common types: images, docs, PDFs)
- [x] Upload progress indicator
- [x] Preview attached files before sending
- [x] Remove attachment before sending
- [x] Display attachments in sent messages (image preview, file icon + name for others)
- [x] Click to download/open attachment

### Task Attachments
- [x] File upload area in task sidebar
- [x] Drag-and-drop support
- [x] List of attached files with download links
- [x] Remove attachment (with confirmation)

### Image Handling
- [x] Inline image preview in messages (thumbnail with click-to-expand)
- [x] Lightbox for full-size image viewing
- [x] Image paste from clipboard

## Prerequisites

- [x] Backend file upload API must be implemented first
- [x] File storage configuration must be decided
- [x] Max file size limits defined

## Technical Notes

- Use `FormData` API for multipart uploads
- Consider chunk upload for large files
- Image thumbnails can be generated server-side or CSS-resized
- Drag-and-drop uses standard HTML5 DnD events

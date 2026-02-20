# Fix: XSS Vulnerability in Message Attachment Template

## Status: Pending

## Severity: High

## Problem

In `web/templates/components/message.html:46`, attachment file names and URLs are interpolated
directly into an inline `onclick` JavaScript attribute:

```html
<a href="{{.URL}}" target="_blank" class="lightbox-trigger"
   onclick="openLightbox(event, '{{.URL}}', '{{.FileName}}')">
```

Go's `html/template` package auto-escapes values in HTML attribute context, but the escaping
rules for JavaScript string literals inside `onclick` attributes are insufficient. A file name
containing a single quote (`'`) breaks out of the JavaScript string context.

### Attack Scenario

1. Attacker uploads a file with name: `photo'); alert(document.cookie); ('`
2. Backend stores this in `AttachmentViewData.FileName` without sanitization
   (`chat_template_handler.go:1034` — directly uses `a.FileName()`)
3. Template renders:
   ```html
   onclick="openLightbox(event, '/api/v1/files/uuid/photo'); alert(document.cookie); ('', 'photo'); alert(document.cookie); ('')"
   ```
4. When any user in the chat clicks the attachment thumbnail, attacker's JavaScript executes
   in their browser context — session cookies, CSRF tokens, and DOM content are exposed

### Why Go Template Escaping Doesn't Help

Go's `html/template` recognizes the `onclick` attribute as a JS context and applies JS escaping.
However, the escaping is designed for simple values, not for preventing injection in complex
concatenated JS strings. The template engine may not correctly handle all edge cases when
values contain both quotes and parentheses.

The fundamental issue is using inline event handlers with interpolated data — this is an
anti-pattern regardless of template engine escaping quality.

## Files to Modify

### 1. `web/templates/components/message.html`

Replace the inline `onclick` with `data-*` attributes and use `addEventListener` from JavaScript:

**Before (line 46):**
```html
<a href="{{.URL}}" target="_blank" class="lightbox-trigger"
   onclick="openLightbox(event, '{{.URL}}', '{{.FileName}}')">
    <img src="{{.URL}}" alt="{{.FileName}}" loading="lazy">
</a>
```

**After:**
```html
<a href="{{.URL}}" target="_blank" class="lightbox-trigger"
   data-lightbox-url="{{.URL}}" data-lightbox-name="{{.FileName}}">
    <img src="{{.URL}}" alt="{{.FileName}}" loading="lazy">
</a>
```

The `data-*` attributes are properly HTML-escaped by Go's template engine (HTML attribute
context escaping is well-defined and reliable). No JavaScript string escaping is needed.

### 2. `web/static/js/app.js`

Replace the `window.openLightbox` global function call with a delegated event listener.
Find the `openLightbox` function (around line 1061) and add a delegated click handler:

**Add delegated listener** (near the existing `openLightbox` definition):

```javascript
// Delegated lightbox handler — replaces inline onclick
document.addEventListener('click', function(event) {
    var trigger = event.target.closest('.lightbox-trigger[data-lightbox-url]');
    if (!trigger) return;
    event.preventDefault();

    var url = trigger.getAttribute('data-lightbox-url');
    var fileName = trigger.getAttribute('data-lightbox-name');
    openLightbox(event, url, fileName);
});
```

Keep the existing `window.openLightbox` function unchanged — it still builds the lightbox
overlay. Only the invocation mechanism changes from inline `onclick` to delegated listener.

### 3. `internal/handler/http/file_handler.go` (defense-in-depth)

Sanitize the file name on upload to strip characters that are dangerous in any context.
Add sanitization in `Upload` before storing:

```go
// Sanitize filename: keep only the base name, replace control characters
safeName := filepath.Base(file.Filename)
safeName = strings.Map(func(r rune) rune {
    if r < 32 || r == '\'' || r == '"' || r == '`' || r == '<' || r == '>' {
        return '_'
    }
    return r
}, safeName)
if safeName == "" || safeName == "." {
    safeName = "unnamed"
}
```

Use `safeName` instead of `file.Filename` in `h.storage.Save()` and the response.

### 4. `web/templates/task/sidebar.html` (if applicable)

Check if the task sidebar also renders attachments with inline `onclick`. If so, apply the
same `data-*` attribute pattern.

## Checklist

- [ ] Replace `onclick="openLightbox(...)"` with `data-lightbox-url` and `data-lightbox-name`
      attributes in `message.html`
- [ ] Add delegated click handler in `app.js` for `.lightbox-trigger[data-lightbox-url]`
- [ ] Sanitize file names on upload in `file_handler.go` (strip quotes, angle brackets, control chars)
- [ ] Check `task/sidebar.html` for same pattern and fix if present
- [ ] Verify lightbox still works after changes (manual test: upload image, click thumbnail)
- [ ] Run `golangci-lint run`

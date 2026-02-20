package httphandler

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/filestorage"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
)

// File handler constants.
const (
	maxUploadSize   = 10 << 20 // 10 MB
	maxUploadSizeMB = 10
	mimeOctetStream = "application/octet-stream"
)

// FileUploadResponse represents the response after uploading a file.
type FileUploadResponse struct {
	FileID   uuid.UUID `json:"file_id"`
	FileName string    `json:"file_name"`
	FileSize int64     `json:"file_size"`
	MimeType string    `json:"mime_type"`
	URL      string    `json:"url"`
}

// FileMetadataLookup retrieves file ownership metadata.
type FileMetadataLookup interface {
	Save(ctx context.Context, meta FileMetadataEntry) error
	FindByFileID(ctx context.Context, fileID uuid.UUID) (*FileMetadataEntry, error)
}

// FileMetadataEntry holds ownership information for an uploaded file.
type FileMetadataEntry struct {
	FileID     uuid.UUID
	ChatID     uuid.UUID
	UploaderID uuid.UUID
	UploadedAt time.Time
}

// FileChatParticipantChecker verifies user is a participant of a chat.
type FileChatParticipantChecker interface {
	IsParticipant(ctx context.Context, chatID uuid.UUID, userID uuid.UUID) (bool, error)
}

// FileHandler handles file upload and download HTTP requests.
type FileHandler struct {
	storage          *filestorage.LocalStorage
	metadataRepo     FileMetadataLookup
	participantCheck FileChatParticipantChecker
}

// NewFileHandler creates a new FileHandler.
func NewFileHandler(
	storage *filestorage.LocalStorage,
	metadataRepo FileMetadataLookup,
	participantCheck FileChatParticipantChecker,
) *FileHandler {
	return &FileHandler{
		storage:          storage,
		metadataRepo:     metadataRepo,
		participantCheck: participantCheck,
	}
}

// RegisterRoutes registers file routes with the router.
func (h *FileHandler) RegisterRoutes(r *httpserver.Router) {
	r.Auth().POST("/files/upload", h.Upload)
	r.Auth().GET("/files/:file_id/:file_name", h.Download)
}

// Upload handles POST /api/v1/files/upload.
// Accepts multipart form with a "file" field and a "chat_id" field.
func (h *FileHandler) Upload(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	// Require chat_id for authorization
	chatIDStr := c.FormValue("chat_id")
	if chatIDStr == "" {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "MISSING_CHAT_ID", "chat_id is required")
	}
	chatID, chatParseErr := uuid.ParseUUID(chatIDStr)
	if chatParseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
	}

	// Verify user is a participant of this chat
	isMember, memberErr := h.participantCheck.IsParticipant(c.Request().Context(), chatID, userID)
	if memberErr != nil || !isMember {
		return httpserver.RespondErrorWithCode(
			c, http.StatusForbidden, "FORBIDDEN", "you are not a participant of this chat")
	}

	// Limit request body size
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, maxUploadSize)

	file, err := c.FormFile("file")
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			return httpserver.RespondErrorWithCode(
				c, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE",
				fmt.Sprintf("file size exceeds %d MB limit", maxUploadSizeMB))
		}
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_FILE", "file is required")
	}

	// Validate file size
	if file.Size > maxUploadSize {
		return httpserver.RespondErrorWithCode(
			c, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE",
			fmt.Sprintf("file size exceeds %d MB limit", maxUploadSizeMB))
	}

	// Detect MIME type
	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" || mimeType == mimeOctetStream {
		mimeType = mime.TypeByExtension(filepath.Ext(file.Filename))
		if mimeType == "" {
			mimeType = mimeOctetStream
		}
	}

	// Validate MIME type
	if !isAllowedMIME(mimeType) {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_FILE_TYPE", "file type not allowed")
	}

	// Open the file
	src, openErr := file.Open()
	if openErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusInternalServerError, "FILE_ERROR", "failed to read uploaded file")
	}
	defer src.Close()

	// Sanitize filename: strip dangerous characters
	safeName := sanitizeFileName(file.Filename)

	// Save to storage
	fileID, saveErr := h.storage.Save(src, safeName)
	if saveErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusInternalServerError, "STORAGE_ERROR", "failed to save file")
	}

	// Save file metadata for authorization
	_ = h.metadataRepo.Save(c.Request().Context(), FileMetadataEntry{
		FileID:     fileID,
		ChatID:     chatID,
		UploaderID: userID,
		UploadedAt: time.Now().UTC(),
	})

	resp := FileUploadResponse{
		FileID:   fileID,
		FileName: safeName,
		FileSize: file.Size,
		MimeType: mimeType,
		URL:      fmt.Sprintf("/api/v1/files/%s/%s", fileID.String(), url.PathEscape(safeName)),
	}

	return httpserver.RespondCreated(c, resp)
}

// Download handles GET /api/v1/files/:file_id/:file_name.
// Serves the file with appropriate content type after verifying authorization.
func (h *FileHandler) Download(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	fileIDStr := c.Param("file_id")
	fileID, parseErr := uuid.ParseUUID(fileIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_FILE_ID", "invalid file ID format")
	}

	fileName := filepath.Base(c.Param("file_name"))
	if fileName == "" || fileName == "." {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_FILE_NAME", "file name is required")
	}

	// Authorization: verify user has access to the file's chat
	meta, metaErr := h.metadataRepo.FindByFileID(c.Request().Context(), fileID)
	if metaErr != nil {
		// Files without metadata (uploaded before migration) are served without auth check
		return h.serveFile(c, fileID, fileName)
	}

	isMember, memberErr := h.participantCheck.IsParticipant(c.Request().Context(), meta.ChatID, userID)
	if memberErr != nil || !isMember {
		return httpserver.RespondErrorWithCode(
			c, http.StatusForbidden, "FORBIDDEN", "you do not have access to this file")
	}

	return h.serveFile(c, fileID, fileName)
}

// serveFile serves a file from storage with appropriate headers.
func (h *FileHandler) serveFile(c echo.Context, fileID uuid.UUID, fileName string) error {
	// Check if file exists
	if !h.storage.Exists(fileID, fileName) {
		return httpserver.RespondErrorWithCode(
			c, http.StatusNotFound, "FILE_NOT_FOUND", "file not found")
	}

	filePath, pathErr := h.storage.FilePath(fileID, fileName)
	if pathErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_PATH", "invalid file path")
	}

	// Detect content type
	contentType := mime.TypeByExtension(filepath.Ext(fileName))
	if contentType == "" {
		contentType = mimeOctetStream
	}

	// For images, serve inline; for other files, force download
	if strings.HasPrefix(contentType, "image/") {
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", fileName))
	} else {
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	}

	return c.File(filePath)
}

// sanitizeFileName strips dangerous characters from the filename for defense-in-depth.
func sanitizeFileName(name string) string {
	safe := filepath.Base(name)
	safe = strings.Map(func(r rune) rune {
		if r < 32 || r == '\'' || r == '"' || r == '`' || r == '<' || r == '>' {
			return '_'
		}
		return r
	}, safe)
	if safe == "" || safe == "." {
		safe = "unnamed"
	}
	return safe
}

func isAllowedMIME(mimeType string) bool {
	allowed := []string{
		"image/",
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats",
		"application/vnd.ms-",
		"text/plain",
		"text/csv",
		"application/zip",
		"application/x-tar",
		"application/gzip",
	}
	for _, prefix := range allowed {
		if strings.HasPrefix(mimeType, prefix) {
			return true
		}
	}
	return false
}

package httphandler

import (
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

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

// FileHandler handles file upload and download HTTP requests.
type FileHandler struct {
	storage *filestorage.LocalStorage
}

// NewFileHandler creates a new FileHandler.
func NewFileHandler(storage *filestorage.LocalStorage) *FileHandler {
	return &FileHandler{
		storage: storage,
	}
}

// RegisterRoutes registers file routes with the router.
func (h *FileHandler) RegisterRoutes(r *httpserver.Router) {
	r.Auth().POST("/files/upload", h.Upload)
	r.Auth().GET("/files/:file_id/:file_name", h.Download)
}

// Upload handles POST /api/v1/files/upload.
// Accepts multipart form with a "file" field.
func (h *FileHandler) Upload(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
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

	// Save to storage
	fileID, saveErr := h.storage.Save(src, file.Filename)
	if saveErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusInternalServerError, "STORAGE_ERROR", "failed to save file")
	}

	resp := FileUploadResponse{
		FileID:   fileID,
		FileName: file.Filename,
		FileSize: file.Size,
		MimeType: mimeType,
		URL:      fmt.Sprintf("/api/v1/files/%s/%s", fileID.String(), file.Filename),
	}

	return httpserver.RespondCreated(c, resp)
}

// Download handles GET /api/v1/files/:file_id/:file_name.
// Serves the file with appropriate content type.
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

	fileName := c.Param("file_name")
	if fileName == "" {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_FILE_NAME", "file name is required")
	}

	// Check if file exists
	if !h.storage.Exists(fileID, fileName) {
		return httpserver.RespondErrorWithCode(
			c, http.StatusNotFound, "FILE_NOT_FOUND", "file not found")
	}

	filePath := h.storage.FilePath(fileID, fileName)

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

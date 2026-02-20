package httphandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/infrastructure/filestorage"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFileMetadataRepo implements httphandler.FileMetadataLookup for tests.
type mockFileMetadataRepo struct {
	entries map[string]*httphandler.FileMetadataEntry
}

func newMockFileMetadataRepo() *mockFileMetadataRepo {
	return &mockFileMetadataRepo{entries: make(map[string]*httphandler.FileMetadataEntry)}
}

func (m *mockFileMetadataRepo) Save(_ context.Context, meta httphandler.FileMetadataEntry) error {
	m.entries[meta.FileID.String()] = &meta
	return nil
}

func (m *mockFileMetadataRepo) FindByFileID(
	_ context.Context,
	fileID uuid.UUID,
) (*httphandler.FileMetadataEntry, error) {
	entry, ok := m.entries[fileID.String()]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return entry, nil
}

// mockParticipantChecker implements httphandler.FileChatParticipantChecker for tests.
type mockParticipantChecker struct {
	participants map[string]map[string]bool // chatID -> userID -> isMember
}

func newMockParticipantChecker() *mockParticipantChecker {
	return &mockParticipantChecker{participants: make(map[string]map[string]bool)}
}

func (m *mockParticipantChecker) AddParticipant(chatID, userID uuid.UUID) {
	key := chatID.String()
	if m.participants[key] == nil {
		m.participants[key] = make(map[string]bool)
	}
	m.participants[key][userID.String()] = true
}

func (m *mockParticipantChecker) IsParticipant(_ context.Context, chatID, userID uuid.UUID) (bool, error) {
	users, ok := m.participants[chatID.String()]
	if !ok {
		return false, nil
	}
	return users[userID.String()], nil
}

func newTestFileHandler(t *testing.T) (
	*httphandler.FileHandler,
	*filestorage.LocalStorage,
	*mockFileMetadataRepo,
	*mockParticipantChecker,
) {
	t.Helper()
	dir := t.TempDir()
	storage, err := filestorage.NewLocalStorage(dir)
	require.NoError(t, err)
	metadataRepo := newMockFileMetadataRepo()
	participantChecker := newMockParticipantChecker()
	handler := httphandler.NewFileHandler(storage, metadataRepo, participantChecker)
	return handler, storage, metadataRepo, participantChecker
}

func createMultipartFile(t *testing.T, fileName, content string) (*bytes.Buffer, string) {
	t.Helper()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", fileName)
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)
	return body, writer.FormDataContentType()
}

func createMultipartFileWithChatID(t *testing.T, fileName, content string, chatID uuid.UUID) (*bytes.Buffer, string) {
	t.Helper()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("chat_id", chatID.String())

	part, err := writer.CreateFormFile("file", fileName)
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)
	return body, writer.FormDataContentType()
}

func TestFileHandler_Upload(t *testing.T) {
	chatID := uuid.NewUUID()
	userID := uuid.UUID("user-123")

	t.Run("successful upload", func(t *testing.T) {
		handler, _, _, participantChecker := newTestFileHandler(t)
		participantChecker.AddParticipant(chatID, userID)
		e := echo.New()

		body, contentType := createMultipartFileWithChatID(t, "test.txt", "hello world", chatID)
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/files/upload", body)
		req.Header.Set(echo.HeaderContentType, contentType)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setupAuthContext(c, userID)

		err := handler.Upload(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Data)
	})

	t.Run("returns file metadata in response", func(t *testing.T) {
		handler, _, _, participantChecker := newTestFileHandler(t)
		participantChecker.AddParticipant(chatID, userID)
		e := echo.New()

		body, contentType := createMultipartFileWithChatID(t, "document.pdf", "pdf content", chatID)
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/files/upload", body)
		req.Header.Set(echo.HeaderContentType, contentType)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setupAuthContext(c, userID)

		err := handler.Upload(c)
		require.NoError(t, err)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		data, ok := resp.Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "document.pdf", data["file_name"])
		assert.NotEmpty(t, data["file_id"])
		assert.NotEmpty(t, data["url"])
	})

	t.Run("unauthorized without user context", func(t *testing.T) {
		handler, _, _, _ := newTestFileHandler(t)
		e := echo.New()

		body, contentType := createMultipartFile(t, "test.txt", "content")
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/files/upload", body)
		req.Header.Set(echo.HeaderContentType, contentType)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Upload(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})

	t.Run("rejects missing chat_id", func(t *testing.T) {
		handler, _, _, _ := newTestFileHandler(t)
		e := echo.New()

		body, contentType := createMultipartFile(t, "test.txt", "content")
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/files/upload", body)
		req.Header.Set(echo.HeaderContentType, contentType)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setupAuthContext(c, userID)

		err := handler.Upload(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "MISSING_CHAT_ID", resp.Error.Code)
	})

	t.Run("rejects non-participant", func(t *testing.T) {
		handler, _, _, _ := newTestFileHandler(t)
		e := echo.New()

		body, contentType := createMultipartFileWithChatID(t, "test.txt", "content", chatID)
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/files/upload", body)
		req.Header.Set(echo.HeaderContentType, contentType)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setupAuthContext(c, userID)

		err := handler.Upload(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusForbidden, rec.Code)
	})

	t.Run("rejects missing file field", func(t *testing.T) {
		handler, _, _, participantChecker := newTestFileHandler(t)
		participantChecker.AddParticipant(chatID, userID)
		e := echo.New()

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("chat_id", chatID.String())
		_ = writer.Close()

		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/files/upload", body)
		req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setupAuthContext(c, userID)

		err := handler.Upload(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "INVALID_FILE", resp.Error.Code)
	})

	t.Run("saves file to storage", func(t *testing.T) {
		handler, storage, _, participantChecker := newTestFileHandler(t)
		participantChecker.AddParticipant(chatID, userID)
		e := echo.New()

		body, contentType := createMultipartFileWithChatID(t, "saved.txt", "stored content", chatID)
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/files/upload", body)
		req.Header.Set(echo.HeaderContentType, contentType)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setupAuthContext(c, userID)

		err := handler.Upload(c)
		require.NoError(t, err)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp.Data.(map[string]any)
		fileID := uuid.UUID(data["file_id"].(string))
		assert.True(t, storage.Exists(fileID, "saved.txt"))
	})
}

func TestFileHandler_Download(t *testing.T) {
	chatID := uuid.NewUUID()
	userID := uuid.UUID("user-123")

	t.Run("downloads existing file without metadata (pre-migration)", func(t *testing.T) {
		handler, storage, _, _ := newTestFileHandler(t)
		e := echo.New()

		fileID, err := storage.Save(strings.NewReader("download me"), "readme.txt")
		require.NoError(t, err)

		req := httptest.NewRequest(stdhttp.MethodGet,
			fmt.Sprintf("/api/v1/files/%s/readme.txt", fileID.String()), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("file_id", "file_name")
		c.SetParamValues(fileID.String(), "readme.txt")
		setupAuthContext(c, userID)

		err = handler.Download(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
		assert.Equal(t, "download me", rec.Body.String())
	})

	t.Run("downloads file with metadata when participant", func(t *testing.T) {
		handler, storage, metadataRepo, participantChecker := newTestFileHandler(t)
		participantChecker.AddParticipant(chatID, userID)
		e := echo.New()

		fileID, err := storage.Save(strings.NewReader("authorized content"), "secret.txt")
		require.NoError(t, err)

		_ = metadataRepo.Save(context.Background(), httphandler.FileMetadataEntry{
			FileID: fileID, ChatID: chatID, UploaderID: userID, UploadedAt: time.Now(),
		})

		req := httptest.NewRequest(stdhttp.MethodGet,
			fmt.Sprintf("/api/v1/files/%s/secret.txt", fileID.String()), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("file_id", "file_name")
		c.SetParamValues(fileID.String(), "secret.txt")
		setupAuthContext(c, userID)

		err = handler.Download(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
		assert.Equal(t, "authorized content", rec.Body.String())
	})

	t.Run("rejects download when not participant", func(t *testing.T) {
		handler, storage, metadataRepo, _ := newTestFileHandler(t)
		e := echo.New()

		fileID, err := storage.Save(strings.NewReader("private"), "private.txt")
		require.NoError(t, err)

		_ = metadataRepo.Save(context.Background(), httphandler.FileMetadataEntry{
			FileID: fileID, ChatID: chatID, UploaderID: uuid.UUID("other-user"), UploadedAt: time.Now(),
		})

		req := httptest.NewRequest(stdhttp.MethodGet,
			fmt.Sprintf("/api/v1/files/%s/private.txt", fileID.String()), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("file_id", "file_name")
		c.SetParamValues(fileID.String(), "private.txt")
		setupAuthContext(c, userID)

		err = handler.Download(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusForbidden, rec.Code)
	})

	t.Run("sets attachment disposition for non-image files", func(t *testing.T) {
		handler, storage, _, _ := newTestFileHandler(t)
		e := echo.New()

		fileID, err := storage.Save(strings.NewReader("pdf data"), "doc.pdf")
		require.NoError(t, err)

		req := httptest.NewRequest(stdhttp.MethodGet,
			fmt.Sprintf("/api/v1/files/%s/doc.pdf", fileID.String()), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("file_id", "file_name")
		c.SetParamValues(fileID.String(), "doc.pdf")
		setupAuthContext(c, userID)

		err = handler.Download(c)
		require.NoError(t, err)
		assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment")
	})

	t.Run("sets inline disposition for image files", func(t *testing.T) {
		handler, storage, _, _ := newTestFileHandler(t)
		e := echo.New()

		fileID, err := storage.Save(strings.NewReader("png data"), "photo.png")
		require.NoError(t, err)

		req := httptest.NewRequest(stdhttp.MethodGet,
			fmt.Sprintf("/api/v1/files/%s/photo.png", fileID.String()), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("file_id", "file_name")
		c.SetParamValues(fileID.String(), "photo.png")
		setupAuthContext(c, userID)

		err = handler.Download(c)
		require.NoError(t, err)
		assert.Contains(t, rec.Header().Get("Content-Disposition"), "inline")
	})

	t.Run("unauthorized without user context", func(t *testing.T) {
		handler, _, _, _ := newTestFileHandler(t)
		e := echo.New()

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/files/some-id/file.txt", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("file_id", "file_name")
		c.SetParamValues("some-id", "file.txt")

		err := handler.Download(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})

	t.Run("returns 400 for invalid file ID", func(t *testing.T) {
		handler, _, _, _ := newTestFileHandler(t)
		e := echo.New()

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/files/not-a-uuid/file.txt", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("file_id", "file_name")
		c.SetParamValues("not-a-uuid", "file.txt")
		setupAuthContext(c, userID)

		err := handler.Download(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("returns 404 for non-existing file", func(t *testing.T) {
		handler, _, _, _ := newTestFileHandler(t)
		e := echo.New()

		fakeID := uuid.NewUUID()
		req := httptest.NewRequest(stdhttp.MethodGet,
			fmt.Sprintf("/api/v1/files/%s/missing.txt", fakeID.String()), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("file_id", "file_name")
		c.SetParamValues(fakeID.String(), "missing.txt")
		setupAuthContext(c, userID)

		err := handler.Download(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})

	t.Run("returns 400 for empty file name", func(t *testing.T) {
		handler, _, _, _ := newTestFileHandler(t)
		e := echo.New()

		fakeID := uuid.NewUUID()
		req := httptest.NewRequest(stdhttp.MethodGet,
			fmt.Sprintf("/api/v1/files/%s/", fakeID.String()), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("file_id", "file_name")
		c.SetParamValues(fakeID.String(), "")
		setupAuthContext(c, userID)

		err := handler.Download(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestIsAllowedMIME(t *testing.T) {
	chatID := uuid.NewUUID()
	userID := uuid.UUID("user-123")

	t.Run("allows image types", func(t *testing.T) {
		handler, _, _, participantChecker := newTestFileHandler(t)
		participantChecker.AddParticipant(chatID, userID)
		e := echo.New()

		body, contentType := createMultipartFileWithChatID(t, "photo.jpg", "jpeg data", chatID)
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/files/upload", body)
		req.Header.Set(echo.HeaderContentType, contentType)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setupAuthContext(c, userID)

		err := handler.Upload(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)
	})

	t.Run("allows pdf", func(t *testing.T) {
		handler, _, _, participantChecker := newTestFileHandler(t)
		participantChecker.AddParticipant(chatID, userID)
		e := echo.New()

		body, contentType := createMultipartFileWithChatID(t, "doc.pdf", "pdf", chatID)
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/files/upload", body)
		req.Header.Set(echo.HeaderContentType, contentType)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setupAuthContext(c, userID)

		err := handler.Upload(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)
	})

	t.Run("allows text/plain", func(t *testing.T) {
		handler, _, _, participantChecker := newTestFileHandler(t)
		participantChecker.AddParticipant(chatID, userID)
		e := echo.New()

		body, contentType := createMultipartFileWithChatID(t, "notes.txt", "text", chatID)
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/files/upload", body)
		req.Header.Set(echo.HeaderContentType, contentType)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setupAuthContext(c, userID)

		err := handler.Upload(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)
	})
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"small bytes", 100, "100 B"},
		{"exactly 1 KB", 1024, "1.0 KB"},
		{"kilobytes", 2560, "2.5 KB"},
		{"exactly 1 MB", 1024 * 1024, "1.0 MB"},
		{"megabytes", 5 * 1024 * 1024, "5.0 MB"},
		{"exactly 1 GB", 1024 * 1024 * 1024, "1.0 GB"},
		{"gigabytes", 3 * 1024 * 1024 * 1024, "3.0 GB"},
		{"1023 bytes", 1023, "1023 B"},
		{"fractional MB", 1536 * 1024, "1.5 MB"},
	}

	// Use TemplateFuncs to get the formatFileSize function
	funcs := httphandler.TemplateFuncs()
	formatFn, ok := funcs["formatFileSize"]
	require.True(t, ok)

	fn := formatFn.(func(int64) string)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(tt.size)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFileUploadResponse_JSON(t *testing.T) {
	resp := httphandler.FileUploadResponse{
		FileID:   uuid.UUID("abc-123"),
		FileName: "test.txt",
		FileSize: 1024,
		MimeType: "text/plain",
		URL:      "/api/v1/files/abc-123/test.txt",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded map[string]any
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "abc-123", decoded["file_id"])
	assert.Equal(t, "test.txt", decoded["file_name"])
	assert.InDelta(t, float64(1024), decoded["file_size"], 0.1)
	assert.Equal(t, "text/plain", decoded["mime_type"])
	assert.Equal(t, "/api/v1/files/abc-123/test.txt", decoded["url"])
}

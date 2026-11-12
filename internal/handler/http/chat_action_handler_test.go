package httphandler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/middleware"
)

// fakeChatActions answers every action with err, or with version 7 of the chat.
type fakeChatActions struct {
	err      error
	replayed bool
	lastKey  string
	calls    int
}

func (f *fakeChatActions) result(chatID uuid.UUID, key string) (*appcore.ActionResult, error) {
	f.calls++
	f.lastKey = key
	if f.err != nil {
		return &appcore.ActionResult{Error: f.err.Error()}, f.err
	}
	return &appcore.ActionResult{ChatID: chatID, Version: 7, Replayed: f.replayed, Success: true}, nil
}

func (f *fakeChatActions) ChangeStatus(
	_ context.Context, chatID uuid.UUID, _ string, _ uuid.UUID, key string,
) (*appcore.ActionResult, error) {
	return f.result(chatID, key)
}

func (f *fakeChatActions) SetPriority(
	_ context.Context, chatID uuid.UUID, _ string, _ uuid.UUID, key string,
) (*appcore.ActionResult, error) {
	return f.result(chatID, key)
}

func (f *fakeChatActions) AssignUser(
	_ context.Context, chatID uuid.UUID, _ *uuid.UUID, _ uuid.UUID, key string,
) (*appcore.ActionResult, error) {
	return f.result(chatID, key)
}

func (f *fakeChatActions) SetDueDate(
	_ context.Context, chatID uuid.UUID, _ *time.Time, _ uuid.UUID, key string,
) (*appcore.ActionResult, error) {
	return f.result(chatID, key)
}

func (f *fakeChatActions) Close(_ context.Context, chatID, _ uuid.UUID, key string) (*appcore.ActionResult, error) {
	return f.result(chatID, key)
}

func (f *fakeChatActions) Reopen(_ context.Context, chatID, _ uuid.UUID, key string) (*appcore.ActionResult, error) {
	return f.result(chatID, key)
}

func (f *fakeChatActions) Rename(
	_ context.Context, chatID uuid.UUID, _ string, _ uuid.UUID, key string,
) (*appcore.ActionResult, error) {
	return f.result(chatID, key)
}

func postChatAction(
	t *testing.T,
	h func(echo.Context) error,
	chatID uuid.UUID,
	body, key string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(chatID.String())
	c.Set(string(middleware.ContextKeyUserID), uuid.NewUUID())
	require.NoError(t, h(c))
	return rec
}

func TestChatActionHandler_Outcome(t *testing.T) {
	chatID := uuid.NewUUID()

	tests := []struct {
		name       string
		err        error
		replayed   bool
		key        string
		wantStatus int
		wantCode   string
	}{
		{name: "applied", key: "k1", wantStatus: http.StatusOK},
		{name: "replayed", key: "k1", replayed: true, wantStatus: http.StatusOK},
		{name: "applied without key", wantStatus: http.StatusOK},
		{
			name: "conflict not applied", err: fmt.Errorf("%w: busy", errs.ErrConcurrentModification),
			wantStatus: http.StatusConflict, wantCode: "CONCURRENT_MODIFICATION",
		},
		{
			name: "unknown outcome", err: fmt.Errorf("save: %w", errs.ErrOutcomeUnknown),
			wantStatus: http.StatusServiceUnavailable, wantCode: "OUTCOME_UNKNOWN",
		},
		{
			name: "key reused", err: errs.ErrIdempotencyKeyReused,
			wantStatus: http.StatusUnprocessableEntity, wantCode: "IDEMPOTENCY_KEY_REUSED",
		},
		{
			name: "not a participant", err: fmt.Errorf("%w: not a chat participant", errs.ErrForbidden),
			wantStatus: http.StatusForbidden, wantCode: "FORBIDDEN",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeChatActions{err: tt.err, replayed: tt.replayed}
			h := httphandler.NewChatActionHandler(svc)

			rec := postChatAction(t, h.SetDueDate, chatID, `{"due_date":"2026-12-01"}`, tt.key)

			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			assert.Equal(t, tt.key, svc.lastKey)
			var body struct {
				Success bool `json:"success"`
				Data    struct {
					ChatID   string `json:"chat_id"`
					Version  int    `json:"version"`
					Replayed bool   `json:"replayed"`
				} `json:"data"`
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			if tt.wantCode != "" {
				assert.False(t, body.Success)
				assert.Equal(t, tt.wantCode, body.Error.Code)
				return
			}
			assert.True(t, body.Success)
			assert.Equal(t, chatID.String(), body.Data.ChatID)
			assert.Equal(t, 7, body.Data.Version)
			assert.Equal(t, tt.replayed, body.Data.Replayed)
			assert.Equal(t, "chatUpdated", rec.Header().Get("Hx-Trigger"))
		})
	}
}

func TestChatActionHandler_RejectsLongKey(t *testing.T) {
	svc := &fakeChatActions{}
	h := httphandler.NewChatActionHandler(svc)

	rec := postChatAction(t, h.Rename, uuid.NewUUID(), `{"title":"x"}`, strings.Repeat("k", 201))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Zero(t, svc.calls)
}

func TestChatActionHandler_EveryActionReportsVersion(t *testing.T) {
	svc := &fakeChatActions{}
	h := httphandler.NewChatActionHandler(svc)
	actions := map[string]struct {
		handler func(echo.Context) error
		body    string
	}{
		"status":   {h.ChangeStatus, `{"status":"Done"}`},
		"priority": {h.ChangePriority, `{"priority":"High"}`},
		"assignee": {h.ChangeAssignee, `{"assignee_id":""}`},
		"due-date": {h.SetDueDate, `{"due_date":""}`},
		"close":    {h.Close, `{}`},
		"reopen":   {h.Reopen, `{}`},
		"rename":   {h.Rename, `{"title":"New"}`},
	}
	for name, a := range actions {
		t.Run(name, func(t *testing.T) {
			rec := postChatAction(t, a.handler, uuid.NewUUID(), a.body, "key-"+name)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			assert.Contains(t, rec.Body.String(), `"version":7`)
			assert.Equal(t, "key-"+name, svc.lastKey)
		})
	}
}

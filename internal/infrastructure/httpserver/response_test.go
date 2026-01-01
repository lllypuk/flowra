package httpserver_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRespondJSON(t *testing.T) {
	tests := []struct {
		name           string
		code           int
		data           any
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "success with data",
			code:           http.StatusOK,
			data:           map[string]string{"key": "value"},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"success":true,"data":{"key":"value"}}`,
		},
		{
			name:           "success with nil data",
			code:           http.StatusOK,
			data:           nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"success":true}`,
		},
		{
			name: "created with struct",
			code: http.StatusCreated,
			data: struct {
				ID string `json:"id"`
			}{ID: "123"},
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"success":true,"data":{"id":"123"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := httpserver.RespondJSON(c, tt.code, tt.data)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
			assert.JSONEq(t, tt.expectedBody, rec.Body.String())
			assert.Contains(t, rec.Header().Get(echo.HeaderContentType), echo.MIMEApplicationJSON)
		})
	}
}

func TestRespondOK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	data := map[string]int{"count": 42}
	err := httpserver.RespondOK(c, data)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"success":true,"data":{"count":42}}`, rec.Body.String())
}

func TestRespondCreated(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	data := map[string]string{"id": "new-resource-id"}
	err := httpserver.RespondCreated(c, data)

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.JSONEq(t, `{"success":true,"data":{"id":"new-resource-id"}}`, rec.Body.String())
}

func TestRespondNoContent(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := httpserver.RespondNoContent(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestRespondError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
		expectedMsg    string
	}{
		{
			name:           "not found error",
			err:            errs.ErrNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
			expectedMsg:    "The requested resource was not found",
		},
		{
			name:           "already exists error",
			err:            errs.ErrAlreadyExists,
			expectedStatus: http.StatusConflict,
			expectedCode:   "ALREADY_EXISTS",
			expectedMsg:    "The resource already exists",
		},
		{
			name:           "invalid input error",
			err:            errs.ErrInvalidInput,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_INPUT",
			expectedMsg:    "Invalid input data",
		},
		{
			name:           "unauthorized error",
			err:            errs.ErrUnauthorized,
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
			expectedMsg:    "Authentication required",
		},
		{
			name:           "forbidden error",
			err:            errs.ErrForbidden,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
			expectedMsg:    "Access denied",
		},
		{
			name:           "concurrent modification error",
			err:            errs.ErrConcurrentModification,
			expectedStatus: http.StatusConflict,
			expectedCode:   "CONCURRENT_MODIFICATION",
			expectedMsg:    "Resource was modified by another request",
		},
		{
			name:           "invalid state error",
			err:            errs.ErrInvalidState,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   "INVALID_STATE",
			expectedMsg:    "Operation not allowed in current state",
		},
		{
			name:           "invalid transition error",
			err:            errs.ErrInvalidTransition,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   "INVALID_TRANSITION",
			expectedMsg:    "State transition not allowed",
		},
		{
			name:           "unknown error",
			err:            errors.New("something unexpected"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
			expectedMsg:    "An internal error occurred",
		},
		{
			name:           "wrapped not found error",
			err:            errors.Join(errors.New("context"), errs.ErrNotFound),
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
			expectedMsg:    "The requested resource was not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := httpserver.RespondError(c, tt.err)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			// Check response structure
			expectedBody := `{
				"success": false,
				"error": {
					"code": "` + tt.expectedCode + `",
					"message": "` + tt.expectedMsg + `"
				}
			}`
			assert.JSONEq(t, expectedBody, rec.Body.String())
		})
	}
}

func TestRespondErrorWithCode(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "VALIDATION_ERROR", "Name is required")

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	expectedBody := `{
		"success": false,
		"error": {
			"code": "VALIDATION_ERROR",
			"message": "Name is required"
		}
	}`
	assert.JSONEq(t, expectedBody, rec.Body.String())
}

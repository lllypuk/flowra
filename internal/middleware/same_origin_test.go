package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRequireSameOriginForCookie(t *testing.T) {
	tests := []struct {
		name   string
		header map[string]string
		want   int
	}{
		{"matching Origin", map[string]string{"Origin": "http://example.com"}, http.StatusOK},
		{"matching Referer", map[string]string{"Referer": "http://example.com/page"}, http.StatusOK},
		{"foreign Origin", map[string]string{"Origin": "http://evil.example"}, http.StatusForbidden},
		{"other port", map[string]string{"Origin": "http://example.com:9999"}, http.StatusForbidden},
		{"null Origin", map[string]string{"Origin": "null"}, http.StatusForbidden},
		{"neither Origin nor Referer", nil, http.StatusForbidden},
		{"Bearer without Origin", map[string]string{"Authorization": "Bearer t"}, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			ok := func(c echo.Context) error { return c.NoContent(http.StatusOK) }
			e.GET("/x", ok, middleware.RequireSameOriginForCookie)
			req := httptest.NewRequest(http.MethodGet, "http://example.com/x", nil)
			for k, v := range tt.header {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.want, rec.Code)
			if tt.want == http.StatusForbidden {
				assert.Contains(t, rec.Body.String(), middleware.CodeOriginMismatch)
			}
		})
	}
}

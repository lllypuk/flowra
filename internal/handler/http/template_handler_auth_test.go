package httphandler_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/web"
)

const (
	stateCookie    = "flowra_state"
	redirectCookie = "flowra_redirect"
	ownerSecret    = "laptop-secret"
)

func newLoginServer(t *testing.T, tokenMode bool) *echo.Echo {
	t.Helper()
	renderer, err := httphandler.NewTemplateRenderer(httphandler.TemplateRendererConfig{FS: web.TemplatesFS})
	require.NoError(t, err)

	handler := httphandler.NewTemplateHandler(renderer, nil, nil, nil)
	if tokenMode {
		handler.SetTokenLogin(httphandler.TokenLogin{
			Validator:  ownerValidator(t, ownerSecret),
			SessionTTL: 24 * time.Hour,
		})
	}

	e := echo.New()
	e.GET("/login", handler.LoginPage)
	e.POST("/login", handler.LoginSubmit)
	e.POST("/auth/logout", handler.LogoutHandler)
	return e
}

func cookieNamed(rec *httptest.ResponseRecorder, name string) *stdhttp.Cookie {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func postLogin(
	e *echo.Echo,
	token, formState, cookieState string,
	extra ...*stdhttp.Cookie,
) *httptest.ResponseRecorder {
	form := url.Values{"token": {token}, "state": {formState}}
	req := httptest.NewRequest(stdhttp.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	if cookieState != "" {
		req.AddCookie(&stdhttp.Cookie{Name: stateCookie, Value: cookieState})
	}
	for _, cookie := range extra {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestLoginPage_TokenMode_ShowsTokenForm(t *testing.T) {
	e := newLoginServer(t, true)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/login", nil))

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `name="token"`)
	assert.NotContains(t, body, "Sign in with SSO")

	state := cookieNamed(rec, stateCookie)
	require.NotNil(t, state)
	assert.Contains(t, body, `value="`+state.Value+`"`)
}

func TestLoginSubmit_TokenMode(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		formState   string
		cookieState string
		wantStatus  int
		wantSession bool
	}{
		{name: "valid token", token: ownerSecret, formState: "s1", cookieState: "s1",
			wantStatus: stdhttp.StatusSeeOther, wantSession: true},
		{name: "valid token with spaces around", token: "  " + ownerSecret + "\n", formState: "s1", cookieState: "s1",
			wantStatus: stdhttp.StatusSeeOther, wantSession: true},
		{name: "wrong token", token: "dev-token-1", formState: "s1", cookieState: "s1",
			wantStatus: stdhttp.StatusUnauthorized},
		{name: "empty token", token: "", formState: "s1", cookieState: "s1",
			wantStatus: stdhttp.StatusUnauthorized},
		{name: "state mismatch", token: ownerSecret, formState: "s1", cookieState: "s2",
			wantStatus: stdhttp.StatusBadRequest},
		{name: "no state cookie", token: ownerSecret, formState: "s1", cookieState: "",
			wantStatus: stdhttp.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postLogin(newLoginServer(t, true), tt.token, tt.formState, tt.cookieState)

			assert.Equal(t, tt.wantStatus, rec.Code)
			session := cookieNamed(rec, "flowra_session")
			if !tt.wantSession {
				assert.Nil(t, session)
				assert.NotContains(t, rec.Body.String(), ownerSecret)
				return
			}
			require.NotNil(t, session)
			assert.Equal(t, ownerSecret, session.Value)
			assert.True(t, session.HttpOnly)
			assert.Equal(t, stdhttp.SameSiteLaxMode, session.SameSite)
			assert.Equal(t, 24*60*60, session.MaxAge)
			assert.Equal(t, "/feed", rec.Header().Get("Location"), "the owner signs in to read the feed")
		})
	}
}

func TestLoginSubmit_TokenMode_ReturnsToSavedPage(t *testing.T) {
	rec := postLogin(newLoginServer(t, true), ownerSecret, "s1", "s1",
		&stdhttp.Cookie{Name: redirectCookie, Value: "/feed"})

	require.Equal(t, stdhttp.StatusSeeOther, rec.Code)
	assert.Equal(t, "/feed", rec.Header().Get("Location"))
}

func TestLoginPage_WithoutTokenLogin_StillShowsTokenForm(t *testing.T) {
	e := newLoginServer(t, false)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/login", nil))

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `name="token"`)
	assert.NotContains(t, rec.Body.String(), "/auth/callback")
}

func TestLoginSubmit_WithoutTokenMode_IsNotServed(t *testing.T) {
	rec := postLogin(newLoginServer(t, false), ownerSecret, "s1", "s1")

	assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	assert.Nil(t, cookieNamed(rec, "flowra_session"))
}

func TestLogoutHandler_ClearsSession(t *testing.T) {
	e := newLoginServer(t, true)
	req := httptest.NewRequest(stdhttp.MethodPost, "/auth/logout", nil)
	req.AddCookie(&stdhttp.Cookie{Name: "flowra_session", Value: ownerSecret})
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	session := cookieNamed(rec, "flowra_session")
	require.NotNil(t, session)
	assert.Empty(t, session.Value)
	assert.Negative(t, session.MaxAge)
}

package httphandler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
)

// maxIdempotencyKeyLength bounds the Idempotency-Key header.
const maxIdempotencyKeyLength = 200

// actionOutcome is the body of an applied action.
type actionOutcome struct {
	ChatID   string `json:"chat_id"`
	Version  int    `json:"version"`
	Replayed bool   `json:"replayed"`
}

// idempotencyKey returns the Idempotency-Key header; ok is false when an error response was written.
func idempotencyKey(c echo.Context) (string, bool, error) {
	key := c.Request().Header.Get("Idempotency-Key")
	if len(key) > maxIdempotencyKeyLength {
		return "", false, httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY",
			"Idempotency-Key must be at most 200 characters")
	}
	return key, true, nil
}

// respondAction writes the outcome of an action: applied with its version, or the error that says why not.
func respondAction(c echo.Context, res *appcore.ActionResult, err error, trigger string) error {
	if err != nil {
		return httpserver.RespondError(c, err)
	}
	c.Response().Header().Set("Hx-Trigger", trigger)
	return httpserver.RespondOK(c, outcomeOf(res))
}

func outcomeOf(res *appcore.ActionResult) actionOutcome {
	return actionOutcome{ChatID: res.ChatID.String(), Version: res.Version, Replayed: res.Replayed}
}

// requiredIdempotencyKey is idempotencyKey for routes that must not run without one.
func requiredIdempotencyKey(c echo.Context) (string, bool, error) {
	key, ok, err := idempotencyKey(c)
	if !ok {
		return "", false, err
	}
	if key == "" {
		return "", false, httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED",
			"Idempotency-Key header is required")
	}
	return key, true, nil
}

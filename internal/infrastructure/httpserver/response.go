package httpserver

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/errs"
)

// Response represents a standard API response.
type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// Error represents an error in the API response.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HTTPError interface allows application errors to define their HTTP representation.
// Errors implementing this interface will be automatically mapped to proper HTTP responses.
type HTTPError interface {
	error
	HTTPStatus() int
	HTTPCode() string
	HTTPMessage() string
}

// RespondJSON sends a successful JSON response.
func RespondJSON(c echo.Context, code int, data any) error {
	return c.JSON(code, Response{
		Success: true,
		Data:    data,
	})
}

// RespondOK sends a 200 OK response with data.
func RespondOK(c echo.Context, data any) error {
	return RespondJSON(c, http.StatusOK, data)
}

// RespondCreated sends a 201 Created response with data.
func RespondCreated(c echo.Context, data any) error {
	return RespondJSON(c, http.StatusCreated, data)
}

// RespondNoContent sends a 204 No Content response.
func RespondNoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// RespondError sends an error JSON response based on the error type.
func RespondError(c echo.Context, err error) error {
	statusCode, apiError := mapError(err)
	return c.JSON(statusCode, Response{
		Success: false,
		Error:   apiError,
	})
}

// RespondErrorWithCode sends an error JSON response with a specific HTTP status code.
func RespondErrorWithCode(c echo.Context, code int, errorCode, message string) error {
	return c.JSON(code, Response{
		Success: false,
		Error: &Error{
			Code:    errorCode,
			Message: message,
		},
	})
}

// mapError maps domain errors to HTTP status codes and API errors.
func mapError(err error) (int, *Error) {
	// First, check if the error implements HTTPError interface
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.HTTPStatus(), &Error{
			Code:    httpErr.HTTPCode(),
			Message: httpErr.HTTPMessage(),
		}
	}

	// Fall back to domain error mapping
	switch {
	case errors.Is(err, errs.ErrNotFound):
		return http.StatusNotFound, &Error{
			Code:    "NOT_FOUND",
			Message: "The requested resource was not found",
		}

	case errors.Is(err, errs.ErrAlreadyExists):
		return http.StatusConflict, &Error{
			Code:    "ALREADY_EXISTS",
			Message: "The resource already exists",
		}

	case errors.Is(err, errs.ErrInvalidInput):
		return http.StatusBadRequest, &Error{
			Code:    "INVALID_INPUT",
			Message: "Invalid input data",
		}

	case errors.Is(err, errs.ErrUnauthorized):
		return http.StatusUnauthorized, &Error{
			Code:    "UNAUTHORIZED",
			Message: "Authentication required",
		}

	case errors.Is(err, errs.ErrForbidden):
		return http.StatusForbidden, &Error{
			Code:    "FORBIDDEN",
			Message: "Access denied",
		}

	case errors.Is(err, errs.ErrConcurrentModification):
		return http.StatusConflict, &Error{
			Code:    "CONCURRENT_MODIFICATION",
			Message: "Resource was modified by another request",
		}

	case errors.Is(err, errs.ErrInvalidState):
		return http.StatusUnprocessableEntity, &Error{
			Code:    "INVALID_STATE",
			Message: "Operation not allowed in current state",
		}

	case errors.Is(err, errs.ErrInvalidTransition):
		return http.StatusUnprocessableEntity, &Error{
			Code:    "INVALID_TRANSITION",
			Message: "State transition not allowed",
		}

	default:
		return http.StatusInternalServerError, &Error{
			Code:    "INTERNAL_ERROR",
			Message: "An internal error occurred",
		}
	}
}

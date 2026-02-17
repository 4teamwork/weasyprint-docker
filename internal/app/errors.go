package app

import (
	"errors"
	"log/slog"
	"net/http"
)

type AppError struct {
	StatusCode int
	Message    string
	Cause      error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func NewBadRequestError(message string, cause error) error {
	return &AppError{StatusCode: http.StatusBadRequest, Message: message, Cause: cause}
}

func NewRequestTooLargeError(message string, cause error) error {
	return &AppError{StatusCode: http.StatusRequestEntityTooLarge, Message: message, Cause: cause}
}

func NewUnauthorizedError(message string, cause error) error {
	return &AppError{StatusCode: http.StatusUnauthorized, Message: message, Cause: cause}
}

func NewForbiddenError(message string, cause error) error {
	return &AppError{StatusCode: http.StatusForbidden, Message: message, Cause: cause}
}

func NewMethodNotAllowedError(message string, cause error) error {
	return &AppError{StatusCode: http.StatusMethodNotAllowed, Message: message, Cause: cause}
}

func NewInternalError(message string, cause error) error {
	return &AppError{StatusCode: http.StatusInternalServerError, Message: message, Cause: cause}
}

func writeHTTPError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		attributes := []any{
			"status", appErr.StatusCode,
			"message", appErr.Message,
		}
		if r != nil {
			attributes = append(attributes,
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)
		}
		if appErr.Cause != nil {
			attributes = append(attributes, "cause", appErr.Cause.Error())
		}
		slog.Error("request failed", attributes...)
		http.Error(w, appErr.Message, appErr.StatusCode)
		return
	}

	attributes := []any{"status", http.StatusInternalServerError}
	if r != nil {
		attributes = append(attributes,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)
	}
	if err != nil {
		attributes = append(attributes, "cause", err.Error())
	}
	slog.Error("request failed with unexpected error type", attributes...)

	http.Error(w, "Failed to process request.", http.StatusInternalServerError)
}

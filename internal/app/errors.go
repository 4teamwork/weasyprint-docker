package app

import (
	"errors"
	"fmt"
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

func writeHTTPError(logger *slog.Logger, w http.ResponseWriter, r *http.Request, err error) {
	var appErr *AppError

	var attributes []any

	if r != nil {
		attributes = append(attributes,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)
	}

	if errors.As(err, &appErr) {
		attributes = append(attributes,
			"status", appErr.StatusCode,
			"message", appErr.Message,
		)
		if appErr.Cause != nil {
			attributes = append(attributes, "cause", appErr.Cause.Error())
		}
		fmt.Println(attributes...)
		logger.Error("request failed", attributes...)
		http.Error(w, appErr.Message, appErr.StatusCode)
		return
	}

	attributes = append(attributes, "status", http.StatusInternalServerError)

	if err != nil {
		attributes = append(attributes, "cause", err.Error())
	}

	logger.Error("request failed with unexpected error type", attributes...)
	http.Error(w, "Failed to process request.", http.StatusInternalServerError)
}

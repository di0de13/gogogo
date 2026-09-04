package apperrors

import "fmt"

type Code string

const (
	CodeInvalidRequest Code = "INVALID_REQUEST"
	CodeNotFound       Code = "NOT_FOUND"
	CodeConflict       Code = "CONFLICT"
	CodeUnavailable    Code = "UNAVAILABLE"
	CodeInternal       Code = "INTERNAL"
)

// AppError keeps a client-safe message while retaining an internal cause.
type AppError struct {
	Code    Code
	Message string
	Status  int
	Err     error
}

func New(code Code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, Status: status}
}

func Wrap(code Code, message string, status int, err error) *AppError {
	return &AppError{Code: code, Message: message, Status: status, Err: err}
}

func (e *AppError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
}

func (e *AppError) Unwrap() error { return e.Err }

func (e *AppError) ClientMessage() string { return e.Message }

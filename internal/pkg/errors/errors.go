package errors

import "fmt"

type AppError struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func New(code, message string, status int) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

func Wrap(err error, code, message string, status int) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  status,
		Err:     err,
	}
}

// Common errors
var (
	ErrNotFound          = New("NOT_FOUND", "Resource not found", 404)
	ErrUnauthorized      = New("UNAUTHORIZED", "Unauthorized", 401)
	ErrForbidden         = New("FORBIDDEN", "Forbidden", 403)
	ErrBadRequest        = New("BAD_REQUEST", "Bad request", 400)
	ErrInternalServer    = New("INTERNAL_SERVER_ERROR", "Internal server error", 500)
	ErrValidation        = New("VALIDATION_ERROR", "Validation failed", 422)
	ErrConflict          = New("CONFLICT", "Resource already exists", 409)
	ErrTooManyRequests   = New("TOO_MANY_REQUESTS", "Too many requests", 429)
	ErrTokenExpired      = New("TOKEN_EXPIRED", "Token has expired", 401)
	ErrInvalidToken      = New("INVALID_TOKEN", "Invalid token", 401)
	ErrAccountLocked     = New("ACCOUNT_LOCKED", "Account is locked", 403)
	ErrInvalidCredentials = New("INVALID_CREDENTIALS", "Invalid credentials", 401)
)
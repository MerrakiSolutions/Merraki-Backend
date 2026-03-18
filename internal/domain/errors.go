package domain

import "errors"

// Domain errors
var (
	ErrNotFound               = errors.New("resource not found")
	ErrInsufficientStock      = errors.New("insufficient stock")
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrForbidden              = errors.New("forbidden")
	ErrDuplicateEntry         = errors.New("duplicate entry")
	ErrInvalidInput           = errors.New("invalid input")
)
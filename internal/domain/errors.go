package domain

import (
	"errors"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrConflict          = errors.New("version conflict")
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrInstrumentBlocked = errors.New("instrument is not available for use")
	ErrDuplicate         = errors.New("resource already exists")
	ErrValidation        = errors.New("validation failed")
	ErrUnauthorized      = errors.New("operation is not authorized")
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationError struct {
	Fields []FieldError `json:"fields"`
}

func (e ValidationError) Error() string { return ErrValidation.Error() }

func (e ValidationError) Unwrap() error { return ErrValidation }

func NewValidationError(field, message string) error {
	return ValidationError{Fields: []FieldError{{Field: field, Message: message}}}
}

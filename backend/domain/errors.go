package domain

import "errors"

// Domain errors. Delivery maps these to transport codes in one place.
var (
	ErrNotFound     = errors.New("not found")
	ErrValidation   = errors.New("validation failed")
	ErrConflict     = errors.New("already exists")
	ErrUnauthorized = errors.New("unauthorized")
)

// FieldError carries a validation failure without HTTP/framework knowledge.
type FieldError struct {
	Field   string
	Message string
}

func (e *FieldError) Error() string        { return e.Field + ": " + e.Message }
func (e *FieldError) Is(target error) bool { return target == ErrValidation }

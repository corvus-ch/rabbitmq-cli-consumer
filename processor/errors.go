package processor

import "fmt"

// NewCreateCommandError creates a new CreateCommandError from the given error.
func NewCreateCommandError(err error) error {
	return &CreateCommandError{err}
}

// CreateCommandError defines an error indicating that the creation of a command failed.
type CreateCommandError struct {
	err error
}

// Error is part of the error builtin.
func (e CreateCommandError) Error() string {
	return fmt.Sprintf("failed to register a consumer: %v", e.err)
}

// NewAcknowledgmentError creates a new AcknowledgmentError from the given error.
func NewAcknowledgmentError(err error) error {
	return &AcknowledgmentError{err}
}

// AcknowledgmentError defines an error indicating that acknowledgment of a message failed.
type AcknowledgmentError struct {
	err error
}

func (e AcknowledgmentError) Error() string {
	return fmt.Sprintf("failed to aknowledge message: %v", e.err)
}

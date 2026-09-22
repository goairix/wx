// Package errors defines the errors returned by the v2 client packages.
package errors

import (
	stderrors "errors"
	"fmt"
	"strings"
)

// Error describes an error returned while talking to a platform API.
//
// Err contains the underlying cause, when one is available, so callers can
// continue to use errors.Is and errors.As across the client boundary.
type Error struct {
	Platform   string
	Operation  string
	HTTPStatus int
	Code       string
	Message    string
	RequestID  string
	Err        error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	prefix := strings.TrimSpace(strings.Join([]string{e.Platform, e.Operation}, " "))
	if e.Message == "" {
		return prefix
	}
	if prefix == "" {
		return e.Message
	}
	return prefix + ": " + e.Message
}

// Unwrap returns the underlying cause, if any.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// New returns an error with the supplied message.
func New(message string) error {
	return stderrors.New(message)
}

// Errorf formats an error message.
func Errorf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// Wrap annotates err with message while preserving err as its cause.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf annotates err with a formatted message while preserving err as its cause.
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return Wrap(err, fmt.Sprintf(format, args...))
}

// Is reports whether err or one of its causes matches target.
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// As finds the first error in err's chain assignable to target.
func As(err error, target interface{}) bool {
	return stderrors.As(err, target)
}

// Unwrap returns the next error in err's chain.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

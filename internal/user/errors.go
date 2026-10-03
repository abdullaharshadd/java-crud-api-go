// Package user contains the user (contact-app account) domain: errors,
// model, persistence and HTTP handling.
package user

import (
	"errors"
	"fmt"
)

// NotFoundMessage is the exact detail message the Java service used when a
// user lookup failed. The grammatical typo is preserved verbatim because it
// is surfaced to clients in the 404 response body.
const NotFoundMessage = "User are not available"

// ErrNotFound is the sentinel for "requested user does not exist". It
// replaces the Java checked exception UserNotFoundException. Callers should
// test for it with errors.Is(err, ErrNotFound); every *NotFoundError also
// matches this sentinel.
var ErrNotFound = errors.New(NotFoundMessage)

// ErrValidation signals that a request body failed bean-style validation
// (the Go replacement for Spring's @Valid / MethodArgumentNotValidException).
var ErrValidation = errors.New("validation failed")

// NotFoundError carries an explicit detail message and/or an underlying
// cause, mirroring the constructor overloads of the Java
// UserNotFoundException. It matches ErrNotFound via errors.Is and exposes its
// cause via errors.Unwrap. Error() is the equivalent of Java getMessage(),
// which the 404 handler writes into the response body.
type NotFoundError struct {
	msg   string
	cause error
}

// NewNotFound mirrors `new UserNotFoundException()`: no message, no cause.
// Java's getMessage() would return null in that case; Error() returns "".
func NewNotFound() *NotFoundError {
	return &NotFoundError{}
}

// NewNotFoundMsg mirrors `new UserNotFoundException(String message)`.
// The service layer uses NewNotFoundMsg(NotFoundMessage).
func NewNotFoundMsg(msg string) *NotFoundError {
	return &NotFoundError{msg: msg}
}

// NewNotFoundMsgCause mirrors `new UserNotFoundException(String, Throwable)`.
func NewNotFoundMsgCause(msg string, cause error) *NotFoundError {
	return &NotFoundError{msg: msg, cause: cause}
}

// NewNotFoundCause mirrors `new UserNotFoundException(Throwable cause)`: the
// message is derived from the cause (Java uses cause.toString(); here
// cause.Error()), or empty when cause is nil.
func NewNotFoundCause(cause error) *NotFoundError {
	if cause == nil {
		return &NotFoundError{}
	}
	return &NotFoundError{msg: cause.Error(), cause: cause}
}

// MIGRATION_NOTE: the protected Java constructor
// UserNotFoundException(String, Throwable, boolean enableSuppression,
// boolean writableStackTrace) has no Go equivalent: Go errors have neither
// suppressed-exception lists nor captured stack traces. It collapses into
// NewNotFoundMsgCause; the two boolean flags are intentionally dropped.

// Error returns the detail message (Java getMessage()).
func (e *NotFoundError) Error() string {
	if e == nil {
		return ""
	}
	return e.msg
}

// Unwrap returns the underlying cause, if any (Java getCause()).
func (e *NotFoundError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Is reports that every NotFoundError is an ErrNotFound.
func (e *NotFoundError) Is(target error) bool {
	return target == ErrNotFound
}

// WrapNotFound annotates ErrNotFound with additional context while keeping it
// detectable via errors.Is.
func WrapNotFound(format string, args ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), ErrNotFound)
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package payload

import (
	"errors"
	"strings"
)

// Code identifies the normalized document payload error category.
type Code string

const (
	// CodeInvalidArgument reports invalid caller input.
	CodeInvalidArgument Code = "invalid_argument"
	// CodeFailedPrecondition reports that the current state blocks the requested operation.
	CodeFailedPrecondition Code = "failed_precondition"
	// CodeNotFound reports that the requested payload resource does not exist.
	CodeNotFound Code = "not_found"
	// CodeInternal reports an unexpected internal failure.
	CodeInternal Code = "internal"
)

// Error wraps a payload error code, message, and optional cause.
type Error struct {
	code    Code
	message string
	cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return string(CodeInternal)
	}

	message := strings.TrimSpace(e.message)
	if message == "" {
		message = string(e.code)
	}

	if e.cause == nil {
		return message
	}

	return message + ": " + e.cause.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func newCodeError(code Code, message string, cause error) error {
	return &Error{
		code:    code,
		message: strings.TrimSpace(message),
		cause:   cause,
	}
}

// InvalidArgument builds a payload error for invalid caller input.
func InvalidArgument(message string) error {
	return newCodeError(CodeInvalidArgument, message, nil)
}

// InvalidArgumentWrap builds a payload invalid-argument error with an underlying cause.
func InvalidArgumentWrap(message string, cause error) error {
	return newCodeError(CodeInvalidArgument, message, cause)
}

// FailedPrecondition builds a payload error for invalid current state.
func FailedPrecondition(message string) error {
	return newCodeError(CodeFailedPrecondition, message, nil)
}

// FailedPreconditionWrap builds a payload failed-precondition error with an underlying cause.
func FailedPreconditionWrap(message string, cause error) error {
	return newCodeError(CodeFailedPrecondition, message, cause)
}

// NotFound builds a payload error for a missing resource.
func NotFound(message string) error {
	return newCodeError(CodeNotFound, message, nil)
}

// NotFoundWrap builds a payload not-found error with an underlying cause.
func NotFoundWrap(message string, cause error) error {
	return newCodeError(CodeNotFound, message, cause)
}

// Internal builds a payload error for unexpected internal failures.
func Internal(message string) error {
	return newCodeError(CodeInternal, message, nil)
}

// InternalWrap builds a payload internal error with an underlying cause.
func InternalWrap(message string, cause error) error {
	return newCodeError(CodeInternal, message, cause)
}

// CodeOf extracts the payload error code from an error chain.
func CodeOf(err error) (Code, bool) {
	var payloadErr *Error
	if !errors.As(err, &payloadErr) || payloadErr == nil {
		return "", false
	}
	return payloadErr.code, true
}

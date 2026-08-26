// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportpkg

import (
	"errors"
	"fmt"
)

const (
	CodeCallerProfileDenied = "caller_profile_denied"
	CodeProfileNotApproved  = "profile_not_approved"
	CodeInvalidMode         = "invalid_mode"
	CodeInvalidFormat       = "invalid_format"
	CodeModelNotFound       = "model_not_found"
	CodeReaderNotRegistered = "reader_not_registered"
	CodeRunnerNotRegistered = "runner_not_registered"
	CodeAsyncNotSupported   = "async_not_supported"
	CodeInvalidSpec         = "invalid_spec"
)

var (
	ErrCallerProfileDenied = &Error{Code: CodeCallerProfileDenied, Text: "caller is not allowed for profile"}
	ErrProfileNotApproved  = &Error{Code: CodeProfileNotApproved, Text: "export profile is not approved"}
	ErrReaderNotRegistered = &Error{Code: CodeReaderNotRegistered, Text: "reader is not registered for profile"}
	ErrAsyncNotSupported   = &Error{Code: CodeAsyncNotSupported, Text: "async export is not supported by Run; use the task job path"}
)

// Error is a structured export platform error.
type Error struct {
	Code      string
	Text      string
	Row       int
	Field     string
	RecordRef string
	cause     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Text != "" {
		return e.Text
	}
	return e.Code
}

// Unwrap returns the wrapped cause for errors.Is / errors.As.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// AsError unwraps or wraps err as *Error.
func AsError(err error) (*Error, bool) {
	var target *Error
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

// Errorf builds a structured error.
func Errorf(code, text string) *Error {
	return &Error{Code: code, Text: text}
}

// ErrorfWrap builds a structured error with cause context.
func ErrorfWrap(code, text string, cause error) *Error {
	if cause == nil {
		return Errorf(code, text)
	}
	return &Error{Code: code, Text: fmt.Sprintf("%s: %v", text, cause), cause: cause}
}

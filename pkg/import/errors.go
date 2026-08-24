// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg

import (
	"errors"
	"fmt"
)

const (
	CodeCallerProfileDenied  = "caller_profile_denied"
	CodePolicyDenied         = "policy_denied"
	CodeDryRunRequiresAtomic = "dry_run_requires_atomic"
	CodeWriterNotRegistered  = "writer_not_registered"
	CodePlanBuilderNotFound  = "plan_builder_not_found"
	CodeInvalidFormat        = "invalid_format"
	CodeInvalidEncoding      = "invalid_encoding"
	CodeEmptyRequired        = "empty_required"
	CodeDuplicateKey         = "duplicate_key"
	CodeConstraint           = "constraint"
	CodeExternalIDProtected  = "external_id_protected"
	CodeExternalIDNotFound   = "external_id_not_found"
	CodeModelNotFound        = "model_not_found"
	CodeRunnerNotRegistered  = "runner_not_registered"
	CodeAsyncNotSupported    = "async_not_supported"
)

var (
	ErrCallerProfileDenied  = &Error{Code: CodeCallerProfileDenied, Text: "caller is not allowed for profile"}
	ErrPolicyDenied         = &Error{Code: CodePolicyDenied, Text: "policy is not allowed for profile"}
	ErrDryRunRequiresAtomic = &Error{Code: CodeDryRunRequiresAtomic, Text: "dry run requires atomic policy"}
	ErrWriterNotRegistered  = &Error{Code: CodeWriterNotRegistered, Text: "writer is not registered for profile"}
	ErrPlanBuilderNotFound  = &Error{Code: CodePlanBuilderNotFound, Text: "plan builder is not registered for source format"}
	ErrAsyncNotSupported    = &Error{Code: CodeAsyncNotSupported, Text: "async import is not supported by Run; use the task job path"}
)

// Error is a structured import platform error with an appendix-D code.
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

// Message converts the error to a report message.
func (e *Error) Message() Message {
	if e == nil {
		return Message{}
	}
	return Message{
		Type:      MessageError,
		Row:       e.Row,
		Field:     e.Field,
		Code:      e.Code,
		Text:      e.Text,
		RecordRef: e.RecordRef,
	}
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

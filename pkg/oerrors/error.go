// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package oerrors

import (
	"fmt"
	"strings"

	xerrors "golang.org/x/exp/errors"
	xfmt "golang.org/x/exp/errors/fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ChoysumError is the unified error type and can be used as a base error or a wrapper.
type ChoysumError struct {
	*ErrorInfo
	cause    error         // Wrapped error; nil for base errors.
	frame    xerrors.Frame // Stack-trace frame.
	hasFrame bool          // Whether stack information is available.
}

// Error implements the error interface.
func (e *ChoysumError) Error() string {
	// Build the base error message.
	msg := fmt.Sprintf("[%s] %s: %s", e.Domain, e.Code, e.Message)

	// Append important metadata.
	var metaInfo []string
	for k, v := range e.Metadata {
		if k != "stack" {
			metaInfo = append(metaInfo, fmt.Sprintf("%s=%s", k, v))
		}
	}

	if len(metaInfo) > 0 {
		msg += " (" + strings.Join(metaInfo, ", ") + ")"
	}

	// Append the original error when this is a wrapped error.
	if e.cause != nil {
		msg += fmt.Sprintf(": %v", e.cause)
	}

	return msg
}

// Unwrap links wrapped errors for errors.Is and errors.As.
func (e *ChoysumError) Unwrap() error {
	return e.cause
}

// Format implements fmt.Formatter for error and stack rendering.
func (e *ChoysumError) Format(f fmt.State, c rune) {
	// Use xerrors formatting when stack information is available on wrapped errors.
	if e.cause != nil && e.hasFrame {
		xfmt.FormatError(f, c, e)
	} else {
		// Otherwise print the error message directly.
		fmt.Fprintf(f, "%s", e.Error())
	}
}

// FormatError implements xerrors.Formatter.
func (e *ChoysumError) FormatError(p xerrors.Printer) error {
	// Build the base error message.
	prefix := fmt.Sprintf("[%s] %s: %s", e.Domain, e.Code, e.Message)

	// Append important metadata.
	var metaInfo []string
	for k, v := range e.Metadata {
		if k != "stack" {
			metaInfo = append(metaInfo, fmt.Sprintf("%s=%s", k, v))
		}
	}

	if len(metaInfo) > 0 {
		prefix += " (" + strings.Join(metaInfo, ", ") + ")"
	}

	p.Print(prefix)

	// Print the stack only in detailed mode when stack information is available.
	if p.Detail() && e.hasFrame {
		e.frame.Format(p)
	}

	// Return the wrapped error to continue chained formatting.
	return e.cause
}

// WithMetadata adds metadata and returns the same error for chaining.
func (e *ChoysumError) WithMetadata(key, value string) *ChoysumError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// WithGrpcCode sets the gRPC status code and returns the same error.
func (e *ChoysumError) WithGrpcCode(code codes.Code) *ChoysumError {
	e.GrpcCode = int32(code)
	return e
}

// ToGrpcStatus converts the error to a gRPC status.
func (e *ChoysumError) ToGrpcStatus() (*status.Status, error) {
	// Use the configured gRPC status code, defaulting to Internal.
	code := codes.Code(e.GrpcCode)
	if code == 0 {
		code = codes.Internal
	}

	st := status.New(code, e.Message)
	return st.WithDetails(e.ErrorInfo)
}

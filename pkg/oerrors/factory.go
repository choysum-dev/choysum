// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package oerrors

import (
	"errors"
	"fmt"

	"github.com/rs/xid"
	xerrors "golang.org/x/exp/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// New creates a new base error.
func New(domain, code, message string) *ChoysumError {
	return &ChoysumError{
		ErrorInfo: &ErrorInfo{
			Domain:   domain,
			Code:     code,
			Message:  message,
			ErrorId:  xid.New().String(),
			Metadata: make(map[string]string),
			GrpcCode: int32(codes.Internal), // Defaults to Internal.
		},
		cause:    nil,   // Base errors do not wrap a cause.
		hasFrame: false, // Base errors do not capture stack information.
	}
}

// Newf creates an error from a formatted string.
func Newf(domain, code, format string, args ...interface{}) *ChoysumError {
	return New(domain, code, fmt.Sprintf(format, args...))
}

// Wrap wraps an existing error and preserves stack tracking.
func Wrap(err error, domain, code, message string) error {
	if err == nil {
		return nil
	}

	// Build the error info payload.
	errInfo := &ErrorInfo{
		Domain:   domain,
		Code:     code,
		Message:  message,
		ErrorId:  xid.New().String(),
		Metadata: make(map[string]string),
		GrpcCode: int32(codes.Internal),
	}

	// Preserve ID and status code when the input is already a ChoysumError.
	var choysumErr *ChoysumError
	if errors.As(err, &choysumErr) {
		errInfo.ErrorId = choysumErr.ErrorId
		errInfo.GrpcCode = choysumErr.GrpcCode

		// Copy important metadata.
		for k, v := range choysumErr.Metadata {
			if _, exists := errInfo.Metadata[k]; !exists {
				errInfo.Metadata[k] = v
			}
		}
	}

	// Capture the current stack frame.
	frame := xerrors.Caller(1)

	// Create the wrapped error and mark that it carries stack information.
	return &ChoysumError{
		ErrorInfo: errInfo,
		cause:     err,
		frame:     frame,
		hasFrame:  true, // Explicitly mark that stack information is present.
	}
}

// Wrapf wraps an error with a formatted message.
func Wrapf(err error, domain, code, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	message := fmt.Sprintf(format, args...)
	return Wrap(err, domain, code, message)
}

// FromGrpcStatus extracts a ChoysumError from a gRPC status.
func FromGrpcStatus(s *status.Status) *ChoysumError {
	if s == nil {
		return nil
	}

	for _, detail := range s.Details() {
		if errInfo, ok := detail.(*ErrorInfo); ok {
			return &ChoysumError{ErrorInfo: errInfo}
		}
	}

	// Create a new error when ErrorInfo is not present.
	return New("grpc", "STATUS_ERROR", s.Message()).
		WithGrpcCode(s.Code())
}

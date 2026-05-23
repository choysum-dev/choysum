// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	bootstrapErrCodeGateError         = "BOOTSTRAP_GATE_ERROR"
	bootstrapErrCodeConflict          = "BOOTSTRAP_CONFLICT"
	bootstrapErrCodeInputInvalid      = "BOOTSTRAP_INPUT_INVALID"
	bootstrapErrCodeWorkspaceNotFresh = "BOOTSTRAP_WORKSPACE_NOT_FRESH"
	bootstrapErrCodeRuntimePrepare    = "BOOTSTRAP_RUNTIME_PREPARE_FAILED"
	bootstrapErrCodeAdminUpdateFailed = "BOOTSTRAP_ADMIN_UPDATE_FAILED"
	bootstrapErrCodeRuntimeNotReady   = "BOOTSTRAP_RUNTIME_NOT_READY"
	bootstrapErrCodeSwitchFailed      = "BOOTSTRAP_SWITCH_FAILED"
)

type bootstrapError struct {
	code    string
	message string
	cause   error
}

func (e *bootstrapError) Error() string {
	if e == nil {
		return ""
	}
	if e.message != "" {
		return e.message
	}
	if e.cause != nil {
		return e.cause.Error()
	}
	return e.code
}

func (e *bootstrapError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func newBootstrapError(code, message string, cause error) *bootstrapError {
	return &bootstrapError{code: code, message: message, cause: cause}
}

func bootstrapErrorCode(err error) string {
	var be *bootstrapError
	if errors.As(err, &be) {
		return be.code
	}
	return bootstrapErrCodeGateError
}

func toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	var be *bootstrapError
	if errors.As(err, &be) {
		return status.Error(mapBootstrapCodeToGRPCCode(be.code), formatBootstrapErrorMessage(be))
	}

	return status.Error(codes.Internal, err.Error())
}

func formatBootstrapErrorMessage(be *bootstrapError) string {
	if be == nil {
		return ""
	}
	code := strings.TrimSpace(be.code)
	message := strings.TrimSpace(be.Error())
	if code == "" {
		return message
	}
	if message == "" {
		return code
	}
	return code + ": " + message
}

func mapBootstrapCodeToGRPCCode(code string) codes.Code {
	switch code {
	case bootstrapErrCodeInputInvalid:
		return codes.InvalidArgument
	case bootstrapErrCodeConflict:
		return codes.Aborted
	case bootstrapErrCodeWorkspaceNotFresh, bootstrapErrCodeRuntimePrepare, bootstrapErrCodeRuntimeNotReady:
		return codes.FailedPrecondition
	case bootstrapErrCodeSwitchFailed:
		return codes.Unavailable
	case bootstrapErrCodeAdminUpdateFailed:
		return codes.Internal
	case bootstrapErrCodeGateError:
		fallthrough
	default:
		return codes.Internal
	}
}

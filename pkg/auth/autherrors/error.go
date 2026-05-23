// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package autherrors

import (
	"github.com/choysum-dev/choysum/pkg/oerrors"
)

const (
	Domain = "auth"
)

// ErrorCode enumerates auth module error codes.
type ErrorCode string

const (
	// General authentication errors.
	ErrInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	ErrInvalidSignature   ErrorCode = "INVALID_SIGNATURE"
	ErrMissingToken       ErrorCode = "MISSING_TOKEN"
	ErrPermissionDenied   ErrorCode = "PERMISSION_DENIED"
	ErrInsufficientScope  ErrorCode = "INSUFFICIENT_SCOPE"

	// Configuration-related errors.
	ErrAuthDisabled         ErrorCode = "AUTH_DISABLED"
	ErrProviderNotFound     ErrorCode = "PROVIDER_NOT_FOUND"
	ErrConfigurationError   ErrorCode = "CONFIGURATION_ERROR"
	ErrMissingConfiguration ErrorCode = "MISSING_CONFIGURATION"

	// Token-related errors.
	ErrInvalidTokenID      ErrorCode = "INVALID_TOKEN_ID"
	ErrInvalidUserID       ErrorCode = "INVALID_USER_ID"
	ErrTokenExpired        ErrorCode = "TOKEN_EXPIRED"
	ErrInvalidAccessToken  ErrorCode = "INVALID_ACCESS_TOKEN"
	ErrTokenAlreadyRevoked ErrorCode = "TOKEN_ALREADY_REVOKED"
	ErrTokenNotFound       ErrorCode = "TOKEN_NOT_FOUND"
	ErrTokenTypeMismatch   ErrorCode = "TOKEN_TYPE_MISMATCH"

	// JWT authenticator errors.
	ErrJWTConfigurationMissing   ErrorCode = "JWT_CONFIGURATION_MISSING"
	ErrKeyProviderInitFailed     ErrorCode = "KEY_PROVIDER_INIT_FAILED"
	ErrKeyProviderNotInitialized ErrorCode = "KEY_PROVIDER_NOT_INITIALIZED"
	ErrTokenSigningFailed        ErrorCode = "TOKEN_SIGNING_FAILED"
	ErrTokenParsingFailed        ErrorCode = "TOKEN_PARSING_FAILED"
	ErrInvalidTokenClaims        ErrorCode = "INVALID_TOKEN_CLAIMS"
	ErrCacheInitFailed           ErrorCode = "CACHE_INIT_FAILED"

	// Key management errors.
	ErrKeyGenerationFailed      ErrorCode = "KEY_GENERATION_FAILED"
	ErrKeyLoadingFailed         ErrorCode = "KEY_LOADING_FAILED"
	ErrInvalidKeyFormat         ErrorCode = "INVALID_KEY_FORMAT"
	ErrKeyFileWriteFailed       ErrorCode = "KEY_FILE_WRITE_FAILED"
	ErrKeyDirectoryCreateFailed ErrorCode = "KEY_DIRECTORY_CREATE_FAILED"

	// Revocation store errors.
	ErrRevocationStoreFailed      ErrorCode = "REVOCATION_STORE_FAILED"
	ErrRevocationStoreUnavailable ErrorCode = "REVOCATION_STORE_UNAVAILABLE"
	ErrTokenCleanupFailed         ErrorCode = "TOKEN_CLEANUP_FAILED"
	ErrDriverNotRegistered        ErrorCode = "DRIVER_NOT_REGISTERED"
)

// String implements the Stringer interface.
func (e ErrorCode) String() string {
	return string(e)
}

// NewAuthError creates an auth module error.
func NewAuthError(code ErrorCode, message string) *oerrors.ChoysumError {
	return oerrors.New(Domain, code.String(), message)
}

// NewAuthErrorf creates an auth module error with a formatted message.
func NewAuthErrorf(code ErrorCode, format string, args ...interface{}) *oerrors.ChoysumError {
	return oerrors.Newf(Domain, code.String(), format, args...)
}

// WrapAuthError wraps an error as an auth module error.
func WrapAuthError(err error, code ErrorCode, message string) error {
	return oerrors.Wrap(err, Domain, code.String(), message)
}

// WrapAuthErrorf wraps an error as an auth module error with a formatted message.
func WrapAuthErrorf(err error, code ErrorCode, format string, args ...interface{}) error {
	return oerrors.Wrapf(err, Domain, code.String(), format, args...)
}

// IsAuthError reports whether the error is an auth module error.
func IsAuthError(err error, codes ...ErrorCode) bool {
	if len(codes) == 0 {
		return oerrors.Is(err, Domain, "")
	}

	for _, code := range codes {
		if oerrors.Is(err, Domain, code.String()) {
			return true
		}
	}
	return false
}

// AsAuthError tries to convert an error to ChoysumError.
func AsAuthError(err error) (*oerrors.ChoysumError, bool) {
	if choysumErr := oerrors.As(err); choysumErr != nil && choysumErr.Domain == Domain {
		return choysumErr, true
	}
	return nil, false
}

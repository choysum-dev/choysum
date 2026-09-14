// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package oerrors

import (
	"errors"
	"fmt"
	"sync/atomic"
)

// ForeignErrorInfoExtractor optionally maps non-Choysum errors (for example
// QuickJS errors) into ErrorInfo. It is registered by packages that own those
// error types so pkg/oerrors does not import CGO-backed runtimes.
type ForeignErrorInfoExtractor func(error) *ErrorInfo

var foreignErrorInfoExtractor atomic.Pointer[ForeignErrorInfoExtractor]

// SetForeignErrorInfoExtractor installs or clears the optional extractor used
// by GetErrorInfo after ChoysumError matching fails. Pass nil to clear.
func SetForeignErrorInfoExtractor(fn ForeignErrorInfoExtractor) {
	if fn == nil {
		foreignErrorInfoExtractor.Store(nil)
		return
	}
	foreignErrorInfoExtractor.Store(&fn)
}

// GetErrorInfo extracts ErrorInfo from an error for logging.
func GetErrorInfo(err error) *ErrorInfo {
	if err == nil {
		return nil
	}

	// Try extracting a ChoysumError first.
	var choysumErr *ChoysumError
	if errors.As(err, &choysumErr) {
		return choysumErr.ErrorInfo
	}

	if ptr := foreignErrorInfoExtractor.Load(); ptr != nil && *ptr != nil {
		if info := (*ptr)(err); info != nil {
			return info
		}
	}

	// Return nil for unrecognized error types.
	return nil
}

// GetErrorAttrs extracts error attributes for structured logging.
func GetErrorAttrs(err error) []any {
	if err == nil {
		return nil
	}

	info := GetErrorInfo(err)
	if info == nil {
		return nil
	}

	attrs := []any{
		"error_id", info.ErrorId,
		"domain", info.Domain,
		"code", info.Code,
	}

	// Add metadata.
	for k, v := range info.Metadata {
		if k != "stack" {
			attrs = append(attrs, "metadata_"+k, v)
		}
	}

	return attrs
}

// FormatErrorBrief formats an error as a compact string.
func FormatErrorBrief(err error) string {
	info := GetErrorInfo(err)
	if info == nil {
		if err == nil {
			return "<nil>"
		}
		return err.Error()
	}

	return fmt.Sprintf("[%s] %s: %s (id=%s)",
		info.Domain, info.Code, info.Message, info.ErrorId)
}

// Is reports whether the error matches the given domain and code.
// It is similar to errors.Is but specialized for ChoysumError.
func Is(err error, domain, code string) bool {
	if err == nil {
		return false
	}

	var choysumErr *ChoysumError
	if !errors.As(err, &choysumErr) {
		return false
	}

	// Check the domain match.
	if choysumErr.Domain != domain {
		return false
	}

	// When code is empty, only the domain must match.
	if code == "" {
		return true
	}

	// Check the code match.
	return choysumErr.Code == code
}

// As tries to convert an error to ChoysumError.
// It is similar to errors.As but returns a ChoysumError pointer directly.
func As(err error) *ChoysumError {
	if err == nil {
		return nil
	}

	var choysumErr *ChoysumError
	if errors.As(err, &choysumErr) {
		return choysumErr
	}

	return nil
}

// Has reports whether the error chain contains the target error.
func Has(err error, target error) bool {
	return errors.Is(err, target)
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package oerrors

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/buke/quickjs-go"
)

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

	var qjsErr *quickjs.Error
	if errors.As(err, &qjsErr) {
		var errorInfo struct {
			ErrorId  string            `json:"errorId"`
			Domain   string            `json:"domain"`
			Code     string            `json:"code"`
			Message  string            `json:"message"`
			GrpcCode int32             `json:"grpcCode"`
			Metadata map[string]string `json:"metadata"`
			Cause    map[string]string `json:"cause"`
		}

		if err := json.Unmarshal([]byte(qjsErr.JSONString), &errorInfo); err != nil {
			return nil
		}

		// cause qjsErr.JSONString dit not contain message
		errorInfo.Message = qjsErr.Message

		return &ErrorInfo{
			ErrorId:  errorInfo.ErrorId,
			Domain:   errorInfo.Domain,
			Code:     errorInfo.Code,
			Message:  errorInfo.Message,
			GrpcCode: errorInfo.GrpcCode,
			Metadata: errorInfo.Metadata,
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

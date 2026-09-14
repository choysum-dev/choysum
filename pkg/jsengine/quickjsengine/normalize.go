// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/oerrors"
)

// NormalizeError converts a QuickJS exception into a ChoysumError at the JS→Go
// boundary. Callers can then use oerrors.GetErrorInfo without depending on
// quickjs-go. Errors that already wrap ChoysumError, or that contain no
// *quickjs.Error, are returned unchanged.
func NormalizeError(err error) error {
	if err == nil {
		return nil
	}
	var ce *oerrors.ChoysumError
	if errors.As(err, &ce) && ce != nil {
		return err
	}
	var qjsErr *quickjs.Error
	if !errors.As(err, &qjsErr) || qjsErr == nil {
		return err
	}
	if info := errorInfoFromQuickJS(qjsErr); info != nil {
		return oerrors.FromInfo(info, err)
	}
	msg := qjsErr.Message
	if msg == "" {
		msg = qjsErr.JSONString
	}
	if msg == "" {
		msg = err.Error()
	}
	return oerrors.Wrap(err, "js", "QUICKJS_ERROR", msg)
}

// NormalizeException converts Context.Exception() (or similar) at a boundary.
// When IsException was true but the engine returned a nil / typed-nil exception,
// it returns an explicit categorized error so callers never wrap nil with %w and
// GetErrorInfo still works.
func NormalizeException(ex error, missingDetails string) error {
	if ex != nil {
		var qjsErr *quickjs.Error
		if !(errors.As(ex, &qjsErr) && qjsErr == nil) {
			return NormalizeError(ex)
		}
	}
	if missingDetails == "" {
		missingDetails = "exception without details"
	}
	return oerrors.New("js", "QUICKJS_ERROR", missingDetails)
}

type quickJSErrorPayload struct {
	ErrorId  string         `json:"errorId"`
	Domain   string         `json:"domain"`
	Code     string         `json:"code"`
	Message  string         `json:"message"`
	GrpcCode int32          `json:"grpcCode"`
	Metadata map[string]any `json:"metadata"`
}

func (p quickJSErrorPayload) empty() bool {
	return p.Domain == "" && p.Code == "" && p.ErrorId == "" && p.Message == "" && len(p.Metadata) == 0 && p.GrpcCode == 0
}

func errorInfoFromQuickJS(qjsErr *quickjs.Error) *oerrors.ErrorInfo {
	if qjsErr == nil {
		return nil
	}

	var payload quickJSErrorPayload
	_ = json.Unmarshal([]byte(qjsErr.JSONString), &payload)
	// Engines often put structured Choysum payloads in Message
	// (e.g. throw new Error(JSON.stringify(info))) while JSONString is "{}".
	if payload.empty() && qjsErr.Message != "" {
		_ = json.Unmarshal([]byte(qjsErr.Message), &payload)
	}
	if payload.empty() {
		return nil
	}
	if payload.Domain == "" {
		payload.Domain = "js"
	}
	if payload.Code == "" {
		payload.Code = "QUICKJS_ERROR"
	}
	if payload.Message == "" {
		payload.Message = qjsErr.Message
	}
	if payload.Message == "" {
		payload.Message = qjsErr.JSONString
	}

	metadata := make(map[string]string, len(payload.Metadata))
	for k, v := range payload.Metadata {
		if s, ok := v.(string); ok {
			metadata[k] = s
			continue
		}
		if b, err := json.Marshal(v); err == nil {
			metadata[k] = string(b)
			continue
		}
		metadata[k] = fmt.Sprint(v)
	}

	return &oerrors.ErrorInfo{
		ErrorId:  payload.ErrorId,
		Domain:   payload.Domain,
		Code:     payload.Code,
		Message:  payload.Message,
		GrpcCode: payload.GrpcCode,
		Metadata: metadata,
	}
}

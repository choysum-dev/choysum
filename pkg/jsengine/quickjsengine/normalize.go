// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"encoding/json"
	"errors"

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
	if errors.As(err, &ce) {
		return err
	}
	var qjsErr *quickjs.Error
	if !errors.As(err, &qjsErr) {
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
// When IsException was true but the engine returned a nil exception, it returns
// an explicit missing-details error so callers never wrap nil with %w.
func NormalizeException(ex error, missingDetails string) error {
	if ex != nil {
		return NormalizeError(ex)
	}
	if missingDetails == "" {
		missingDetails = "exception without details"
	}
	return errors.New(missingDetails)
}

func errorInfoFromQuickJS(qjsErr *quickjs.Error) *oerrors.ErrorInfo {
	if qjsErr == nil {
		return nil
	}

	var payload struct {
		ErrorId  string            `json:"errorId"`
		Domain   string            `json:"domain"`
		Code     string            `json:"code"`
		Message  string            `json:"message"`
		GrpcCode int32             `json:"grpcCode"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(qjsErr.JSONString), &payload); err != nil {
		return nil
	}
	// JSON payloads without domain/code (e.g. "null" or "{}") carry no
	// structured info; fall back to the generic QuickJS error instead.
	if payload.Domain == "" && payload.Code == "" {
		return nil
	}
	if qjsErr.Message != "" {
		payload.Message = qjsErr.Message
	} else if payload.Message == "" {
		payload.Message = qjsErr.JSONString
	}

	return &oerrors.ErrorInfo{
		ErrorId:  payload.ErrorId,
		Domain:   payload.Domain,
		Code:     payload.Code,
		Message:  payload.Message,
		GrpcCode: payload.GrpcCode,
		Metadata: payload.Metadata,
	}
}

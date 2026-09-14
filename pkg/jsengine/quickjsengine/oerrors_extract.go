// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"encoding/json"
	"errors"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/oerrors"
)

func init() {
	oerrors.SetForeignErrorInfoExtractor(errorInfoFromQuickJS)
}

func errorInfoFromQuickJS(err error) *oerrors.ErrorInfo {
	var qjsErr *quickjs.Error
	if !errors.As(err, &qjsErr) {
		return nil
	}

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
	// qjsErr.JSONString does not contain message
	errorInfo.Message = qjsErr.Message

	return &oerrors.ErrorInfo{
		ErrorId:  errorInfo.ErrorId,
		Domain:   errorInfo.Domain,
		Code:     errorInfo.Code,
		Message:  errorInfo.Message,
		GrpcCode: errorInfo.GrpcCode,
		Metadata: errorInfo.Metadata,
	}
}

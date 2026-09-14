// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"testing"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/oerrors"
)

func TestErrorInfoFromQuickJS(t *testing.T) {
	qjsErr := &quickjs.Error{
		Message:    "js exploded",
		JSONString: `{"errorId":"err-1","domain":"web","code":"EJS","grpcCode":7,"metadata":{"tenant":"acme"}}`,
	}
	info := oerrors.GetErrorInfo(qjsErr)
	if info == nil {
		t.Fatal("expected quickjs error info via registered extractor")
	}
	if info.ErrorId != "err-1" || info.Domain != "web" || info.Code != "EJS" || info.Message != "js exploded" || info.GrpcCode != 7 {
		t.Fatalf("unexpected quickjs error info: %#v", info)
	}
	if info.Metadata["tenant"] != "acme" {
		t.Fatalf("expected metadata preserved, got %#v", info.Metadata)
	}

	if info := oerrors.GetErrorInfo(&quickjs.Error{Message: "bad", JSONString: "{"}); info != nil {
		t.Fatalf("expected invalid JSON to return nil, got %#v", info)
	}
}

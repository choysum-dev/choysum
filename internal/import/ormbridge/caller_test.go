// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ormbridge_test

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/ormbridge"
)

func TestServiceName(t *testing.T) {
	t.Parallel()
	got, err := ormbridge.ServiceName("base.Country", "Create")
	if err != nil || got != "base.Country.Create" {
		t.Fatalf("ServiceName = %q %v", got, err)
	}
	if _, err := ormbridge.ServiceName("Country", "Create"); err == nil {
		t.Fatal("expected error for short model name")
	}
}

func TestMergeImportContext(t *testing.T) {
	t.Parallel()
	got := ormbridge.MergeImportContext(map[string]any{"lang": "zh_CN"})
	if got["import_file"] != true {
		t.Fatalf("import_file = %#v", got["import_file"])
	}
	if got["lang"] != "zh_CN" {
		t.Fatalf("lang = %#v", got["lang"])
	}
}

func TestCallerContextRoundTrip(t *testing.T) {
	t.Parallel()
	caller := stubCaller{}
	ctx := ormbridge.ContextWithCaller(context.Background(), caller)
	got, ok := ormbridge.CallerFromContext(ctx)
	if !ok || got != caller {
		t.Fatalf("CallerFromContext = %#v %v", got, ok)
	}
}

type stubCaller struct{}

func (stubCaller) Call(context.Context, ormbridge.CallRequest) (any, error) {
	return nil, nil
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package po_test

import (
	"context"
	"strings"
	"testing"

	exportplan "github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/registry"
	posink "github.com/choysum-dev/choysum/internal/export/sink/po"
	i18npo "github.com/choysum-dev/choysum/internal/i18n/po"
)

func TestWrite_nilResult(t *testing.T) {
	w := posink.Writer{}
	if err := w.Write(context.Background(), nil, exportplan.Plan{}, nil); err == nil {
		t.Fatal("expected nil result error")
	}
}

func TestWrite_emptyEntries(t *testing.T) {
	w := posink.Writer{}
	if err := w.Write(context.Background(), nil, exportplan.Plan{}, &registry.Result{}); err == nil {
		t.Fatal("expected empty entries error")
	}
}

func TestWrite_populatesBytes(t *testing.T) {
	result := &registry.Result{
		POEntries: []i18npo.Entry{{
			Msgid:  "Hello",
			Msgstr: "你好",
		}},
	}
	w := posink.Writer{}
	if err := w.Write(context.Background(), nil, exportplan.Plan{}, result); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if len(result.POBytes) == 0 {
		t.Fatal("expected PO bytes")
	}
	if !strings.Contains(string(result.POBytes), `msgid "Hello"`) {
		t.Fatalf("body = %q", string(result.POBytes))
	}
}

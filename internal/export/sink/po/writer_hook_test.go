// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package po

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	exportplan "github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/registry"
	i18npo "github.com/choysum-dev/choysum/internal/i18n/po"
)

func TestWrite_poWriteError(t *testing.T) {
	orig := writePOEntries
	t.Cleanup(func() { writePOEntries = orig })
	writePOEntries = func(io.Writer, []i18npo.Entry) error { return errors.New("write boom") }

	w := Writer{}
	err := w.Write(context.Background(), nil, exportplan.Plan{}, &registry.Result{
		POEntries: []i18npo.Entry{{Msgid: "Hello", Msgstr: "你好"}},
	})
	if err == nil || !strings.Contains(err.Error(), "write po") {
		t.Fatalf("err = %v", err)
	}
}

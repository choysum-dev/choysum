// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importcli_test

import (
	"context"
	"testing"

	importcli "github.com/choysum-dev/choysum/internal/cli/import"
)

func TestRunRecord_RequiresModelAndSource(t *testing.T) {
	_, err := importcli.RunRecord(context.Background(), nil, importcli.RecordOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}

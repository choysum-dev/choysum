// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package json_test

import (
	"context"
	"testing"

	jsonadapter "github.com/choysum-dev/choysum/internal/import/adapter/json"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestBuilder_BuildInitdataPlan(t *testing.T) {
	plan, err := jsonadapter.Builder{}.Build(context.Background(), importpkg.Spec{
		Profile: importpkg.ProfileInitdata,
		Module:  "auth",
		Source:  importpkg.Source{Format: "json", Path: "/modules/auth"},
		Options: importpkg.Options{
			InitdataFiles: []string{"data/bootstrap.json"},
			WithDemo:      true,
			DemoFiles:     []string{"data/demo.json"},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(plan.Units) != 2 {
		t.Fatalf("unit count = %d, want 2", len(plan.Units))
	}
}

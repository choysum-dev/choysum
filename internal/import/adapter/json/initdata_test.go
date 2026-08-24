// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package json_test

import (
	"context"
	"errors"
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

func TestBuilder_BuildValidationErrors(t *testing.T) {
	_, err := jsonadapter.Builder{}.Build(context.Background(), importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Module:  "auth",
		Source:  importpkg.Source{Format: "json", Path: "/modules/auth"},
	})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("wrong profile error = %v", err)
	}

	_, err = jsonadapter.Builder{}.Build(context.Background(), importpkg.Spec{
		Profile: importpkg.ProfileInitdata,
		Module:  "auth",
		Source:  importpkg.Source{Format: "json"},
	})
	if !errors.As(err, &impErr) {
		t.Fatalf("missing path error = %v", err)
	}

	_, err = jsonadapter.Builder{}.Build(context.Background(), importpkg.Spec{
		Profile: importpkg.ProfileInitdata,
		Source:  importpkg.Source{Format: "json", Path: "/modules/auth"},
	})
	if !errors.As(err, &impErr) {
		t.Fatalf("missing module error = %v", err)
	}
}

func TestBuilder_BuildSkipsEmptyPathsAndDemoWithoutFiles(t *testing.T) {
	plan, err := jsonadapter.Builder{}.Build(context.Background(), importpkg.Spec{
		Profile: importpkg.ProfileInitdata,
		Module:  "auth",
		Source:  importpkg.Source{Format: "json", Path: "/modules/auth"},
		Options: importpkg.Options{
			InitdataFiles: []string{" ", "data/bootstrap.json"},
			WithDemo:      true,
			DemoFiles:     []string{"  "},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(plan.Units) != 1 {
		t.Fatalf("unit count = %d, want 1", len(plan.Units))
	}

	plan, err = jsonadapter.Builder{}.Build(context.Background(), importpkg.Spec{
		Profile: importpkg.ProfileInitdata,
		Module:  "auth",
		Source:  importpkg.Source{Format: "json", Path: "/modules/auth"},
	})
	if err != nil {
		t.Fatalf("Build empty: %v", err)
	}
	if len(plan.Units) != 0 {
		t.Fatalf("unit count = %d, want 0", len(plan.Units))
	}
}

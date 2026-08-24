// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/runner"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestApplyInitdata_nilModule(t *testing.T) {
	if err := applyInitdata(context.Background(), nil, nil, importpkg.CallerLifecycle, false); err != nil {
		t.Fatalf("applyInitdata(nil): %v", err)
	}
}

func TestApplyInitdata_buildsSpecAndRuns(t *testing.T) {
	t.Cleanup(func() { importpkg.SetRun(runner.Run) })

	var got importpkg.Spec
	importpkg.SetRun(func(_ context.Context, _ scope.Scope, spec importpkg.Spec) (importpkg.Report, error) {
		got = spec
		return importpkg.Report{Profile: importpkg.ProfileInitdata}, nil
	})

	dataJSON, err := json.Marshal([]string{"data/bootstrap.json"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	demoJSON, err := json.Marshal([]string{"data/demo.json"})
	if err != nil {
		t.Fatalf("marshal demo: %v", err)
	}
	mod := &meta.Module{
		Name:           "auth",
		Path:           "/modules/auth",
		ApplicationStr: "auth",
		DataStr:        dataJSON,
		DemoStr:        demoJSON,
	}
	if err := applyInitdata(context.Background(), nil, mod, importpkg.CallerLifecycle, true); err != nil {
		t.Fatalf("applyInitdata: %v", err)
	}
	if got.Profile != importpkg.ProfileInitdata || got.Caller != importpkg.CallerLifecycle {
		t.Fatalf("spec = %+v", got)
	}
	if !got.Options.WithDemo || len(got.Options.InitdataFiles) != 1 || len(got.Options.DemoFiles) != 1 {
		t.Fatalf("options = %+v", got.Options)
	}
}

func TestApplyInitdata_invalidManifestJSON(t *testing.T) {
	mod := &meta.Module{Name: "auth", Path: "/tmp", DataStr: []byte("{")}
	if err := applyInitdata(context.Background(), nil, mod, importpkg.CallerLifecycle, false); err == nil {
		t.Fatal("expected manifest decode error")
	}
}

func TestApplyInitdata_invalidDemoManifestJSON(t *testing.T) {
	dataJSON, err := json.Marshal([]string{"data.json"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	mod := &meta.Module{Name: "auth", Path: "/tmp", DataStr: dataJSON, DemoStr: []byte("{")}
	if err := applyInitdata(context.Background(), nil, mod, importpkg.CallerLifecycle, true); err == nil {
		t.Fatal("expected demo manifest decode error")
	}
}

func TestApplyInitdata_ignoresMalformedDemoWhenWithDemoFalse(t *testing.T) {
	t.Cleanup(func() { importpkg.SetRun(runner.Run) })

	var got importpkg.Spec
	importpkg.SetRun(func(_ context.Context, _ scope.Scope, spec importpkg.Spec) (importpkg.Report, error) {
		got = spec
		return importpkg.Report{Profile: importpkg.ProfileInitdata}, nil
	})

	dataJSON, err := json.Marshal([]string{"data/bootstrap.json"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	mod := &meta.Module{
		Name:           "auth",
		Path:           "/modules/auth",
		ApplicationStr: "auth",
		DataStr:        dataJSON,
		DemoStr:        []byte("{"),
	}
	if err := applyInitdata(context.Background(), nil, mod, importpkg.CallerLifecycle, false); err != nil {
		t.Fatalf("applyInitdata with malformed demo and withDemo=false: %v", err)
	}
	if got.Options.WithDemo || len(got.Options.DemoFiles) != 0 {
		t.Fatalf("demo options should be omitted when withDemo=false, got %+v", got.Options)
	}
}

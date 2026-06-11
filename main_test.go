// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"runtime/debug"
	"strings"
	"testing"
)

type fakeCommander struct {
	execute func() error
}

func (f fakeCommander) Execute() error {
	if f.execute != nil {
		return f.execute()
	}
	return nil
}

func TestMainExecutesCommander(t *testing.T) {
	originalNewCommander := newCommander
	originalExit := exitFunc
	t.Cleanup(func() {
		newCommander = originalNewCommander
		exitFunc = originalExit
	})

	called := false
	exitCode := -1
	newCommander = func(ctx context.Context) interface{ Execute() error } {
		if ctx == nil {
			t.Fatal("expected non-nil context")
		}
		return fakeCommander{execute: func() error {
			called = true
			return nil
		}}
	}
	exitFunc = func(code int) {
		exitCode = code
	}

	main()

	if !called {
		t.Fatal("expected commander Execute to be called")
	}
	if exitCode != -1 {
		t.Fatalf("unexpected exit code %d", exitCode)
	}
}

func TestMainExitsOnCommanderError(t *testing.T) {
	originalNewCommander := newCommander
	originalExit := exitFunc
	t.Cleanup(func() {
		newCommander = originalNewCommander
		exitFunc = originalExit
	})

	called := false
	exitCode := -1
	newCommander = func(context.Context) interface{ Execute() error } {
		return fakeCommander{execute: func() error {
			called = true
			return errors.New("boom")
		}}
	}
	exitFunc = func(code int) {
		exitCode = code
	}

	main()

	if !called {
		t.Fatal("expected commander Execute to be called")
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
}

func TestDefaultNewCommanderFactory(t *testing.T) {
	originalNewCommander := newCommander
	originalVersion := version
	originalCommit := commit
	originalDate := date
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		newCommander = originalNewCommander
		version = originalVersion
		commit = originalCommit
		date = originalDate
		readBuildInfo = originalReadBuildInfo
	})

	version = "dev"
	commit = "none"
	date = "unknown"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return nil, false
	}

	commander := originalNewCommander(context.Background())
	if commander == nil {
		t.Fatal("expected default commander factory to return a commander")
	}
}

func TestGetBuildVersion_ReleaseBuildSkipsBuildInfo(t *testing.T) {
	originalVersion := version
	originalCommit := commit
	originalDate := date
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		date = originalDate
		readBuildInfo = originalReadBuildInfo
	})

	version = "v1.2.3"
	commit = "abc1234"
	date = "2026-06-11T00:00:00Z"
	called := false
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		called = true
		return nil, false
	}

	got := getBuildVersion()

	if called {
		t.Fatal("expected build info reader to be skipped for non-dev version")
	}
	if got != "v1.2.3 (commit: abc1234, date: 2026-06-11T00:00:00Z)" {
		t.Fatalf("unexpected version string: %q", got)
	}
}

func TestGetBuildVersion_DevBuildWithLongRevision(t *testing.T) {
	originalVersion := version
	originalCommit := commit
	originalDate := date
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		date = originalDate
		readBuildInfo = originalReadBuildInfo
	})

	version = "dev"
	commit = "none"
	date = "unknown"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "1234567890abcdef"},
				{Key: "vcs.time", Value: "2026-06-11T03:00:00Z"},
			},
		}, true
	}

	got := getBuildVersion()

	if !strings.Contains(got, "dev-1234567") {
		t.Fatalf("expected shortened revision in version string, got %q", got)
	}
	if !strings.Contains(got, "date: 2026-06-11T03:00:00Z") {
		t.Fatalf("expected vcs.time in version string, got %q", got)
	}
}

func TestGetBuildVersion_DevBuildWithDirtyTree(t *testing.T) {
	originalVersion := version
	originalCommit := commit
	originalDate := date
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		date = originalDate
		readBuildInfo = originalReadBuildInfo
	})

	version = "dev"
	commit = "none"
	date = "unknown"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "1234567890abcdef"},
				{Key: "vcs.modified", Value: "true"},
			},
		}, true
	}

	got := getBuildVersion()

	if !strings.Contains(got, "dev-1234567-dirty") {
		t.Fatalf("expected dirty suffix in version string, got %q", got)
	}
	if !strings.Contains(got, "commit: 1234567") {
		t.Fatalf("expected clean commit field without dirty suffix, got %q", got)
	}
}

func TestGetBuildVersion_DevBuildUsesMainVersionWhenAvailable(t *testing.T) {
	originalVersion := version
	originalCommit := commit
	originalDate := date
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		date = originalDate
		readBuildInfo = originalReadBuildInfo
	})

	version = "dev"
	commit = "none"
	date = "unknown"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "v1.2.3"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "1234567890abcdef"},
				{Key: "vcs.time", Value: "2026-06-11T05:00:00Z"},
			},
		}, true
	}

	got := getBuildVersion()

	if got != "v1.2.3 (commit: 1234567, date: 2026-06-11T05:00:00Z)" {
		t.Fatalf("unexpected version string: %q", got)
	}
}

func TestGetBuildVersion_DevBuildIgnoresDevelMainVersion(t *testing.T) {
	originalVersion := version
	originalCommit := commit
	originalDate := date
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		date = originalDate
		readBuildInfo = originalReadBuildInfo
	})

	version = "dev"
	commit = "none"
	date = "unknown"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "(devel)"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "1234567890abcdef"},
				{Key: "vcs.modified", Value: "true"},
			},
		}, true
	}

	got := getBuildVersion()

	if got != "dev-1234567-dirty (commit: 1234567, date: unknown)" {
		t.Fatalf("unexpected version string: %q", got)
	}
}

func TestGetBuildVersion_DevBuildWithoutRevisionFallsBackToDev(t *testing.T) {
	originalVersion := version
	originalCommit := commit
	originalDate := date
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		date = originalDate
		readBuildInfo = originalReadBuildInfo
	})

	version = "dev"
	commit = "none"
	date = "unknown"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Settings: []debug.BuildSetting{
				{Key: "vcs.time", Value: "2026-06-11T04:30:00Z"},
				{Key: "vcs.modified", Value: "true"},
			},
		}, true
	}

	got := getBuildVersion()

	if got != "dev (commit: none, date: 2026-06-11T04:30:00Z)" {
		t.Fatalf("unexpected version string: %q", got)
	}
}

func TestGetBuildVersion_DevBuildWithShortRevision(t *testing.T) {
	originalVersion := version
	originalCommit := commit
	originalDate := date
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		date = originalDate
		readBuildInfo = originalReadBuildInfo
	})

	version = "dev"
	commit = "none"
	date = "unknown"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abc"},
			},
		}, true
	}

	got := getBuildVersion()

	if !strings.Contains(got, "dev-abc") {
		t.Fatalf("expected short revision without panic, got %q", got)
	}
}

func TestGetBuildVersion_DevBuildWithoutBuildInfo(t *testing.T) {
	originalVersion := version
	originalCommit := commit
	originalDate := date
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		date = originalDate
		readBuildInfo = originalReadBuildInfo
	})

	version = "dev"
	commit = "none"
	date = "unknown"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return nil, false
	}

	got := getBuildVersion()

	if got != "dev (commit: none, date: unknown)" {
		t.Fatalf("unexpected version string: %q", got)
	}
}

func TestGetBuildVersion_DevBuildWithNilBuildInfo(t *testing.T) {
	originalVersion := version
	originalCommit := commit
	originalDate := date
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		date = originalDate
		readBuildInfo = originalReadBuildInfo
	})

	version = "dev"
	commit = "none"
	date = "unknown"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return nil, true
	}

	got := getBuildVersion()

	if got != "dev (commit: none, date: unknown)" {
		t.Fatalf("unexpected version string: %q", got)
	}
}

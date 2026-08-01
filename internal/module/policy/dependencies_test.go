// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestResolveInstalledDependencies_EmptyDependsStrReturnsNoDeps(t *testing.T) {
	called := false
	deps, err := ResolveInstalledDependencies(func(name string) (*meta.Module, error) {
		called = true
		return &meta.Module{Name: name, Status: meta.Installed}, nil
	}, &meta.Module{Name: "core"})
	if err != nil {
		t.Fatalf("ResolveInstalledDependencies() error = %v", err)
	}
	if called {
		t.Fatal("loader should not be called when depends is empty")
	}
	if len(deps) != 0 {
		t.Fatalf("expected no dependencies, got %d", len(deps))
	}
}

func TestResolveInstalledDependencies_WhitespaceDependsStrReturnsNoDeps(t *testing.T) {
	called := false
	deps, err := ResolveInstalledDependencies(func(name string) (*meta.Module, error) {
		called = true
		return &meta.Module{Name: name, Status: meta.Installed}, nil
	}, &meta.Module{Name: "core", DependsStr: []byte("   ")})
	if err != nil {
		t.Fatalf("ResolveInstalledDependencies() error = %v", err)
	}
	if called {
		t.Fatal("loader should not be called when depends is whitespace")
	}
	if len(deps) != 0 {
		t.Fatalf("expected no dependencies, got %d", len(deps))
	}
}

func TestResolveInstalledDependencies_InvalidDependsStrReturnsError(t *testing.T) {
	_, err := ResolveInstalledDependencies(func(name string) (*meta.Module, error) {
		return &meta.Module{Name: name, Status: meta.Installed}, nil
	}, &meta.Module{Name: "core", DependsStr: []byte("[")})
	if err == nil || !strings.Contains(err.Error(), "error unmarshal depends") {
		t.Fatalf("expected unmarshal error, got %v", err)
	}
}

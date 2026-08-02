// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendplugin

import (
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/esbplugins"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestRegisterVirtualSource_NilAndEmpty(t *testing.T) {
	(*BackendPlugin)(nil).RegisterVirtualSource("/x/field_default.ts", "x")
	p := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{}}
	p.RegisterVirtualSource("   ", "nope")
	p.Mu.RLock()
	defer p.Mu.RUnlock()
	if len(p.virtualSources) != 0 {
		t.Fatalf("expected no sources, got %#v", p.virtualSources)
	}
}

func TestRegisterVirtualSource_DualKeys(t *testing.T) {
	p := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{}}
	raw := filepath.Join(t.TempDir(), "partner", "service", "models", "__generated__", "field_default.ts")
	p.RegisterVirtualSource(raw, "export default class FieldDefault {}")
	p.Mu.RLock()
	defer p.Mu.RUnlock()
	normalized := normalizeBackendPluginPath(raw)
	slashKey := filepath.ToSlash(filepath.Clean(raw))
	if _, ok := p.virtualSources[normalized]; !ok {
		t.Fatalf("expected normalized key %q in %#v", normalized, p.virtualSources)
	}
	if slashKey != normalized {
		if _, ok := p.virtualSources[slashKey]; !ok {
			t.Fatalf("expected slash key %q in %#v", slashKey, p.virtualSources)
		}
	}
}

func TestLookupVirtualSource_EmptyCandidateSkipped(t *testing.T) {
	p := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{}}
	p.RegisterVirtualSource(filepath.Join(t.TempDir(), "field_default.ts"), "body")
	p.Mu.RLock()
	defer p.Mu.RUnlock()
	if _, ok := p.lookupVirtualSource("   "); ok {
		t.Fatal("whitespace path should not resolve")
	}
	if _, ok := (*BackendPlugin)(nil).lookupVirtualSource("x"); ok {
		t.Fatal("nil plugin")
	}
	empty := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{}}
	if _, ok := empty.lookupVirtualSource("x"); ok {
		t.Fatal("empty map")
	}
}

func TestResolveVirtualSourcePath_RelativeAndGuards(t *testing.T) {
	if _, ok := (*BackendPlugin)(nil).resolveVirtualSourcePath("x", ""); ok {
		t.Fatal("nil plugin")
	}
	p := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{
		Module: &meta.Module{Path: filepath.Join(t.TempDir(), "partner")},
	}}
	if _, ok := p.resolveVirtualSourcePath("   ", t.TempDir()); ok {
		t.Fatal("empty path")
	}

	dir := t.TempDir()
	abs := filepath.Join(dir, "service", "models", "__generated__", "field_default.ts")
	p.RegisterVirtualSource(abs, "export default class FieldDefault {}")

	rel := "service/models/__generated__/field_default.ts"
	resolved, ok := p.resolveVirtualSourcePath(rel, filepath.Join(dir))
	if !ok || resolved == "" {
		t.Fatalf("expected relative resolve, got %q ok=%v", resolved, ok)
	}

	// Fall through when unregistered.
	if _, ok := p.resolveVirtualSourcePath("other/field_default.ts", dir); ok {
		t.Fatal("unregistered relative should miss")
	}
}

func TestResolveVirtualSourcePath_CleanFallback(t *testing.T) {
	p := &BackendPlugin{BasePlugin: &esbplugins.BasePlugin{}}
	p.Mu.Lock()
	p.virtualSources = map[string]string{
		"field_default.ts": "body",
	}
	p.Mu.Unlock()
	resolved, ok := p.resolveVirtualSourcePath("field_default.ts", "")
	if !ok {
		t.Fatal("expected resolve via raw key")
	}
	if resolved == "" {
		t.Fatal("expected non-empty resolved path")
	}
}

func TestFirstNonEmptyPath(t *testing.T) {
	if got := firstNonEmptyPath("", "  ", "keep"); got != "keep" {
		t.Fatalf("got %q", got)
	}
	if got := firstNonEmptyPath("", "  "); got != "" {
		t.Fatalf("got %q", got)
	}
}

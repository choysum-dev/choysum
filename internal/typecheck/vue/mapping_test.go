// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vue_test

import (
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/typecheck/vue"
)

func TestRemapOffset_SimpleSegment(t *testing.T) {
	mappings := []vue.SpanMapping{
		{SourceStart: 10, SourceEnd: 20, GeneratedStart: 100, GeneratedEnd: 110, Verification: true},
		{SourceStart: 50, SourceEnd: 51, GeneratedStart: 200, GeneratedEnd: 201, Verification: false},
		{SourceStart: 0, SourceEnd: 0, GeneratedStart: 300, GeneratedEnd: 301, Verification: true},
	}
	pos, ok := vue.RemapOffset(mappings, 105)
	if !ok || pos != 15 {
		t.Fatalf("got %d %v", pos, ok)
	}
	if _, ok := vue.RemapOffset(mappings, 200); ok {
		t.Fatal("verification=false must be skipped")
	}
	if _, ok := vue.RemapOffset(mappings, 999); ok {
		t.Fatal("miss must be false")
	}
	pos, ok = vue.RemapOffset(mappings, 300)
	if !ok || pos != 0 {
		t.Fatalf("zero-length source segment: %d %v", pos, ok)
	}
	// Clamp when generated span is longer than source span.
	pos, ok = vue.RemapOffset([]vue.SpanMapping{
		{SourceStart: 1, SourceEnd: 2, GeneratedStart: 10, GeneratedEnd: 20, Verification: true},
	}, 18)
	if !ok || pos != 1 {
		t.Fatalf("clamp got %d %v", pos, ok)
	}
}

func TestGoldenCoder_LoadsFixture(t *testing.T) {
	golden, err := filepath.Abs(filepath.Join("..", "testdata", "vue", "golden"))
	if err != nil {
		t.Fatal(err)
	}
	coder := vue.NewGoldenCoder(golden)
	script, err := coder.CreateServiceScript("/x/script_setup_ok.vue", "", vue.CodegenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if script.ScriptKind != "ts" || script.EmbeddedID != "script_ts" {
		t.Fatalf("%#v", script)
	}
	if script.Content == "" || len(script.Mappings) == 0 {
		t.Fatal("empty golden")
	}
	if _, err := coder.CreateServiceScript("/x/missing.vue", "", vue.CodegenOptions{}); err == nil {
		t.Fatal("expected missing golden error")
	}
	var nilCoder *vue.GoldenCoder
	if _, err := nilCoder.CreateServiceScript("a.vue", "", vue.CodegenOptions{}); err == nil {
		t.Fatal("nil coder")
	}
	if _, err := vue.NewGoldenCoder(golden).CreateServiceScript("", "", vue.CodegenOptions{}); err == nil {
		t.Fatal("empty path")
	}
}

func TestHelperOverlays(t *testing.T) {
	o := vue.HelperOverlays()
	if o[vue.HelperTemplatePath] == "" || o[vue.HelperPropsPath] == "" {
		t.Fatalf("%#v", o)
	}
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vue_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/typecheck/vue"
)

func TestRemapOffset_SimpleSegment(t *testing.T) {
	mappings := []vue.SpanMapping{
		{SourceStart: 10, SourceEnd: 20, GeneratedStart: 100, GeneratedEnd: 110, Verification: true},
		{SourceStart: 50, SourceEnd: 51, GeneratedStart: 200, GeneratedEnd: 201, Verification: false},
		{SourceStart: 0, SourceEnd: 0, GeneratedStart: 300, GeneratedEnd: 301, Verification: true},
		// Overlapping inner segment wins via greater GeneratedStart.
		{SourceStart: 12, SourceEnd: 14, GeneratedStart: 104, GeneratedEnd: 106, Verification: true},
	}
	pos, ok := vue.RemapOffset(mappings, 105)
	if !ok || pos != 13 {
		t.Fatalf("inner segment got %d %v", pos, ok)
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
	pos, ok = vue.RemapOffset([]vue.SpanMapping{
		{SourceStart: 1, SourceEnd: 2, GeneratedStart: 10, GeneratedEnd: 20, Verification: true},
	}, 18)
	if !ok || pos != 1 {
		t.Fatalf("clamp got %d %v", pos, ok)
	}
}

func TestRemapRange(t *testing.T) {
	mappings := []vue.SpanMapping{
		{SourceStart: 10, SourceEnd: 20, GeneratedStart: 100, GeneratedEnd: 110, Verification: true},
	}
	start, length, ok := vue.RemapRange(mappings, 102, 3)
	if !ok || start != 12 || length != 3 {
		t.Fatalf("got %d %d %v", start, length, ok)
	}
	start, length, ok = vue.RemapRange(mappings, 102, 0)
	if !ok || start != 12 || length != 0 {
		t.Fatalf("zero length got %d %d %v", start, length, ok)
	}
	if _, _, ok := vue.RemapRange(mappings, 999, 1); ok {
		t.Fatal("miss")
	}
	// End falls outside mapping: keep start with length 1.
	start, length, ok = vue.RemapRange([]vue.SpanMapping{
		{SourceStart: 5, SourceEnd: 6, GeneratedStart: 50, GeneratedEnd: 51, Verification: true},
	}, 50, 5)
	if !ok || start != 5 || length != 1 {
		t.Fatalf("partial end got %d %d %v", start, length, ok)
	}
	// Inverted source ends (adjacent gen spans map to descending source offsets).
	start, length, ok = vue.RemapRange([]vue.SpanMapping{
		{SourceStart: 100, SourceEnd: 101, GeneratedStart: 10, GeneratedEnd: 11, Verification: true},
		{SourceStart: 50, SourceEnd: 51, GeneratedStart: 11, GeneratedEnd: 12, Verification: true},
	}, 10, 2)
	if !ok || start != 100 || length != 0 {
		t.Fatalf("inverted end got %d %d %v", start, length, ok)
	}
}

func TestGoldenCoder_LoadsFixture(t *testing.T) {
	golden, err := filepath.Abs(filepath.Join("..", "testdata", "vue", "golden"))
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile(filepath.Join("..", "testdata", "vue", "fixtures", "script_setup_ok.vue"))
	if err != nil {
		t.Fatal(err)
	}
	coder := vue.NewGoldenCoder(golden)
	script, err := coder.CreateServiceScript("/x/script_setup_ok.vue", string(fixture), vue.CodegenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if script.ScriptKind != "ts" || script.EmbeddedID != "script_ts" {
		t.Fatalf("%#v", script)
	}
	if script.Content == "" || len(script.Mappings) == 0 || script.SourceContent == "" {
		t.Fatal("empty golden")
	}
	if _, err := coder.CreateServiceScript("/x/script_setup_ok.vue", "tampered", vue.CodegenOptions{}); err == nil {
		t.Fatal("expected SHA mismatch")
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

	// Corrupt mappings JSON.
	badDir := t.TempDir()
	mustWrite(t, filepath.Join(badDir, "x.vue.service.txt"), "export {};\n")
	mustWrite(t, filepath.Join(badDir, "x.vue.mappings.json"), "{not-json")
	if _, err := vue.NewGoldenCoder(badDir).CreateServiceScript("x.vue", "", vue.CodegenOptions{}); err == nil {
		t.Fatal("expected parse error")
	}
	// Missing mappings file.
	badDir2 := t.TempDir()
	mustWrite(t, filepath.Join(badDir2, "y.vue.service.txt"), "export {};\n")
	if _, err := vue.NewGoldenCoder(badDir2).CreateServiceScript("y.vue", "", vue.CodegenOptions{}); err == nil {
		t.Fatal("expected missing meta")
	}
}

func TestHelperOverlays(t *testing.T) {
	o := vue.HelperOverlays()
	if o[vue.HelperTemplatePath] == "" || o[vue.HelperPropsPath] == "" {
		t.Fatalf("%#v", o)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

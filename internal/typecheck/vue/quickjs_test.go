// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vue_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/typecheck/vue"
)

func TestQuickJSCoder_MatchesGolden(t *testing.T) {
	fixturePath := filepath.Join("..", "testdata", "vue", "fixtures", "script_setup_ok.vue")
	source, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	goldenDir, err := filepath.Abs(filepath.Join("..", "testdata", "vue", "golden"))
	if err != nil {
		t.Fatal(err)
	}
	golden := vue.NewGoldenCoder(goldenDir)
	want, err := golden.CreateServiceScript("/fixtures/script_setup_ok.vue", string(source), vue.CodegenOptions{})
	if err != nil {
		t.Fatal(err)
	}

	coder := vue.NewQuickJSCoder()
	t.Cleanup(func() { _ = coder.Close() })
	got, err := coder.CreateServiceScript("/fixtures/script_setup_ok.vue", string(source), vue.CodegenOptions{
		CurrentDirectory: "/fixtures",
	})
	if err != nil {
		t.Fatalf("QuickJSCoder: %v", err)
	}
	if got.EmbeddedID != want.EmbeddedID || got.ScriptKind != want.ScriptKind {
		t.Fatalf("meta got %#v want %#v", got, want)
	}
	if got.Content != want.Content {
		t.Fatalf("content mismatch (got %d bytes, want %d)", len(got.Content), len(want.Content))
	}
	if len(got.Mappings) == 0 {
		t.Fatal("empty mappings")
	}
}

func TestQuickJSCoder_NoNode(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	fixturePath := filepath.Join("..", "testdata", "vue", "fixtures", "script_setup_ok.vue")
	source, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	coder := vue.NewQuickJSCoder()
	t.Cleanup(func() { _ = coder.Close() })
	got, err := coder.CreateServiceScript("/x/script_setup_ok.vue", string(source), vue.CodegenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Content == "" || got.ScriptKind != "ts" {
		t.Fatalf("got kind=%q contentLen=%d", got.ScriptKind, len(got.Content))
	}
}

func TestQuickJSCoder_Nil(t *testing.T) {
	var c *vue.QuickJSCoder
	if _, err := c.CreateServiceScript("a.vue", "", vue.CodegenOptions{}); err == nil {
		t.Fatal("expected nil coder error")
	}
}

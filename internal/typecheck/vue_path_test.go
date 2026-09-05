// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestScriptKindStrategy_B(t *testing.T) {
	if ScriptKindStrategy != "B" {
		t.Fatalf("ScriptKindStrategy = %q", ScriptKindStrategy)
	}
	got := toVueProgramPath("/app/web/App.vue")
	if got != "/app/web/App.vue.ts" {
		t.Fatalf("got %q", got)
	}
	vuePath, ok := fromVueProgramPath("/app/web/App.vue.ts")
	if !ok || vuePath != "/app/web/App.vue" {
		t.Fatalf("got %q %v", vuePath, ok)
	}
}

func TestCollectRootFiles_All(t *testing.T) {
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	app := filepath.Join(modules, "demo")
	mustMkdir(t, filepath.Join(app, "web"))
	mustWrite(t, filepath.Join(app, "web", "ui.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "web", "App.vue"), "<script setup lang=\"ts\"></script>\n")
	mustWrite(t, filepath.Join(app, "web", "Skip.test.vue"), "<script setup lang=\"ts\"></script>\n")

	files, err := CollectRootFiles(t.Context(), modules, "demo", ScopeAll)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(files, "\n")
	if !strings.Contains(joined, "ui.ts") || !strings.Contains(joined, "App.vue") {
		t.Fatalf("missing roots: %v", files)
	}
	if strings.Contains(joined, "Skip.test.vue") {
		t.Fatalf("unexpected test vue: %v", files)
	}

	noVue, err := CollectRootFiles(t.Context(), modules, "demo", ScopeNoVue)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(noVue, "\n"), "App.vue") {
		t.Fatalf("ScopeNoVue must exclude vue: %v", noVue)
	}
}

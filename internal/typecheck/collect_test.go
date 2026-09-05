// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectRootFiles_Service(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	modules := filepath.Join(dir, "modules")
	app := filepath.Join(modules, "demo")
	mustMkdir(t, filepath.Join(app, "service", "nested"))
	mustMkdir(t, filepath.Join(app, "web"))
	mustMkdir(t, filepath.Join(app, "service", "tests"))
	mustWrite(t, filepath.Join(app, "index.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "service", "a.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "service", "a.test.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "service", "tests", "b.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "web", "ui.ts"), "export {};\n")
	mustWrite(t, filepath.Join(app, "service", "nested", "c.ts"), "export {};\n")

	files, err := CollectRootFiles(modules, "demo", ScopeService)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(files, "\n")
	for _, want := range []string{"index.ts", "a.ts", "c.ts"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %s in %v", want, files)
		}
	}
	for _, ban := range []string{"a.test.ts", "ui.ts", filepath.Join("tests", "b.ts")} {
		if strings.Contains(joined, ban) {
			t.Fatalf("unexpected %s in %v", ban, files)
		}
	}
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

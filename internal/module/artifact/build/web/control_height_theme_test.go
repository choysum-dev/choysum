// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestControlHeightThemeUtilitiesEmit locks the Dense Admin control ruler:
// product theme.css must emit h-control / min-h-control / size-control from --height-* / --size-*
// (Choysum's Go Tailwind dialect supports these namespaces; stock TW docs alone
// are not enough to assume they work).
func TestControlHeightThemeUtilitiesEmit(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	themePath := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..", "modules", "web", "web", "styles", "theme.css")
	dialect, err := os.ReadFile(themePath)
	if err != nil {
		t.Fatalf("read theme.css: %v", err)
	}
	css, _, err := GenerateTailwindCSS(string(dialect), []string{
		"h-control", "h-control-sm", "h-control-lg", "size-control", "min-h-control",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{".h-control", ".h-control-sm", ".h-control-lg", ".size-control", ".min-h-control"} {
		if !strings.Contains(css, needle) {
			t.Fatalf("expected %s in generated CSS from modules/web theme.css", needle)
		}
	}
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/evanw/esbuild/pkg/api"
)

func TestChoysumGeneratedPluginResolvesImport(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), ".choysum")
	outCSS := filepath.Join(home, "generated", "styles", choyTailwindGeneratedCSSName)
	if err := os.MkdirAll(filepath.Dir(outCSS), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outCSS, []byte("/* ok */\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	outdir := t.TempDir()
	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   `import "` + ChoyTailwindGeneratedImportPath + `"; export default 1;`,
			ResolveDir: t.TempDir(),
			Sourcefile: "entry.ts",
			Loader:     api.LoaderTS,
		},
		Bundle:   true,
		Write:    true,
		Outdir:   outdir,
		Plugins:  []api.Plugin{choysumGeneratedPlugin(home)},
		Format:   api.FormatESModule,
		Platform: api.PlatformBrowser,
		LogLevel: api.LogLevelSilent,
	})
	if len(result.Errors) > 0 {
		t.Fatalf("build errors: %v", result.Errors)
	}
}

func TestChoyTailwindGeneratedCSSPath(t *testing.T) {
	t.Parallel()
	if _, err := ChoyTailwindGeneratedCSSPath(""); err == nil {
		t.Fatal("expected empty path error")
	}
	got, err := ChoyTailwindGeneratedCSSPath(filepath.Join(t.TempDir(), ".choysum"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != choyTailwindGeneratedCSSName {
		t.Fatalf("basename = %q", filepath.Base(got))
	}
	if filepath.Base(filepath.Dir(got)) != "styles" {
		t.Fatalf("parent = %q", filepath.Dir(got))
	}
}

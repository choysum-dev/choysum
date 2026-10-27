// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"os"
	"path/filepath"
	"strings"
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
	const probe = ".choysum-gen-plugin-probe{color:#abcdef}"
	if err := os.WriteFile(outCSS, []byte(probe+"\n"), 0o644); err != nil {
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
		Metafile: true,
	})
	if len(result.Errors) > 0 {
		t.Fatalf("build errors: %v", result.Errors)
	}
	slashCSS := filepath.ToSlash(outCSS)
	if !strings.Contains(result.Metafile, slashCSS) && !strings.Contains(result.Metafile, strings.ReplaceAll(slashCSS, "/", "\\/")) {
		t.Fatalf("metafile missing resolved CSS path %q:\n%s", outCSS, result.Metafile)
	}
	found := false
	err := filepath.WalkDir(outdir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(body), "choysum-gen-plugin-probe") {
			found = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("bundle output missing probe rule from generated CSS")
	}
}

func TestChoysumGeneratedPluginRejectsEmptyRoot(t *testing.T) {
	t.Parallel()
	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   `import "` + ChoyTailwindGeneratedImportPath + `"; export default 1;`,
			ResolveDir: t.TempDir(),
			Sourcefile: "entry.ts",
			Loader:     api.LoaderTS,
		},
		Bundle:   true,
		Write:    false,
		Plugins:  []api.Plugin{choysumGeneratedPlugin("")},
		Format:   api.FormatESModule,
		Platform: api.PlatformBrowser,
		LogLevel: api.LogLevelSilent,
	})
	if len(result.Errors) == 0 {
		t.Fatal("expected error for empty defaultChoysumPath")
	}
	joined := fmtErrors(result.Errors)
	if !strings.Contains(joined, "defaultChoysumPath") {
		t.Fatalf("unexpected errors: %s", joined)
	}
}

func TestChoysumGeneratedPluginRejectsTraversal(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), ".choysum")
	if err := os.MkdirAll(filepath.Join(home, "generated"), 0o755); err != nil {
		t.Fatal(err)
	}
	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   `import "@choysum/generated/../../outside.css"; export default 1;`,
			ResolveDir: t.TempDir(),
			Sourcefile: "entry.ts",
			Loader:     api.LoaderTS,
		},
		Bundle:   true,
		Write:    false,
		Plugins:  []api.Plugin{choysumGeneratedPlugin(home)},
		Format:   api.FormatESModule,
		Platform: api.PlatformBrowser,
		LogLevel: api.LogLevelSilent,
	})
	if len(result.Errors) == 0 {
		t.Fatal("expected traversal error")
	}
	joined := fmtErrors(result.Errors)
	if !strings.Contains(joined, "escapes generated root") {
		t.Fatalf("unexpected errors: %s", joined)
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

func fmtErrors(msgs []api.Message) string {
	parts := make([]string, 0, len(msgs))
	for _, m := range msgs {
		parts = append(parts, m.Text)
	}
	return strings.Join(parts, "; ")
}

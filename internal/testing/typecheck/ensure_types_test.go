// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/esmresolver"
)

func TestEnsureTypeAssets_EarlyReturns(t *testing.T) {
	if err := ensureTypeAssets(nil, nil, "", "app"); err != nil {
		t.Fatal(err)
	}
	if err := ensureTypeAssets(context.Background(), nil, t.TempDir(), "  "); err != nil {
		t.Fatal(err)
	}

	modules := t.TempDir()
	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	vuePath := filepath.Join(home, "pkg", "types", "esm.sh_vue@1", "index.d.ts")
	makeDir(t, filepath.Dir(vuePath))
	writeFile(t, vuePath, "export {}\n")
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "vue": ["`+filepath.ToSlash(vuePath)+`"]
    }
  }
}
`)
	var stderr strings.Builder
	if err := ensureTypeAssets(context.Background(), &stderr, modules, "demo"); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected skip before fetch, stderr=%q", stderr.String())
	}
}

func TestEnsureTypeAssets_SkipWithoutEnv(t *testing.T) {
	t.Setenv("CHOYSUM_HOME", "")
	t.Setenv("CHOYSUM_TEST_TMP", "")
	modules := t.TempDir()
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{"compilerOptions":{"paths":{}}}`)
	if err := ensureTypeAssets(context.Background(), nil, modules, "demo"); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureTypeAssets_NoVuePathMapping(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	t.Setenv("CHOYSUM_TEST_TMP", "")
	modules := t.TempDir()
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "lodash": ["./missing.d.ts"]
    }
  }
}
`)
	var stderr strings.Builder
	if err := ensureTypeAssets(context.Background(), &stderr, modules, "demo"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "fetching critical type assets") {
		t.Fatalf("expected fetch notice, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "no resolvable vue version") {
		t.Fatalf("expected stub fallback notice, got %q", stderr.String())
	}
}

func TestEnsureTypeAssets_DiscoversVueFromTypesCache(t *testing.T) {
	origFetch := fetchTypeDefinition
	origClient := newTypeFetchHTTPClient
	t.Cleanup(func() {
		fetchTypeDefinition = origFetch
		newTypeFetchHTTPClient = origClient
	})
	newTypeFetchHTTPClient = func() *http.Client { return &http.Client{} }

	tmp := t.TempDir()
	t.Setenv("CHOYSUM_TEST_TMP", tmp)
	t.Setenv("CHOYSUM_HOME", "")
	typesDir := filepath.Join(tmp, "cache", "pkg", "types")
	makeDir(t, typesDir)
	writeCompleteVueGraph(t, typesDir, "3.5.35")

	modules := t.TempDir()
	// No tsconfig — mirrors CI shards after checkout (modules/tsconfig is gitignored).
	fetchTypeDefinition = func(_ context.Context, _ *http.Client, _, td, pkg, ver string) (*esmresolver.TypeFetchResult, []esmresolver.TypeFetchResult, error) {
		if pkg != "vue" || ver != "3.5.35" {
			t.Fatalf("unexpected fetch %s@%s", pkg, ver)
		}
		p := filepath.Join(td, "esm.sh_vue@3.5.35_dist_vue.d.mts.d.ts")
		return &esmresolver.TypeFetchResult{Package: "vue", Version: ver, CachedPath: p, FromCache: true}, nil, nil
	}

	var stderr strings.Builder
	if err := ensureTypeAssets(context.Background(), &stderr, modules, "auth"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "ensuring vue@3.5.35") {
		t.Fatalf("stderr=%q", stderr.String())
	}
	paths := readModuleTSConfigPaths(modules)
	if _, ok := paths["vue"]; !ok {
		t.Fatalf("expected vue path written into modules/tsconfig, got %#v", paths)
	}
}

func TestEnsureTypeAssets_CanceledContext(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	modules := t.TempDir()
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "vue": ["../../.choysum/pkg/types/esm.sh_vue@3.5.0/index.d.ts"]
    }
  }
}
`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ensureTypeAssets(ctx, nil, modules, "demo")
	if err == nil {
		t.Fatal("expected canceled error")
	}
}

func TestResolvePinnedPackageVersion(t *testing.T) {
	modules := t.TempDir()
	if got := resolvePinnedPackageVersion(modules, "vue"); got != "" {
		t.Fatalf("no tsconfig: %q", got)
	}

	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "vue": [
        "../../.choysum/pkg/types/esm.sh_vue@3.5.13/index.d.ts"
      ],
      "@types/node": [
        "../../.choysum/pkg/types/esm.sh_@types_node@22.20.1/index.d.ts"
      ],
      "other": ["./x.d.ts"]
    }
  }
}
`)
	if got := resolvePinnedPackageVersion(modules, "vue"); got != "3.5.13" {
		t.Fatalf("vue = %q", got)
	}
	if got := resolvePinnedTypesNodeVersion(modules); got != "22.20.1" {
		t.Fatalf("@types/node = %q", got)
	}
	if got := resolvePinnedPackageVersion(modules, "other"); got != "" {
		t.Fatalf("unversioned = %q", got)
	}
	if got := resolvePinnedPackageVersion(modules, "missing"); got != "" {
		t.Fatalf("missing = %q", got)
	}
}

func TestVueTypesEntryVerRE_DeclarationSuffix(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"esm.sh_vue@3.5.35.d.ts", "3.5.35"},
		{"esm.sh_vue@3.5.35_dist_vue.d.mts.d.ts", "3.5.35"},
		{"esm.sh_vue@3.5.35", "3.5.35"},
	}
	for _, tt := range cases {
		m := vueTypesEntryVerRE.FindStringSubmatch(tt.name)
		if len(m) != 2 || m[1] != tt.want {
			t.Fatalf("%s: got %#v want %q", tt.name, m, tt.want)
		}
	}
}

func TestResolvePinnedPackageVersion_TypesNodeFallbackRE(t *testing.T) {
	modules := t.TempDir()
	// Primary RE matches a wrong package first; @types/node fallback RE still wins.
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "@types/node": [
        "x/esm.sh_wrong@1.2.3/y/esm.sh_@types_node@18.19.0/z"
      ]
    }
  }
}
`)
	if got := resolvePinnedTypesNodeVersion(modules); got != "18.19.0" {
		t.Fatalf("got %q", got)
	}
}

func TestEnsureTypeAssets_FetchSuccessAndFailures(t *testing.T) {
	origFetch := fetchTypeDefinition
	origClient := newTypeFetchHTTPClient
	t.Cleanup(func() {
		fetchTypeDefinition = origFetch
		newTypeFetchHTTPClient = origClient
	})
	newTypeFetchHTTPClient = func() *http.Client { return &http.Client{} }

	setup := func(t *testing.T) (home, modules string) {
		t.Helper()
		home = t.TempDir()
		t.Setenv("CHOYSUM_HOME", home)
		t.Setenv("CHOYSUM_TEST_TMP", "")
		modules = t.TempDir()
		writeFile(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "vue": ["../../.choysum/pkg/types/esm.sh_vue@3.5.0_dist_vue.d.mts.d.ts"],
      "@types/node": ["../../.choysum/pkg/types/esm.sh_@types_node@22.20.1_index.d.ts.d.ts"]
    }
  }
}
`)
		return home, modules
	}

	t.Run("vue fetch error", func(t *testing.T) {
		_, modules := setup(t)
		fetchTypeDefinition = func(context.Context, *http.Client, string, string, string, string) (*esmresolver.TypeFetchResult, []esmresolver.TypeFetchResult, error) {
			return nil, nil, errors.New("network down")
		}
		err := ensureTypeAssets(context.Background(), io.Discard, modules, "demo")
		if err == nil || !strings.Contains(err.Error(), "fetch vue@3.5.0") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("node fetch error", func(t *testing.T) {
		_, modules := setup(t)
		fetchTypeDefinition = func(_ context.Context, _ *http.Client, _, typesDir, pkg, _ string) (*esmresolver.TypeFetchResult, []esmresolver.TypeFetchResult, error) {
			if pkg == "vue" {
				p := filepath.Join(typesDir, "esm.sh_vue@3.5.0_dist_vue.d.mts.d.ts")
				if err := os.WriteFile(p, []byte("export {}\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				return &esmresolver.TypeFetchResult{CachedPath: p}, nil, nil
			}
			return nil, nil, errors.New("node fail")
		}
		err := ensureTypeAssets(context.Background(), nil, modules, "demo")
		if err == nil || !strings.Contains(err.Error(), "fetch @types/node") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		home, modules := setup(t)
		fetchTypeDefinition = func(_ context.Context, _ *http.Client, _, typesDir, pkg, _ string) (*esmresolver.TypeFetchResult, []esmresolver.TypeFetchResult, error) {
			switch pkg {
			case "vue":
				p := filepath.Join(typesDir, "esm.sh_vue@3.5.0_dist_vue.d.mts.d.ts")
				if err := os.WriteFile(p, []byte("export {}\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				// Transitive siblings required for HasResolvableVueTypes.
				for _, name := range []string{
					"esm.sh_@vue_runtime-dom@3.5.0_dist_runtime-dom.d.ts.d.ts",
					"esm.sh_@vue_runtime-core@3.5.0_dist_runtime-core.d.ts.d.ts",
					"esm.sh_@vue_reactivity@3.5.0_dist_reactivity.d.ts.d.ts",
				} {
					body := "export {}\n"
					if strings.Contains(name, "runtime-core") {
						body = "export type PropType<T> = any;\ndeclare function h(...args: any[]): any;\n"
					}
					if strings.Contains(name, "reactivity") {
						body = "export declare function toRef(...args: any[]): any;\n"
					}
					if err := os.WriteFile(filepath.Join(typesDir, name), []byte(body), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				return &esmresolver.TypeFetchResult{CachedPath: p}, nil, nil
			case "@types/node":
				p := filepath.Join(typesDir, "esm.sh_@types_node@22.20.1_index.d.ts.d.ts")
				if err := os.WriteFile(p, []byte("export {}\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				return &esmresolver.TypeFetchResult{CachedPath: p}, nil, nil
			default:
				return nil, nil, nil
			}
		}
		var stderr strings.Builder
		if err := ensureTypeAssets(context.Background(), &stderr, modules, "demo"); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stderr.String(), "fetching critical type assets") {
			t.Fatalf("stderr=%q", stderr.String())
		}
		bridge := filepath.Join(home, "pkg", "types", "typeRoots", "node", "index.d.ts")
		if _, err := os.Stat(bridge); err != nil {
			t.Fatalf("bridge: %v", err)
		}
	})

	t.Run("vue still missing after fetch", func(t *testing.T) {
		home, modules := setup(t)
		fetchTypeDefinition = func(context.Context, *http.Client, string, string, string, string) (*esmresolver.TypeFetchResult, []esmresolver.TypeFetchResult, error) {
			// Pretend fetch succeeded but wrote nothing resolvable.
			return &esmresolver.TypeFetchResult{CachedPath: filepath.Join(home, "pkg", "types", "wrong-name.d.ts")}, nil, nil
		}
		err := ensureTypeAssets(context.Background(), nil, modules, "demo")
		if err == nil || !strings.Contains(err.Error(), "type-fetch entry missing") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestEnsureNodeCompilerTypes_FetchBranches(t *testing.T) {
	orig := fetchTypeDefinition
	t.Cleanup(func() { fetchTypeDefinition = orig })

	modules := t.TempDir()
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "@types/node": ["../../.choysum/pkg/types/esm.sh_@types_node@22.20.1_index.d.ts.d.ts"]
    }
  }
}
`)
	typesDir := t.TempDir()

	fetchTypeDefinition = func(context.Context, *http.Client, string, string, string, string) (*esmresolver.TypeFetchResult, []esmresolver.TypeFetchResult, error) {
		return nil, nil, nil
	}
	if err := ensureNodeCompilerTypes(context.Background(), &http.Client{}, "https://esm.sh", typesDir, modules); err != nil {
		t.Fatal(err)
	}

	fetchTypeDefinition = func(context.Context, *http.Client, string, string, string, string) (*esmresolver.TypeFetchResult, []esmresolver.TypeFetchResult, error) {
		return &esmresolver.TypeFetchResult{CachedPath: "   "}, nil, nil
	}
	if err := ensureNodeCompilerTypes(context.Background(), &http.Client{}, "https://esm.sh", typesDir, modules); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureTypeAssets_MkdirFail(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, "pkg"), "not-a-directory\n")
	t.Setenv("CHOYSUM_HOME", home)
	t.Setenv("CHOYSUM_TEST_TMP", "")
	modules := t.TempDir()
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{"compilerOptions":{"paths":{}}}`)
	err := ensureTypeAssets(context.Background(), nil, modules, "demo")
	if err == nil || !strings.Contains(err.Error(), "create types dir") {
		t.Fatalf("err = %v", err)
	}
}

func TestEnsureTypeAssets_EmptyPreferWriteDir(t *testing.T) {
	t.Setenv("CHOYSUM_HOME", t.TempDir())
	t.Setenv("CHOYSUM_TEST_TMP", "")
	orig := preferTypesWriteDir
	t.Cleanup(func() { preferTypesWriteDir = orig })
	preferTypesWriteDir = func() string { return "" }

	modules := t.TempDir()
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{
  "compilerOptions": {
    "paths": {
      "vue": ["../../.choysum/pkg/types/esm.sh_vue@3.5.0/index.d.ts"]
    }
  }
}
`)
	err := ensureTypeAssets(context.Background(), nil, modules, "demo")
	if err == nil || !strings.Contains(err.Error(), "cannot resolve type-fetch write dir") {
		t.Fatalf("err = %v", err)
	}
}

func TestEnsureNodeCompilerTypes_NoPinnedVersion(t *testing.T) {
	modules := t.TempDir()
	writeFile(t, filepath.Join(modules, "tsconfig.json"), `{"compilerOptions":{"paths":{}}}`)
	if err := ensureNodeCompilerTypes(context.Background(), &http.Client{}, "https://esm.sh", t.TempDir(), modules); err != nil {
		t.Fatal(err)
	}
}

func TestWriteTypeRootBridge_PathHookErrors(t *testing.T) {
	origAbs, origRel := filepathAbs, filepathRel
	t.Cleanup(func() {
		filepathAbs = origAbs
		filepathRel = origRel
	})

	typesDir := t.TempDir()
	cached := filepath.Join(typesDir, "esm.sh_node@1", "index.d.ts")
	makeDir(t, filepath.Dir(cached))
	writeFile(t, cached, "export {}\n")

	filepathAbs = func(string) (string, error) { return "", errors.New("abs types dir") }
	if err := writeTypeRootBridge(typesDir, "node", cached); err == nil || !strings.Contains(err.Error(), "abs types dir") {
		t.Fatalf("typesDir abs err = %v", err)
	}

	filepathAbs = filepath.Abs
	relName := "esm.sh_node@1/index.d.ts"
	filepathAbs = func(p string) (string, error) {
		if p == relName {
			return "", errors.New("abs cached")
		}
		return filepath.Abs(p)
	}
	if err := writeTypeRootBridge(typesDir, "node", relName); err == nil || !strings.Contains(err.Error(), "abs cached") {
		t.Fatalf("cached abs err = %v", err)
	}

	filepathAbs = filepath.Abs
	filepathRel = func(string, string) (string, error) { return "", errors.New("rel bridge") }
	if err := writeTypeRootBridge(typesDir, "node", cached); err == nil || !strings.Contains(err.Error(), "rel bridge") {
		t.Fatalf("rel err = %v", err)
	}
}

func TestWriteTypeRootBridge(t *testing.T) {
	if err := writeTypeRootBridge(t.TempDir(), "", "/x"); err != nil {
		t.Fatal(err)
	}
	if err := writeTypeRootBridge(t.TempDir(), "node", ""); err != nil {
		t.Fatal(err)
	}

	typesDir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.d.ts")
	writeFile(t, outside, "export {}\n")
	if err := writeTypeRootBridge(typesDir, "node", outside); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(typesDir, "typeRoots", "node", "index.d.ts")); !os.IsNotExist(err) {
		t.Fatal("outside cache must not create bridge")
	}

	cached := filepath.Join(typesDir, "esm.sh_@types_node@1", "index.d.ts")
	makeDir(t, filepath.Dir(cached))
	writeFile(t, cached, "export {}\n")
	if err := writeTypeRootBridge(typesDir, "node", cached); err != nil {
		t.Fatal(err)
	}
	bridge := filepath.Join(typesDir, "typeRoots", "node", "index.d.ts")
	data, err := os.ReadFile(bridge)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, `compilerOptions.types="node"`) {
		t.Fatalf("body = %s", body)
	}
	if !strings.Contains(body, "reference path=") {
		t.Fatalf("missing reference: %s", body)
	}

	// Relative cached path under typesDir.
	relCached := "esm.sh_@types_node@1/index.d.ts"
	if err := writeTypeRootBridge(typesDir, "node", relCached); err != nil {
		t.Fatal(err)
	}

	// Cached file inside typeRoots/node so Rel has no leading "." → "./" prefix.
	inner := filepath.Join(typesDir, "typeRoots", "node", "inner.d.ts")
	writeFile(t, inner, "export {}\n")
	if err := writeTypeRootBridge(typesDir, "node", inner); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(bridge)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `path="./inner.d.ts"`) {
		t.Fatalf("expected ./inner.d.ts bridge, got %s", data)
	}

	// MkdirAll fails when typeRoots is a file.
	bad := t.TempDir()
	writeFile(t, filepath.Join(bad, "typeRoots"), "file\n")
	nested := filepath.Join(bad, "cached.d.ts")
	writeFile(t, nested, "export {}\n")
	if err := writeTypeRootBridge(bad, "node", nested); err == nil {
		t.Fatal("expected mkdir failure")
	}
}

func writeCompleteVueGraph(t *testing.T, typesDir, ver string) {
	t.Helper()
	entry := filepath.Join(typesDir, fmt.Sprintf("esm.sh_vue@%s_dist_vue.d.mts.d.ts", ver))
	writeFile(t, entry, "export * from './runtime-dom';\n")
	for _, name := range []string{
		fmt.Sprintf("esm.sh_@vue_runtime-dom@%s_dist_runtime-dom.d.ts.d.ts", ver),
		fmt.Sprintf("esm.sh_@vue_runtime-core@%s_dist_runtime-core.d.ts.d.ts", ver),
		fmt.Sprintf("esm.sh_@vue_reactivity@%s_dist_reactivity.d.ts.d.ts", ver),
	} {
		body := "export {}\n"
		if strings.Contains(name, "runtime-core") {
			body = "export type PropType<T> = any;\ndeclare function h(...args: any[]): any;\n"
		}
		if strings.Contains(name, "reactivity") {
			body = "export declare function toRef(...args: any[]): any;\n"
		}
		writeFile(t, filepath.Join(typesDir, name), body)
	}
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/esmresolver"
	gonative "github.com/choysum-dev/choysum/internal/typecheck"
	"github.com/choysum-dev/choysum/pkg/config"
	xfmt "golang.org/x/exp/errors/fmt"
)

var (
	typesNodeVersionRE = regexp.MustCompile(`esm\.sh_@types_node@([^/_]+)`)
	esmShPkgVersionRE  = regexp.MustCompile(`(?:^|/)esm\.sh_(.+?)@([^/_]+)`)
)

// ensureTypeAssets downloads critical type-fetch .d.ts files (vue + @types/node)
// when modules/tsconfig cannot resolve `vue`. It intentionally does not walk the
// full module depends closure — packages like @vicons/material have thousands of
// transitive declaration files and would make typecheck cold-start untenable.
//
// Committed modules/tsconfig path mappings are left unchanged; gonative rewrite
// remaps ../../.choysum/pkg/types/… onto CHOYSUM_HOME / CHOYSUM_TEST_TMP caches.
func ensureTypeAssets(ctx context.Context, stderr io.Writer, modulesRoot, app string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	modulesRoot = filepath.Clean(modulesRoot)
	app = strings.TrimSpace(app)
	if modulesRoot == "" || app == "" {
		return nil
	}
	if gonative.HasResolvableVueTypes(modulesRoot, "") {
		return nil
	}
	// Fixtures call TypecheckApp without CLI harness env; keep stub/vue-less
	// behavior there. Real `choysum test typecheck` sets CHOYSUM_HOME (run home)
	// and/or CHOYSUM_TEST_TMP (CI cache).
	if strings.TrimSpace(os.Getenv("CHOYSUM_TEST_TMP")) == "" && strings.TrimSpace(os.Getenv("CHOYSUM_HOME")) == "" {
		return nil
	}

	typesDir := gonative.PreferTypesWriteDir()
	if typesDir == "" {
		return xfmt.Errorf("typecheck: cannot resolve type-fetch write dir (set CHOYSUM_HOME or CHOYSUM_TEST_TMP)")
	}
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		return xfmt.Errorf("typecheck: create types dir: %w", err)
	}

	if stderr != nil {
		_, _ = fmt.Fprintf(stderr, "# typecheck %s: fetching critical type assets into %s\n", app, typesDir)
	}

	upstream := config.DefaultESMUpstreamURL
	client := esmresolver.NewTypeFetchHTTPClient(30 * time.Second)
	if transport, ok := client.Transport.(*http.Transport); ok {
		defer transport.CloseIdleConnections()
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	vueVersion := resolvePinnedPackageVersion(modulesRoot, "vue")
	if vueVersion == "" {
		// Fixture / non-Vue apps: no vue path mapping to fetch.
		return nil
	}
	if _, _, err := esmresolver.FetchTypeDefinition(client, upstream, typesDir, "vue", vueVersion); err != nil {
		return xfmt.Errorf("typecheck: fetch vue@%s: %w", vueVersion, err)
	}
	if err := ensureNodeCompilerTypes(client, upstream, typesDir, modulesRoot); err != nil {
		return err
	}

	if !gonative.HasResolvableVueTypes(modulesRoot, "") {
		return xfmt.Errorf("typecheck: vue types still missing after type-fetch into %s (run: go run . type-fetch %s)", typesDir, app)
	}
	return nil
}

func ensureNodeCompilerTypes(client *http.Client, upstream, typesDir, modulesRoot string) error {
	version := resolvePinnedTypesNodeVersion(modulesRoot)
	if version == "" {
		return nil
	}
	result, _, err := esmresolver.FetchTypeDefinition(client, upstream, typesDir, "@types/node", version)
	if err != nil {
		return xfmt.Errorf("typecheck: fetch @types/node@%s: %w", version, err)
	}
	if result == nil || strings.TrimSpace(result.CachedPath) == "" {
		return nil
	}
	return writeTypeRootBridge(typesDir, "node", result.CachedPath)
}

func resolvePinnedTypesNodeVersion(modulesRoot string) string {
	return resolvePinnedPackageVersion(modulesRoot, "@types/node")
}

func resolvePinnedPackageVersion(modulesRoot, pkgName string) string {
	paths := readModuleTSConfigPaths(modulesRoot)
	entries, ok := paths[pkgName]
	if !ok {
		return ""
	}
	wantKey := strings.ReplaceAll(pkgName, "/", "_")
	for _, entry := range entries {
		entry = filepath.ToSlash(entry)
		if m := esmShPkgVersionRE.FindStringSubmatch(entry); len(m) == 3 {
			if m[1] == wantKey || m[1] == pkgName {
				return m[2]
			}
		}
		// @types/node fallback: esm.sh_@types_node@22.20.1_…
		if pkgName == "@types/node" {
			if m := typesNodeVersionRE.FindStringSubmatch(entry); len(m) == 2 {
				return m[1]
			}
		}
	}
	return ""
}

func writeTypeRootBridge(typesDir, typeName, cachedPath string) error {
	typeName = strings.TrimSpace(typeName)
	cachedPath = filepath.Clean(strings.TrimSpace(cachedPath))
	if typeName == "" || cachedPath == "" {
		return nil
	}
	absTypesDir, err := filepath.Abs(typesDir)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(cachedPath) {
		cachedPath, err = filepath.Abs(cachedPath)
		if err != nil {
			return err
		}
	}
	if !strings.HasPrefix(cachedPath, absTypesDir+string(os.PathSeparator)) {
		return nil
	}
	typePkgDir := filepath.Join(absTypesDir, "typeRoots", typeName)
	if err := os.MkdirAll(typePkgDir, 0o755); err != nil {
		return err
	}
	relCachedPath, err := filepath.Rel(typePkgDir, cachedPath)
	if err != nil {
		return err
	}
	relCachedPath = filepath.ToSlash(relCachedPath)
	if !strings.HasPrefix(relCachedPath, ".") {
		relCachedPath = "./" + relCachedPath
	}
	content := fmt.Sprintf("// Generated by typecheck for compilerOptions.types=%q.\n/// <reference path=%q />\n", typeName, relCachedPath)
	return os.WriteFile(filepath.Join(typePkgDir, "index.d.ts"), []byte(content), 0o644)
}

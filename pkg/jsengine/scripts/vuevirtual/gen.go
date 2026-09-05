// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

//go:build ignore

// gen.go builds the vuevirtual embeddable script (dist/index.js).
// Prefer local node_modules (npm install in this directory) because
// @vue/language-core + typescript are large and often fail on esm.sh.
// Falls back to the choysum-esm-resolver CDN plugin when node_modules is absent.
//
// Invoke via: go generate ./pkg/jsengine/scripts/vuevirtual/...

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/choysum-dev/choysum/internal/esmresolver"
	"github.com/evanw/esbuild/pkg/api"
)

func main() {
	cacheDir := os.Getenv("CHOYSUM_HOME")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "gen.go: cannot determine home directory: %v\n", err)
			os.Exit(1)
		}
		cacheDir = filepath.Join(home, ".choysum")
	}

	entryPoint := "src/index.ts"
	outFile := "dist/index.js"
	useLocal := fileExists("node_modules/@vue/language-core/package.json") &&
		fileExists("node_modules/typescript/package.json")

	opts := api.BuildOptions{
		EntryPoints:       []string{entryPoint},
		Outfile:           outFile,
		Bundle:            true,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		TreeShaking:       api.TreeShakingTrue,
		Format:            api.FormatIIFE,
		GlobalName:        "vuevirtual",
		Platform:          api.PlatformBrowser,
		Banner: map[string]string{
			"js": "if(typeof require==='undefined'){globalThis.require=function(m){return null;};}",
		},
		Alias: map[string]string{
			"path": "path-browserify",
		},
		Write: true,
	}

	if useLocal {
		fmt.Println("vuevirtual: bundling from local node_modules")
		opts.AbsWorkingDir, _ = os.Getwd()
	} else {
		fmt.Println("vuevirtual: bundling via esm.sh resolver (no local node_modules)")
		client := esmresolver.NewTypeFetchHTTPClient(30 * time.Second)
		typesDir := filepath.Join(cacheDir, "pkg", "types")
		if results, err := esmresolver.FetchTypesForModule(client, "https://esm.sh", typesDir, "."); err == nil {
			if err := esmresolver.UpdateTsconfigPaths("tsconfig.json", results); err != nil {
				fmt.Fprintf(os.Stderr, "gen.go: warning: %v\n", err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "gen.go: warning: failed to fetch types: %v\n", err)
		}
		opts.Plugins = []api.Plugin{
			esmresolver.New(
				esmresolver.WithCacheDir(cacheDir),
				esmresolver.WithTarget("es2020"),
				esmresolver.WithModulePath("."),
			).Plugin(),
		}
	}

	result := api.Build(opts)
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			loc := ""
			if e.Location != nil {
				loc = fmt.Sprintf("%s:%d:%d ", e.Location.File, e.Location.Line, e.Location.Column)
			}
			fmt.Fprintf(os.Stderr, "gen.go: %s%s\n", loc, e.Text)
		}
		if !useLocal {
			fmt.Fprintf(os.Stderr, "gen.go: tip: cd pkg/jsengine/scripts/vuevirtual && npm install\n")
		}
		os.Exit(1)
	}

	info, err := os.Stat(outFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen.go: stat %s: %v\n", outFile, err)
		os.Exit(1)
	}
	fmt.Printf("vuevirtual: built %s (%d bytes)\n", outFile, info.Size())
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

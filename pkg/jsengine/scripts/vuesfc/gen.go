// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

//go:build ignore

// gen.go builds the vuesfc embeddable script (dist/index.js) using the
// Go esbuild API with the choysum-esm-resolver plugin. All bare imports
// are resolved through the ESM CDN with local caching — no node_modules
// required.
//
// Invoke via: go generate ./pkg/jsengine/scripts/vuesfc/...

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

	// 1. Fetch Types and Update tsconfig.json for IDE
	client := esmresolver.NewTypeFetchHTTPClient(30 * time.Second)
	typesDir := filepath.Join(cacheDir, "pkg", "types")
	if results, err := esmresolver.FetchTypesForModule(client, "https://esm.sh", typesDir, "."); err == nil {
		if err := esmresolver.UpdateTsconfigPaths("tsconfig.json", results); err != nil {
			fmt.Fprintf(os.Stderr, "gen.go: warning: %v\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "gen.go: warning: failed to fetch types: %v\n", err)
	}

	// Resolve the vuesfc source directory relative to this file's location.
	// When invoked via go generate, the working directory is the package
	// directory (pkg/jsengine/scripts/vuesfc/).
	entryPoint := "src/index.ts"
	outFile := "dist/index.js"

	result := api.Build(api.BuildOptions{
		EntryPoints:       []string{entryPoint},
		Outfile:           outFile,
		Bundle:            true,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		TreeShaking:       api.TreeShakingTrue,
		Format:            api.FormatIIFE,
		GlobalName:        "sfc",
		Platform:          api.PlatformBrowser,
		// Provide a global require stub so that esm.sh CJS interop wrappers
		// (which check typeof require<"u") resolve to a safe no-op instead
		// of throwing "Dynamic require of ... is not supported" in QuickJS.
		Banner: map[string]string{"js": "var require=function(m){return null;};"},
		Plugins: []api.Plugin{
			esmresolver.New(
				esmresolver.WithCacheDir(cacheDir),
				esmresolver.WithTarget("es2020"),
				esmresolver.WithModulePath("."),
			).Plugin(),
		},
		Write: true,
	})

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			loc := ""
			if e.Location != nil {
				loc = fmt.Sprintf("%s:%d:%d ", e.Location.File, e.Location.Line, e.Location.Column)
			}
			fmt.Fprintf(os.Stderr, "gen.go: %s%s\n", loc, e.Text)
		}
		os.Exit(1)
	}

	fmt.Println("vuesfc: built dist/index.js")
}

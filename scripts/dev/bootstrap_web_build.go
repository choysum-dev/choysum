// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

//go:build ignore

// bootstrap_web_build.go builds internal/bootstrap/web/dist using only Go
// tooling (esbuild + esmresolver + vueplugin). No Node.js, npm, or Vite
// required.
//
// Invoke via: go run ./scripts/dev/bootstrap_web_build.go

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "github.com/choysum-dev/choysum/internal/defaultengine"
	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	"github.com/choysum-dev/choysum/internal/esmresolver"
	"github.com/choysum-dev/choysum/internal/vueplugin"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
)

// buildScope implements scope.Scope and scope.FactoryInputCarrier for a
// standalone, non-DB build run. It provides only the configuration needed
// for JS engine resolution and executor startup.
type buildScope struct {
	ctx context.Context
	cfg *config.Config
}

func (s *buildScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *buildScope) Session() *scope.Session              { return nil }
func (s *buildScope) Transactor() scope.Transactor         { return nil }
func (s *buildScope) WithContext(ctx context.Context) scope.Scope {
	clone := *s
	clone.ctx = ctx
	return &clone
}
func (s *buildScope) Context() context.Context { return s.ctx }
func (s *buildScope) Logger() *slog.Logger     { return slog.Default() }

// FactoryInput exposes enough config for ServerRuntimeOptionsFromScope to
// return JsEngineFactory="quickjs" so the default engine factory can be
// resolved during executor creation.
func (s *buildScope) FactoryInput() scope.FactoryInput {
	if s.cfg == nil {
		return nil
	}
	return &buildFactoryInput{cfg: s.cfg}
}

type buildFactoryInput struct{ cfg *config.Config }

func (i *buildFactoryInput) Environment() string                  { return "" }
func (i *buildFactoryInput) ModulesPath() string                  { return "" }
func (i *buildFactoryInput) DistPath() string                     { return "" }
func (i *buildFactoryInput) TmpPath() string                      { return "" }
func (i *buildFactoryInput) DefaultChoysumPath() string           { return "" }
func (i *buildFactoryInput) ConfigPath() string                   { return "" }
func (i *buildFactoryInput) ESMUpstreamURL() string               { return "" }
func (i *buildFactoryInput) NpmRegistryURL() string               { return "" }
func (i *buildFactoryInput) ModuleCatalogIndexURL() string        { return "" }
func (i *buildFactoryInput) CompileConfig() *config.CompileConfig { return nil }
func (i *buildFactoryInput) AuthConfig() *config.AuthConfig       { return nil }
func (i *buildFactoryInput) TaskConfig() *config.TaskConfig       { return nil }
func (i *buildFactoryInput) LogConfig() *config.LogConfig         { return nil }
func (i *buildFactoryInput) ServerConfig() *config.ServerConfig   { return i.cfg.Server }

func main() {
	start := time.Now()

	cacheDir := resolveCacheDir()
	bootstrapDir := "internal/bootstrap/web"
	srcDir := filepath.Join(bootstrapDir, "src")
	distDir := filepath.Join(bootstrapDir, "dist")
	assetsDir := filepath.Join(distDir, "assets")

	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap_web_build: mkdir assets: %v\n", err)
		os.Exit(1)
	}

	// ----- 1. Fetch type definitions and update tsconfig for IDE support -----
	client := esmresolver.NewTypeFetchHTTPClient(30 * time.Second)
	typesDir := filepath.Join(cacheDir, "pkg", "types")
	if results, err := esmresolver.FetchTypesForModule(client, "https://esm.sh", typesDir, bootstrapDir); err == nil {
		tsConfigPath := filepath.Join(bootstrapDir, "tsconfig.json")
		if err := esmresolver.UpdateTsconfigPaths(tsConfigPath, results); err != nil {
			fmt.Fprintf(os.Stderr, "bootstrap_web_build: warning: update tsconfig paths: %v\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "bootstrap_web_build: warning: type fetch: %v\n", err)
	}

	// ----- 2. Create compiler executor (minimal scope, QuickJS engine) -----
	cfg := &config.Config{
		Server: &config.ServerConfig{
			JsEngineFactory:   "quickjs",
			JsExecutorFactory: "default",
		},
	}
	runtimeScope := &buildScope{ctx: context.Background(), cfg: cfg}

	executor, err := jsexecutor.NewCompilerExecutor(runtimeScope)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap_web_build: create compiler executor: %v\n", err)
		os.Exit(1)
	}
	if err := executor.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap_web_build: start compiler executor: %v\n", err)
		os.Exit(1)
	}
	defer executor.Stop()

	// ----- 3. Build with esbuild + esmresolver + vueplugin -----
	entryPoint := filepath.Join(srcDir, "main.ts")
	indexHTML := filepath.Join(distDir, "index.html")
	sourceHTML := filepath.Join(bootstrapDir, "index.html")

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entryPoint},
		Outdir:      assetsDir,
		PublicPath:  "/bootstrap/assets",
		Bundle:      true,
		Format:      api.FormatESModule,
		Splitting:   true,
		Metafile:    true,
		Write:       true,
		EntryNames:  "[dir]/[name]-[hash]",
		Platform:    api.PlatformBrowser,
		Target:      api.ES2020,
		Loader: map[string]api.Loader{
			".png":  api.LoaderFile,
			".svg":  api.LoaderFile,
			".css":  api.LoaderCSS,
			".scss": api.LoaderCSS,
			".sass": api.LoaderCSS,
		},
		Plugins: []api.Plugin{
			esmresolver.New(
				esmresolver.WithCacheDir(cacheDir),
				esmresolver.WithTarget("es2020"),
				esmresolver.WithModulePath(bootstrapDir),
			).Plugin(),
			vueplugin.NewPlugin(
				vueplugin.WithJsExecutor(executor),
				vueplugin.WithIndexHtmlOptions(vueplugin.IndexHtmlOptions{
					SourceFile: sourceHTML,
					OutFile:    indexHTML,
					RemoveTagXPaths: []string{
						`//script[@src="/src/main.ts"]`,
					},
				}),
			),
		},
		Define: map[string]string{
			"import.meta.env.MODE": "'production'",
			"import.meta.env.PROD": "true",
			"import.meta.env.DEV":  "false",
			"import.meta.env.SSR":  "false",
		},
	})

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			loc := ""
			if e.Location != nil {
				loc = fmt.Sprintf("%s:%d:%d ", e.Location.File, e.Location.Line, e.Location.Column)
			}
			fmt.Fprintf(os.Stderr, "bootstrap_web_build: %s%s\n", loc, e.Text)
		}
		os.Exit(1)
	}

	fmt.Printf("bootstrap_web_build: dist ready (elapsed: %s)\n", time.Since(start).Round(time.Millisecond))
}

func resolveCacheDir() string {
	if dir := os.Getenv("CHOYSUM_HOME"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap_web_build: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".choysum")
}

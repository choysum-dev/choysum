// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

//go:build ignore

// gen.go generates bootstrap web assets in place using only Go tooling.
// It performs two steps:
// 1. Generate bootstrap_pb.ts from bootstrap.proto via gots.
// 2. Build dist assets and index.html via esbuild + esmresolver + vueplugin.
//
// Invoke via: go generate ./internal/bootstrap/web/...

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bufbuild/protocompile"
	_ "github.com/choysum-dev/choysum/internal/defaultengine"
	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	"github.com/choysum-dev/choysum/internal/esmresolver"
	"github.com/choysum-dev/choysum/internal/module/artifact/generate/grpcwebplugin/gots"
	"github.com/choysum-dev/choysum/internal/vueplugin"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/pluginpb"
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

	bootstrapDir, repoRoot, err := resolvePaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap_web_generate: resolve paths: %v\n", err)
		os.Exit(1)
	}

	if err := generateBootstrapProto(repoRoot, bootstrapDir); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap_web_generate: proto generation: %v\n", err)
		os.Exit(1)
	}

	cacheDir := resolveCacheDir()
	if err := buildBootstrapDist(cacheDir, bootstrapDir); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap_web_generate: dist build: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("bootstrap_web_generate: completed (elapsed: %s)\n", time.Since(start).Round(time.Millisecond))
}

func resolvePaths() (bootstrapDir string, repoRoot string, err error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", "", fmt.Errorf("cannot resolve caller path")
	}
	bootstrapDir = filepath.Dir(thisFile)
	repoRoot = filepath.Clean(filepath.Join(bootstrapDir, "..", "..", ".."))
	return bootstrapDir, repoRoot, nil
}

func generateBootstrapProto(repoRoot, bootstrapDir string) error {
	protoRelPath := filepath.ToSlash(filepath.Join("internal", "bootstrap", "proto", "bootstrap.proto"))
	protoDir := filepath.Join(repoRoot, "internal", "bootstrap", "proto")

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{repoRoot, protoDir},
		}),
	}

	fds, err := compiler.Compile(context.Background(), protoRelPath)
	if err != nil {
		return fmt.Errorf("compile %s: %w", protoRelPath, err)
	}

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{protoRelPath},
		Parameter:      proto.String("target=ts"),
	}
	for _, fd := range fds {
		req.ProtoFile = append(req.ProtoFile, protodesc.ToFileDescriptorProto(fd))
	}

	gen := gots.NewGenerator()
	resp, err := gen.Generate(req)
	if err != nil {
		return fmt.Errorf("gots generate: %w", err)
	}

	outDir := filepath.Join(
		bootstrapDir,
		"src",
		"gen",
		"bootstrap",
		"internal",
		"bootstrap",
		"proto",
	)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	spdxHeader := `// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

`

	for _, file := range resp.GetFile() {
		outPath := filepath.Join(outDir, filepath.Base(file.GetName()))
		content := spdxHeader + file.GetContent()
		if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", outPath, err)
		}
		fmt.Printf("bootstrap_web_generate: wrote %s\n", outPath)
	}

	return nil
}

func buildBootstrapDist(cacheDir, bootstrapDir string) error {
	srcDir := filepath.Join(bootstrapDir, "src")
	distDir := filepath.Join(bootstrapDir, "dist")
	assetsDir := filepath.Join(distDir, "assets")

	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return fmt.Errorf("mkdir assets: %w", err)
	}

	// Fetch type definitions and update tsconfig for IDE support.
	client := esmresolver.NewTypeFetchHTTPClient(30 * time.Second)
	typesDir := filepath.Join(cacheDir, "pkg", "types")
	if results, err := esmresolver.FetchTypesForModule(client, "https://esm.sh", typesDir, bootstrapDir); err == nil {
		tsConfigPath := filepath.Join(bootstrapDir, "tsconfig.json")
		if err := esmresolver.UpdateTsconfigPaths(tsConfigPath, results); err != nil {
			fmt.Fprintf(os.Stderr, "bootstrap_web_generate: warning: update tsconfig paths: %v\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "bootstrap_web_generate: warning: type fetch: %v\n", err)
	}

	cfg := &config.Config{
		Server: &config.ServerConfig{
			JsEngineFactory:   "quickjs",
			JsExecutorFactory: "default",
		},
	}
	runtimeScope := &buildScope{ctx: context.Background(), cfg: cfg}

	executor, err := jsexecutor.NewCompilerExecutor(runtimeScope)
	if err != nil {
		return fmt.Errorf("create compiler executor: %w", err)
	}
	if err := executor.Start(); err != nil {
		return fmt.Errorf("start compiler executor: %w", err)
	}
	defer executor.Stop()

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
		var b strings.Builder
		for _, e := range result.Errors {
			loc := ""
			if e.Location != nil {
				loc = fmt.Sprintf("%s:%d:%d ", e.Location.File, e.Location.Line, e.Location.Column)
			}
			msg := fmt.Sprintf("%s%s", loc, e.Text)
			fmt.Fprintf(os.Stderr, "bootstrap_web_generate: %s\n", msg)
			if b.Len() > 0 {
				b.WriteString("; ")
			}
			b.WriteString(msg)
		}
		return fmt.Errorf("esbuild failed: %s", b.String())
	}

	return nil
}

func resolveCacheDir() string {
	if dir := os.Getenv("CHOYSUM_HOME"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap_web_generate: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".choysum")
}

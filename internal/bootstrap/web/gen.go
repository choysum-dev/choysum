// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

//go:build ignore

// gen.go builds the bootstrap web dist assets (dist/) using the Go esbuild API
// with the choysum-esm-resolver plugin and the vueplugin for Vue SFC compilation.
// All bare imports are resolved through the ESM CDN with local caching — no
// node_modules required.
//
// The build script also regenerates the Connect-Web TypeScript client stubs
// from internal/bootstrap/proto/bootstrap.proto using the pure-Go gots generator.
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
	"time"

	"github.com/bufbuild/protocompile"
	"github.com/choysum-dev/choysum/internal/esmresolver"
	"github.com/choysum-dev/choysum/internal/module/artifact/generate/grpcwebplugin/gots"
	"github.com/choysum-dev/choysum/internal/vueplugin"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"

	// Register the default QuickJS compiler factory so jsexecutor.NewCompilerExecutor
	// can resolve the "default" factory name via its init() side-effect.
	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"

	// Register the QuickJS engine factory ("quickjs") so the executor can
	// create QuickJS runtime instances for Vue SFC compilation.
	_ "github.com/choysum-dev/choysum/internal/defaultengine"

	"github.com/evanw/esbuild/pkg/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gen.go: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Determine the cache directory (shared with the rest of the choysum toolchain).
	cacheDir := os.Getenv("CHOYSUM_HOME")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		cacheDir = filepath.Join(home, ".choysum")
	}

	// Resolve the web package directory from this source file's location.
	// This works regardless of the current working directory (go generate,
	// go run from repo root, etc.).
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("cannot determine gen.go location")
	}
	webDir := filepath.Dir(thisFile)
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(webDir)))

	// Proto file path relative to the repo root.
	protoRel := filepath.Join("internal", "bootstrap", "proto", "bootstrap.proto")
	protoDir := repoRoot

	// Output directories (always relative to the web package directory).
	srcGenDir := filepath.Join(webDir, "src", "gen", "bootstrap")
	distDir := filepath.Join(webDir, "dist")

	// ---------------------------------------------------------------------------
	// 1. Generate Connect-Web TypeScript client stubs from bootstrap.proto.
	// ---------------------------------------------------------------------------
	if err := generateProtoStubs(protoDir, protoRel, srcGenDir); err != nil {
		return fmt.Errorf("proto generation: %w", err)
	}

	// ---------------------------------------------------------------------------
	// 2. Fetch TypeScript types for IDE support.
	// ---------------------------------------------------------------------------
	client := esmresolver.NewTypeFetchHTTPClient(30 * time.Second)
	typesDir := filepath.Join(cacheDir, "pkg", "types")
	tsconfigPath := filepath.Join(webDir, "tsconfig.json")
	if results, err := esmresolver.FetchTypesForModule(client, "https://esm.sh", typesDir, webDir); err == nil {
		if err := esmresolver.UpdateTsconfigPaths(tsconfigPath, results); err != nil {
			fmt.Fprintf(os.Stderr, "gen.go: warning: failed to update tsconfig paths: %v\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "gen.go: warning: failed to fetch types: %v\n", err)
	}

	// ---------------------------------------------------------------------------
	// 3. Create and start the QuickJS compiler executor for Vue SFC compilation.
	// ---------------------------------------------------------------------------
	jsExec, err := newCompilerExecutor()
	if err != nil {
		return fmt.Errorf("create js executor: %w", err)
	}
	if err := jsExec.Start(); err != nil {
		return fmt.Errorf("start js executor: %w", err)
	}
	defer func() {
		if err := jsExec.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "gen.go: warning: failed to stop js executor: %v\n", err)
		}
	}()

	// ---------------------------------------------------------------------------
	// 4. Build the bootstrap web application with esbuild.
	// ---------------------------------------------------------------------------
	entryPoint := filepath.Join(webDir, "src", "main.ts")
	indexHTML := filepath.Join(webDir, "index.html")
	outIndexHTML := filepath.Join(distDir, "index.html")

	// Prepare the Vue SFC plugin with HTML index processing.
	vuePlugin := vueplugin.NewPlugin(
		vueplugin.WithJsExecutor(jsExec),
		vueplugin.WithIndexHtmlOptions(vueplugin.IndexHtmlOptions{
			SourceFile: indexHTML,
			OutFile:    outIndexHTML,
			// Remove the Vite <script type="module"> entrypoint tag — it is
			// replaced by the injected bundle tags.
			RemoveTagXPaths: []string{`//script[@type="module" and @src="/src/main.ts"]`},
		}),
	)

	// ESM resolver plugin — intercepts bare imports and resolves them via esm.sh.
	esmPlugin := esmresolver.New(
		esmresolver.WithCacheDir(cacheDir),
		esmresolver.WithTarget("es2020"),
		esmresolver.WithModulePath(webDir),
	).Plugin()

	// CSS external plugin — element-plus CSS imports (e.g.
	// 'element-plus/es/components/alert/style/css') resolve to CSS files
	// wrapped as JS modules by esm.sh. Mark them as external so esbuild
	// skips parsing CSS syntax as JavaScript. In production the bootstrap
	// page loads element-plus styles from a CDN <link> tag in index.html.
	cssExternalPlugin := api.Plugin{
		Name: "bootstrap-css-external",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^element-plus\/.*\/style\/css$`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{External: true}, nil
				})
		},
	}

	// Ensure the dist directory exists.
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		return fmt.Errorf("create dist dir: %w", err)
	}

	result := api.Build(api.BuildOptions{
		EntryPoints:       []string{entryPoint},
		Outfile:           filepath.Join(distDir, "index.js"),
		Bundle:            true,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		TreeShaking:       api.TreeShakingTrue,
		Format:            api.FormatIIFE,
		Platform:          api.PlatformBrowser,
		// Provide a global require stub for esm.sh CJS interop wrappers.
		Banner: map[string]string{
			"js": "if(typeof require==='undefined'){globalThis.require=function(m){return null;};}",
		},
		Tsconfig: filepath.Join(webDir, "tsconfig.json"),
		Plugins: []api.Plugin{
			cssExternalPlugin, // Must run before esmPlugin to intercept CSS imports.
			esmPlugin,
			vuePlugin,
		},
		Write:    true,
		Metafile: true,
	})

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			loc := ""
			if e.Location != nil {
				loc = fmt.Sprintf("%s:%d:%d ", e.Location.File, e.Location.Line, e.Location.Column)
			}
			fmt.Fprintf(os.Stderr, "gen.go: %s%s\n", loc, e.Text)
		}
		return fmt.Errorf("esbuild build failed with %d error(s)", len(result.Errors))
	}

	fmt.Println("bootstrap-web: built dist/")
	return nil
}

// generateProtoStubs compiles bootstrap.proto with protocompile and runs the
// pure-Go gots generator to produce Connect-Web TypeScript client stubs.
func generateProtoStubs(protoDir, protoRel, outDir string) error {
	// Use the relative path as the proto file name for protocompile.
	protoFileName := protoRel

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{
				protoDir,
			},
		}),
	}

	files, err := compiler.Compile(context.Background(), protoFileName)
	if err != nil {
		return fmt.Errorf("compile proto: %w", err)
	}

	// Build the CodeGeneratorRequest for the gots generator.
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{protoFileName},
		Parameter:      proto.String("target=ts"),
	}
	for _, fd := range files {
		req.ProtoFile = append(req.ProtoFile, protodesc.ToFileDescriptorProto(fd))
	}

	// Run the pure-Go TypeScript generator.
	resp, err := gots.NewGenerator().Generate(req)
	if err != nil {
		return fmt.Errorf("gots generate: %w", err)
	}

	// Write generated files to src/gen/bootstrap/.
	for _, f := range resp.GetFile() {
		fileName := f.GetName()
		content := f.GetContent()
		outPath := filepath.Join(outDir, fileName)
		parentDir := filepath.Dir(outPath)
		if err := os.MkdirAll(parentDir, 0o755); err != nil {
			return fmt.Errorf("create gen dir %s: %w", parentDir, err)
		}
		if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", outPath, err)
		}
		fmt.Printf("bootstrap-web: generated %s\n", outPath)
	}

	return nil
}

// bootstrapScope is a minimal scope.Scope implementation used solely to
// satisfy the jsexecutor factory contract for the compiler executor.
type bootstrapScope struct {
	ctx context.Context
	cfg *config.Config
}

func (s *bootstrapScope) Context() context.Context { return s.ctx }
func (s *bootstrapScope) Logger() *slog.Logger     { return slog.Default() }
func (s *bootstrapScope) Session() *scope.Session  { return nil }
func (s *bootstrapScope) WithContext(ctx context.Context) scope.Scope {
	clone := *s
	clone.ctx = ctx
	return &clone
}
func (s *bootstrapScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *bootstrapScope) Transactor() scope.Transactor {
	return scope.NewRunSessionTransactor(s)
}

// FactoryInput returns a scope.FactoryInput that exposes the server config
// so the executor can resolve the "quickjs" engine factory by name.
func (s *bootstrapScope) FactoryInput() scope.FactoryInput {
	return bootstrapFactoryInput{cfg: s.cfg}
}

type bootstrapFactoryInput struct {
	cfg *config.Config
}

func (i bootstrapFactoryInput) Environment() string { return "" }
func (i bootstrapFactoryInput) ServerConfig() *config.ServerConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.Server
}

// newCompilerExecutor creates an unstarted compiler executor using the
// "default" factory registered by internal/defaultjsexecutor and the
// "quickjs" engine factory registered by internal/defaultengine.
func newCompilerExecutor() (jsexecutor.JsExecutor, error) {
	srvCfg := config.NewDefaultServerConfig()
	srvCfg.JsEngineFactory = "quickjs"
	scope := &bootstrapScope{
		ctx: context.Background(),
		cfg: &config.Config{Server: srvCfg},
	}
	exec, err := jsexecutor.NewCompilerExecutor(scope)
	if err != nil {
		return nil, fmt.Errorf("create compiler executor: %w", err)
	}
	return exec, nil
}

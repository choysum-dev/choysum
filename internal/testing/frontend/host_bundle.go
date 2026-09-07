// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/internal/esmresolver"
	"github.com/choysum-dev/choysum/internal/vueplugin"
	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/choysummount"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/evanw/esbuild/pkg/api"
	xfmt "golang.org/x/exp/errors/fmt"
)

// Test seams for rare OS / path failures (overridden in unit tests).
var (
	hostFilepathAbs = filepath.Abs
	hostUserHomeDir = os.UserHomeDir
)

// VueHostBundleOptions configures an FE unit host bundle with real vue + optional SFC.
type VueHostBundleOptions struct {
	RepoRoot   string
	EntryPath  string
	Outfile    string
	Sourcemap  bool
	WorkingDir string
	// CacheDir for esmresolver (default: $CHOYSUM_HOME or ~/.choysum).
	CacheDir string
	// JsExecutor required when bundling .vue (vueplugin).
	JsExecutor jsexecutor.ScriptExecutor
	// WithVuePlugin enables vueplugin (needed for .vue entries/imports).
	WithVuePlugin bool
}

// BuildFrontendVueHostBundle bundles an entry with real vue (esmresolver) and choysummount alias.
// It does not call NewModuleBuilder. Default --fe remains Vitest until PR-unit-final-fe.
func BuildFrontendVueHostBundle(opts VueHostBundleOptions) (*BundleResult, error) {
	repoRoot := strings.TrimSpace(opts.RepoRoot)
	entry := strings.TrimSpace(opts.EntryPath)
	if repoRoot == "" {
		return nil, xfmt.Errorf("vue host bundle: empty repo root")
	}
	if entry == "" {
		return nil, xfmt.Errorf("vue host bundle: empty entry path")
	}
	mountPath, err := ChoysumMountSourcePath()
	if err != nil {
		return nil, xfmt.Errorf("vue host bundle: choysummount path: %w", err)
	}
	mountPath, err = hostFilepathAbs(mountPath)
	if err != nil {
		return nil, xfmt.Errorf("vue host bundle: abs choysummount: %w", err)
	}

	cacheDir := strings.TrimSpace(opts.CacheDir)
	if cacheDir == "" {
		cacheDir = os.Getenv("CHOYSUM_HOME")
	}
	if cacheDir == "" {
		home, homeErr := hostUserHomeDir()
		if homeErr != nil {
			return nil, xfmt.Errorf("vue host bundle: cache dir: %w", homeErr)
		}
		cacheDir = filepath.Join(home, ".choysum")
	}

	outfile := strings.TrimSpace(opts.Outfile)
	if outfile == "" {
		outfile = filepath.Join(filepath.Dir(entry), "vue-host.bundle.js")
	}
	if err := osMkdirAll(filepath.Dir(outfile), 0o755); err != nil {
		return nil, xfmt.Errorf("vue host bundle: mkdir: %w", err)
	}

	modulesDir := filepath.Join(repoRoot, "modules")
	vueSpec := "vue@" + choysummount.VuePackageVersion
	plugins := []api.Plugin{
		{
			Name: "choysum-test-utils-alias",
			Setup: func(build api.PluginBuild) {
				build.OnResolve(api.OnResolveOptions{Filter: `^@choysum/test-utils$`},
					func(args api.OnResolveArgs) (api.OnResolveResult, error) {
						return api.OnResolveResult{Path: mountPath, Namespace: "file"}, nil
					})
			},
		},
		esmresolver.New(
			esmresolver.WithCacheDir(cacheDir),
			esmresolver.WithTarget("es2020"),
			esmresolver.WithModulePath(repoRoot),
		).Plugin(),
	}
	if opts.WithVuePlugin {
		if opts.JsExecutor == nil {
			return nil, xfmt.Errorf("vue host bundle: JsExecutor required when WithVuePlugin is set")
		}
		plugins = append(plugins, vueplugin.NewPlugin(vueplugin.WithJsExecutor(opts.JsExecutor)))
	}

	absWorkingDir := strings.TrimSpace(opts.WorkingDir)
	if absWorkingDir == "" {
		absWorkingDir = repoRoot
	}
	// vueplugin resolves `@/` via TsconfigRaw relative to AbsWorkingDir; keep that
	// mapping pointed at repoRoot/modules even when WorkingDir differs (fixture dirs).
	modulesFromWork, err := filepath.Rel(absWorkingDir, modulesDir)
	if err != nil {
		return nil, xfmt.Errorf("vue host bundle: modules relpath: %w", err)
	}
	modulesGlob := filepath.ToSlash(filepath.Join(modulesFromWork, "*"))
	tsconfigRawBytes, err := json.Marshal(map[string]any{
		"compilerOptions": map[string]any{
			"baseUrl": ".",
			"paths": map[string][]string{
				"@/*": {modulesGlob},
			},
		},
	})
	if err != nil {
		return nil, xfmt.Errorf("vue host bundle: tsconfig: %w", err)
	}

	buildOpts := api.BuildOptions{
		EntryPoints:   []string{entry},
		Bundle:        true,
		Write:         true,
		Outfile:       outfile,
		Platform:      api.PlatformBrowser,
		Format:        api.FormatIIFE,
		Target:        api.ES2020,
		LogLevel:      api.LogLevelWarning,
		AbsWorkingDir: absWorkingDir,
		Alias: map[string]string{
			"@":   modulesDir,
			"vue": vueSpec,
		},
		TsconfigRaw: string(tsconfigRawBytes),
		Loader: map[string]api.Loader{
			".svg":  api.LoaderDataURL,
			".png":  api.LoaderDataURL,
			".jpg":  api.LoaderDataURL,
			".css":  api.LoaderEmpty,
			".scss": api.LoaderEmpty,
			".sass": api.LoaderEmpty,
		},
		Plugins: plugins,
		Define: map[string]string{
			"import.meta.env.MODE":                    "'test'",
			"import.meta.env.PROD":                    "false",
			"import.meta.env.DEV":                     "true",
			"import.meta.env.SSR":                     "false",
			"process.env.NODE_ENV":                    "'test'",
			"__VUE_OPTIONS_API__":                     "true",
			"__VUE_PROD_DEVTOOLS__":                   "false",
			"__VUE_PROD_HYDRATION_MISMATCH_DETAILS__": "false",
		},
	}
	if opts.Sourcemap {
		buildOpts.Sourcemap = api.SourceMapLinked
	}

	result := esbuildBuild(buildOpts)
	if len(result.Errors) > 0 {
		var b strings.Builder
		for _, e := range result.Errors {
			b.WriteString(e.Text)
			b.WriteByte('\n')
		}
		return nil, xfmt.Errorf("vue host bundle: esbuild: %s", b.String())
	}
	jsBytes, err := osReadFile(outfile)
	if err != nil {
		return nil, xfmt.Errorf("vue host bundle: read outfile: %w", err)
	}
	out := &BundleResult{
		JS:     string(jsBytes),
		JSPath: outfile,
	}
	if opts.Sourcemap {
		mapPath := outfile + ".map"
		if _, statErr := osStatBundle(mapPath); statErr == nil {
			out.MapPath = mapPath
		}
	}
	for _, w := range result.Warnings {
		out.Warnings = append(out.Warnings, w.Text)
	}
	return out, nil
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
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
	hostFilepathRel = filepath.Rel
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
	// ExtraStubAliases maps import paths (package names or absolute file paths) to stub files.
	// Default FE unit stubs (element-plus, icons, vue-router, path stubs, auth store) always apply.
	// OPage.vue is stubbed only for page/view product importers (see feUnitPathStubPath).
	ExtraStubAliases map[string]string
	// DisableDefaultFEStubs skips built-in package/path stubs (host unit tests only).
	DisableDefaultFEStubs bool
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
	stubDir := filepath.Join(repoRoot, "internal", "testing", "frontend", "testdata", "stubs")

	alias := map[string]string{
		"@":   modulesDir,
		"vue": vueSpec,
	}
	for k, v := range opts.ExtraStubAliases {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		alias[k] = v
	}

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
	}
	if !opts.DisableDefaultFEStubs {
		stubs := feUnitStubPaths{
			ElementPlus:    filepath.Join(stubDir, "element_plus.js"),
			Icons:          filepath.Join(stubDir, "element_plus_icons.js"),
			Router:         filepath.Join(stubDir, "vue_router.js"),
			PageMount:      filepath.Join(stubDir, "page_mount.js"),
			OPage:          filepath.Join(stubDir, "OPage.stub.vue"),
			ChildView:      filepath.Join(stubDir, "ChildView.stub.vue"),
			AuthStore:      filepath.Join(stubDir, "auth_store.js"),
			I18n:           filepath.Join(stubDir, "i18n_create_translate.js"),
			I18nStore:      filepath.Join(stubDir, "i18n_store.js"),
			Registry:       filepath.Join(stubDir, "store_registry.js"),
			Scope:          filepath.Join(stubDir, "store_scope_manager.js"),
			Permission:     filepath.Join(stubDir, "use_permission.js"),
			PageComposable: filepath.Join(stubDir, "page_composables.js"),
			Vicons:         filepath.Join(stubDir, "vicons_material.js"),
			VueEcharts:     filepath.Join(stubDir, "vue_echarts.js"),
			Vuedraggable:   filepath.Join(stubDir, "vuedraggable.js"),
			Echarts:        filepath.Join(stubDir, "echarts.js"),
		}
		plugins = append(plugins, api.Plugin{
			Name: "choysum-fe-unit-package-stubs",
			Setup: func(build api.PluginBuild) {
				build.OnResolve(api.OnResolveOptions{Filter: `^(element-plus|@element-plus/icons-vue|@vicons/material|vue-router|@choysum/page-mount|vue-echarts|vuedraggable|echarts(/.*)?)$`},
					func(args api.OnResolveArgs) (api.OnResolveResult, error) {
						// Filter only admits known package names; lookup always succeeds.
						path, _ := feUnitPackageStubPath(args.Path, stubs)
						return api.OnResolveResult{Path: path, Namespace: "file"}, nil
					})
			},
		})
		// OPage.vue is stubbed only for page/view product importers via feUnitPathStubPath
		// (not a blanket OPage.vue$ rewrite), so OPage.mapping and similar FE units mount the real SUT.
		plugins = append(plugins, api.Plugin{
			Name: "choysum-fe-unit-path-stubs",
			Setup: func(build api.PluginBuild) {
				build.OnResolve(api.OnResolveOptions{Filter: `.*`},
					func(args api.OnResolveArgs) (api.OnResolveResult, error) {
						p := filepath.ToSlash(args.Path)
						importer := filepath.ToSlash(args.Importer)
						joined := p
						if !filepath.IsAbs(args.Path) && args.ResolveDir != "" {
							joined = filepath.ToSlash(filepath.Clean(filepath.Join(args.ResolveDir, args.Path)))
						}
						if path, ok := feUnitPathStubPath(p, joined, importer, stubs); ok {
							return api.OnResolveResult{Path: path, Namespace: "file"}, nil
						}
						return api.OnResolveResult{}, nil
					})
			},
		})
	}
	plugins = append(plugins, esmresolver.New(
		esmresolver.WithCacheDir(cacheDir),
		esmresolver.WithTarget("es2020"),
		esmresolver.WithModulePath(repoRoot),
	).Plugin())
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
	modulesFromWork, err := hostFilepathRel(absWorkingDir, modulesDir)
	if err != nil {
		return nil, xfmt.Errorf("vue host bundle: modules relpath: %w", err)
	}
	modulesGlob := filepath.ToSlash(filepath.Join(modulesFromWork, "*"))
	tsconfigRawBytes, err := jsonMarshal(map[string]any{
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
		Alias:         alias,
		TsconfigRaw:   string(tsconfigRawBytes),
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
			"import.meta.env.MODE":                        "'test'",
			"import.meta.env.PROD":                        "false",
			"import.meta.env.DEV":                         "true",
			"import.meta.env.SSR":                         "false",
			"import.meta.env.CHOYSUM_ENABLE_REGISTRATION": "true",
			"process.env.NODE_ENV":                        "'test'",
			"__VUE_OPTIONS_API__":                         "true",
			"__VUE_PROD_DEVTOOLS__":                       "false",
			"__VUE_PROD_HYDRATION_MISMATCH_DETAILS__":     "false",
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

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
	// Default FE unit stubs (vue-router, page-mount, tiptap, path stubs, auth store) always apply.
	// Page.vue / legacy OPage.vue use a dedicated rewrite (skips Page*.test.ts / OPage*.test.ts
	// importers) plus path stubs so domain page mounts stay light.
	ExtraStubAliases map[string]string
	// DisableDefaultFEStubs skips built-in package/path stubs (host unit tests only).
	DisableDefaultFEStubs bool
}

// BuildFrontendVueHostBundle bundles an entry with real vue (esmresolver) and choysummount alias.
// It does not call NewModuleBuilder.
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
	stubDir := filepath.Join(repoRoot, "internal", "testing", "frontend", "testdata", "stubs")

	// Keep path aliases for @; vue (+ exact peers from modules/web) are pinned
	// via WithBareImportPins so esm.sh does not float to incompatible majors
	// (e.g. @tanstack/vue-table v9 renaming useVueTable).
	alias := map[string]string{
		"@": modulesDir,
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
			Router:              filepath.Join(stubDir, "vue_router.js"),
			PageMount:           filepath.Join(stubDir, "page_mount.js"),
			OPage:               filepath.Join(stubDir, "OPage.stub.vue"),
			ChildView:           filepath.Join(stubDir, "ChildView.stub.vue"),
			AuthStore:           filepath.Join(stubDir, "auth_store.js"),
			I18n:                filepath.Join(stubDir, "i18n_create_translate.js"),
			I18nStore:           filepath.Join(stubDir, "i18n_store.js"),
			Registry:            filepath.Join(stubDir, "store_registry.js"),
			Scope:               filepath.Join(stubDir, "store_scope_manager.js"),
			Permission:          filepath.Join(stubDir, "use_permission.js"),
			PageComposable:      filepath.Join(stubDir, "page_composables.js"),
			Vuedraggable:        filepath.Join(stubDir, "vuedraggable.js"),
			TipTapVue3:          filepath.Join(stubDir, "tiptap_vue3.js"),
			TipTapStarterKit:    filepath.Join(stubDir, "tiptap_starter_kit.js"),
			TipTapExtensionLink: filepath.Join(stubDir, "tiptap_extension_link.js"),
			DOMPurify:           filepath.Join(stubDir, "dompurify.js"),
			UnovisVue:           filepath.Join(stubDir, "unovis_vue.js"),
			UnovisTs:            filepath.Join(stubDir, "unovis_ts.js"),
			RekaUI:              filepath.Join(stubDir, "reka_ui.js"),
			LucideVueNext:       filepath.Join(stubDir, "lucide_vue_next.js"),
		}
		plugins = append(plugins, api.Plugin{
			Name: "choysum-fe-unit-package-stubs",
			Setup: func(build api.PluginBuild) {
				build.OnResolve(api.OnResolveOptions{Filter: `^(vue-router|@choysum/page-mount|vuedraggable|@tiptap/vue-3|@tiptap/starter-kit|@tiptap/extension-link|dompurify|@unovis/vue|@unovis/ts|reka-ui|lucide-vue-next)$`},
					func(args api.OnResolveArgs) (api.OnResolveResult, error) {
						// Filter and feUnitPackageStubPath must stay in sync; if they
						// drifted, fall through instead of resolving to an empty path.
						path, ok := feUnitPackageStubPath(args.Path, stubs)
						if !ok {
							return api.OnResolveResult{}, nil
						}
						return api.OnResolveResult{Path: path, Namespace: "file"}, nil
					})
			},
		})
		plugins = append(plugins, api.Plugin{
			Name: "choysum-fe-unit-page-stub",
			Setup: func(build api.PluginBuild) {
				// Blanket Page.vue / legacy OPage.vue rewrite for product pages (path-alias
				// resolves often leave Importer empty, so feUnitPathStubPath alone is not
				// enough). Does not match ChoyPage.vue. Skip when a web Page unit test
				// imports the real SUT.
				build.OnResolve(api.OnResolveOptions{Filter: `(?:^|/)(?:OPage|Page)\.vue$`},
					func(args api.OnResolveArgs) (api.OnResolveResult, error) {
						importer := filepath.ToSlash(args.Importer)
						// Match feUnitPathStubPath: web component tests/SFCs keep real Page.
						isWebComponentUnit := strings.Contains(importer, "/web/web/components/") &&
							(strings.Contains(importer, ".test.") || strings.Contains(importer, ".spec.") ||
								strings.HasSuffix(importer, ".vue"))
						base := filepath.Base(importer)
						if isWebComponentUnit ||
							strings.HasPrefix(base, "Page.mapping.test.") ||
							strings.HasPrefix(base, "Page.test.") ||
							strings.HasPrefix(base, "OPage.mapping.test.") ||
							strings.HasPrefix(base, "OPage.test.") {
							return api.OnResolveResult{}, nil
						}
						return api.OnResolveResult{Path: stubs.OPage, Namespace: "file"}, nil
					})
			},
		})
		plugins = append(plugins, api.Plugin{
			Name: "choysum-fe-unit-child-view-stub",
			Setup: func(build api.PluginBuild) {
				// Same Importer-empty problem as Page: stub nested Form/List/Kanban for
				// product page/view mounts. Skip when a FE unit test imports the *View SUT,
				// or when web/web/components code is the importer (real child mounts).
				build.OnResolve(api.OnResolveOptions{Filter: `(FormView|ListView|KanbanView)\.vue$`},
					func(args api.OnResolveArgs) (api.OnResolveResult, error) {
						importer := filepath.ToSlash(args.Importer)
						if strings.Contains(importer, "/web/web/components/") ||
							strings.HasSuffix(importer, ".test.ts") ||
							strings.HasSuffix(importer, ".spec.ts") {
							return api.OnResolveResult{}, nil
						}
						return api.OnResolveResult{Path: stubs.ChildView, Namespace: "file"}, nil
					})
			},
		})
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
	barePins, pinErr := vueHostBareImportPins(repoRoot)
	if pinErr != nil {
		return nil, pinErr
	}
	plugins = append(plugins, esmresolver.New(
		esmresolver.WithCacheDir(cacheDir),
		esmresolver.WithTarget("es2020"),
		esmresolver.WithModulePath(repoRoot),
		esmresolver.WithBareImportPins(barePins),
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

// vueHostBareImportPins merges the host Vue pin with exact versions from
// modules/web/package.json so FE unit bundles resolve Choy kit peers
// (TanStack Table, Reka, …) instead of floating esm.sh majors.
func vueHostBareImportPins(repoRoot string) (map[string]string, error) {
	pins := choysummount.VueBareImportPins()
	webPins, err := esmresolver.ExactPinsFromPackageJSON(filepath.Join(repoRoot, "modules", "web"))
	if err != nil {
		return nil, xfmt.Errorf("vue host bundle: exact pins from modules/web: %w", err)
	}
	if len(webPins) == 0 {
		return pins, nil
	}
	for name, ver := range webPins {
		if name == "vue" || strings.HasPrefix(name, "@vue/") {
			continue // host Vue pin wins for single-instance correctness
		}
		pins[name] = ver
	}
	return pins, nil
}

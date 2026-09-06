// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/choysum-dev/choysum/internal/typecheck/vue"
)

// Check typechecks an application's TypeScript roots using
// typescript-go-internal. It does not invoke Node or vue-tsc.
// ScopeNoVue includes web TS/TSX plus embedded vite/client and subpath ambient.
// ScopeAll also includes .vue files via Coder service-script overlays (Strategy B).
func Check(ctx context.Context, opts Options) (Result, error) {
	if err := validateOptions(opts); err != nil {
		return Result{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctxErr(ctx); err != nil {
		return Result{}, err
	}

	modulesPath, err := absPath(opts.ModulesPath)
	if err != nil {
		return Result{}, err
	}
	repoRoot, err := absPath(opts.RepoRoot)
	if err != nil {
		return Result{}, err
	}
	if err := ctxErr(ctx); err != nil {
		return Result{}, err
	}

	scope := opts.Scope
	files, err := CollectRootFiles(ctx, modulesPath, opts.App, scope)
	if err != nil {
		if !errors.Is(err, ErrNoRootFiles) || len(opts.Overlays) == 0 {
			return Result{}, err
		}
		files = nil
	}
	if err := ctxErr(ctx); err != nil {
		return Result{}, err
	}

	overlays := resolveOverlaysAgainstModules(opts.Overlays, modulesPath)
	var ambientOverlays map[string]string
	var vueScripts map[string]vue.ServiceScript
	switch scope {
	case ScopeNoVue:
		ambientOverlays = BuiltInAmbientOverlays(modulesPath)
		overlays = mergeOverlays(ambientOverlays, overlays)
	case ScopeAll:
		ambientOverlays = BuiltInVueAmbientOverlays(modulesPath, repoRoot)
		overlays = mergeOverlays(ambientOverlays, overlays)
		coder, err := resolveVueCoder(opts)
		if err != nil {
			return Result{}, err
		}
		// Only close coders we create; caller-supplied Options.Coder stays open.
		if opts.Coder == nil {
			defer closeVueCoder(coder)
		}
		caseSensitive := newTypecheckFS(overlays).UseCaseSensitiveFileNames()
		vuePaths := mergeVuePaths(
			collectVuePaths(files),
			collectVueOverlayPaths(modulesPath, opts.App, overlays, caseSensitive),
		)
		// TypeScript follows imports into other modules' .vue SFCs; codegen those
		// too so Host does not serve raw SFC text (and so ambient *.vue is unnecessary).
		webVuePaths, err := collectModulesWebVuePaths(modulesPath)
		if err != nil {
			return Result{}, err
		}
		vuePaths = mergeVuePaths(vuePaths, webVuePaths)
		vueOverlays, scripts, err := prepareVueOverlays(coder, vuePaths, modulesPath, overlays)
		if err != nil {
			return Result{}, err
		}
		vueScripts = scripts
		overlays = mergeOverlays(overlays, vueOverlays)
		files = rewriteVueRootsToProgramPaths(files)
	}
	fs := newTypecheckFS(overlays)
	files = appendOverlayRoots(files, modulesPath, opts.App, scope, overlays, fs.UseCaseSensitiveFileNames())
	for _, ambient := range sortedOverlayPaths(ambientOverlays) {
		files = appendUniqueSlash(files, ambient)
	}
	if scope == ScopeAll {
		for _, helper := range sortedOverlayPaths(vue.HelperOverlays()) {
			files = appendUniqueSlash(files, normalizePathKey(helper))
		}
	}
	if coreAmbient := filepath.ToSlash(filepath.Join(modulesPath, "core", "types", "$choysum.d.ts")); fs.FileExists(coreAmbient) {
		files = appendUniqueSlash(files, coreAmbient)
	}
	if len(files) == 0 {
		return Result{}, ErrNoRootFiles
	}
	if err := ctxErr(ctx); err != nil {
		return Result{}, err
	}

	compilerOpts, err := BuildCompilerOptions(modulesPath, repoRoot)
	if err != nil {
		return Result{}, err
	}
	if err := ctxErr(ctx); err != nil {
		return Result{}, err
	}

	host := newHost(modulesPath, fs)
	program := buildProgram(host, files, compilerOpts)
	if err := ctxErr(ctx); err != nil {
		return Result{}, err
	}
	diags, err := collectDiagnostics(ctx, program)
	if err != nil {
		return Result{}, err
	}
	res := toResult(diags)
	res.Diagnostics = remapDiagnostics(res.Diagnostics, vueScripts)
	res.Diagnostics = filterDiagnosticsToApp(res.Diagnostics, modulesPath, opts.App)
	res.Diagnostics = suppressVueTemplateParityNoise(res.Diagnostics)
	return res, nil
}

// filterDiagnosticsToApp keeps diagnostics under modules/<app>/ (and virtual
// .vue.ts siblings). Matches the historical vue-tsc include scope for an app.
func filterDiagnosticsToApp(diags []Diagnostic, modulesPath, app string) []Diagnostic {
	app = strings.TrimSpace(app)
	if app == "" || len(diags) == 0 {
		return diags
	}
	prefix := filepath.ToSlash(filepath.Join(modulesPath, app)) + "/"
	out := make([]Diagnostic, 0, len(diags))
	for _, d := range diags {
		file := filepath.ToSlash(d.File)
		if strings.HasPrefix(file, prefix) {
			out = append(out, d)
		}
	}
	return out
}

// suppressVueTemplateParityNoise drops template diagnostics that vue-tsc does
// not surface for the same sources (expose-only refs still emit ['$el'] access;
// empty emit tuples / VNodeProps∩onClick conflicts differ under typescript-go).
func suppressVueTemplateParityNoise(diags []Diagnostic) []Diagnostic {
	if len(diags) == 0 {
		return diags
	}
	out := make([]Diagnostic, 0, len(diags))
	for _, d := range diags {
		if isVueDiagnosticFile(d.File) && isVueTemplateParityNoise(d) {
			continue
		}
		out = append(out, d)
	}
	return out
}

func isVueDiagnosticFile(file string) bool {
	lower := strings.ToLower(filepath.ToSlash(file))
	return strings.HasSuffix(lower, ".vue") || strings.HasSuffix(lower, ".vue.ts")
}

func isVueTemplateParityNoise(d Diagnostic) bool {
	switch d.Code {
	case 2339:
		if strings.Contains(d.Message, "'$el'") {
			return true
		}
		// Slot `default` / collapsed setup ctx (`{}`) under language-core + typescript-go.
		if strings.Contains(d.Message, "does not exist on type '{}'") {
			return true
		}
		if strings.Contains(d.Message, "'default'") &&
			(strings.Contains(d.Message, "__VLS") ||
				strings.Contains(d.Message, "Slots") ||
				strings.Contains(d.Message, "slots")) {
			return true
		}
		// setup() context helper: `.expose` on attrs/slots/emit bag | undefined.
		if strings.Contains(d.Message, "'expose'") &&
			(strings.Contains(d.Message, "attrs") ||
				strings.Contains(d.Message, "slots") ||
				strings.Contains(d.Message, "__VLS_Slots")) {
			return true
		}
		return false
	case 18048:
		// Optional chaining gaps on generated __VLS_* temps.
		return strings.Contains(d.Message, "__VLS_")
	case 2493:
		return strings.Contains(d.Message, "Tuple type '[]'")
	case 2322:
		// VNodeProps ∩ component props often collapses under typescript-go for
		// template listener object literals (`{ onX: ... }`). Do not use a bare
		// "on" substring — it matches inside "NonNullable" itself.
		return strings.Contains(d.Message, "NonNullable") &&
			(strings.Contains(d.Message, "{ on") || strings.Contains(d.Message, "VNodeProps"))
	case 7006, 7031:
		// Generated script-setup / render params (props, attrs, v-model, $event)
		// often lack contextual types under typescript-go; vue-tsc does not
		// surface them. Only applies to .vue via isVueDiagnosticFile.
		return true
	case 2558:
		// defineComponent<Props>(...) when the resolved overload has no type params.
		return strings.Contains(d.Message, "type arguments")
	case 7053:
		return strings.Contains(d.Message, `type '""'`) || strings.Contains(d.Message, "__VLS_Slots")
	case 2552:
		return strings.Contains(d.Message, "__VLS_asFunctionalElement")
	case 2589:
		// Template-generated deep component instantiation only.
		return strings.Contains(d.Message, "excessively deep") && strings.Contains(d.Message, "__VLS")
	case 2349:
		// Template slot/call-site fallout — require language-core helpers.
		return strings.Contains(d.Message, "not callable") && strings.Contains(d.Message, "__VLS")
	default:
		return false
	}
}

// Test hook for mid-phase cancellation coverage.
var ctxErr = func(ctx context.Context) error { return ctx.Err() }

// resolveOverlaysAgainstModules makes overlay keys absolute under modulesPath
// so relative overlay paths do not resolve against the process CWD.
func resolveOverlaysAgainstModules(overlays map[string]string, modulesPath string) map[string]string {
	if len(overlays) == 0 {
		return nil
	}
	out := make(map[string]string, len(overlays))
	for k, v := range overlays {
		path := strings.TrimSpace(k)
		if path == "" {
			continue
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(modulesPath, path)
		}
		out[normalizePathKey(path)] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// appendOverlayRoots adds overlay-only paths that match the scope collection
// rules (app-root / service / web), including virtual files.
func appendOverlayRoots(files []string, modulesPath, app string, scope Scope, overlays map[string]string, caseSensitive bool) []string {
	switch scope {
	case ScopeService, ScopeNoVue, ScopeAll:
	default:
		return files
	}
	if len(overlays) == 0 {
		return files
	}
	app = strings.TrimSpace(app)
	appRoot := filepath.ToSlash(filepath.Join(modulesPath, app))
	overlayKeys := make([]string, 0, len(overlays))
	for k := range overlays {
		overlayKeys = append(overlayKeys, k)
	}
	slices.Sort(overlayKeys)
	allowTSX := scope == ScopeNoVue || scope == ScopeAll
	allowVue := scope == ScopeAll
	for _, path := range overlayKeys {
		norm := normalizePathKey(path)
		if norm == "" || shouldSkipTSFileName(filepath.Base(norm)) {
			continue
		}
		lower := strings.ToLower(norm)
		isVue := strings.HasSuffix(lower, ".vue")
		isTSX := strings.HasSuffix(lower, ".tsx")
		isTS := strings.HasSuffix(lower, ".ts")
		if isVue {
			if !allowVue {
				continue
			}
		} else if !(isTS || (isTSX && allowTSX)) {
			continue
		}
		rel, ok := cutPathPrefix(norm, appRoot+"/", caseSensitive)
		if !ok {
			continue
		}
		if !strings.Contains(rel, "/") {
			// App-root roots are .ts / .d.ts only (not .tsx / .vue).
			if isTSX || isVue {
				continue
			}
			files = appendUniqueSlash(files, norm)
			continue
		}
		relLower := rel
		if !caseSensitive {
			relLower = strings.ToLower(rel)
		}
		switch {
		case strings.HasPrefix(relLower, "service/"):
			if isTSX || isVue {
				continue
			}
		case allowTSX && strings.HasPrefix(relLower, "web/"):
			// web allows .ts/.tsx and, for ScopeAll, .vue
		default:
			continue
		}
		parts := strings.Split(rel, "/")
		skip := false
		for _, part := range parts[:len(parts)-1] {
			if shouldSkipScanDir(part) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		if isVue {
			files = appendUniqueSlash(files, toVueProgramPath(norm))
			continue
		}
		files = appendUniqueSlash(files, norm)
	}
	return files
}

func cutPathPrefix(path, prefix string, caseSensitive bool) (string, bool) {
	if caseSensitive {
		return strings.CutPrefix(path, prefix)
	}
	if len(path) < len(prefix) {
		return "", false
	}
	if !strings.EqualFold(path[:len(prefix)], prefix) {
		return "", false
	}
	return path[len(prefix):], true
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func appendUniqueSlash(files []string, path string) []string {
	path = filepath.ToSlash(path)
	for _, f := range files {
		if f == path {
			return files
		}
	}
	return append(files, path)
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/choysum-dev/choysum/internal/typecheck/vue"
)

// walkModulesWebVueDir is filepath.WalkDir for collectModulesWebVuePaths; tests may override.
var walkModulesWebVueDir = filepath.WalkDir

// prepareVueOverlays builds Strategy-B overlays: each .vue and .vue.ts path maps
// to the same service-script text, plus language-core helper declaration files.
// Source text prefers existingOverlays, then disk; a missing source is an error.
func prepareVueOverlays(coder vue.Coder, vuePaths []string, modulesPath string, existingOverlays map[string]string) (map[string]string, map[string]vue.ServiceScript, error) {
	if coder == nil {
		return nil, nil, fmt.Errorf("typecheck: Vue Coder is required for ScopeAll")
	}
	out := make(map[string]string)
	scripts := make(map[string]vue.ServiceScript, len(vuePaths)*2)
	for path, content := range vue.HelperOverlays() {
		out[normalizePathKey(path)] = content
	}
	opts := vue.CodegenOptions{CurrentDirectory: modulesPath}
	for _, vuePath := range vuePaths {
		norm := normalizePathKey(vuePath)
		src, err := readVueSource(norm, vuePath, existingOverlays)
		if err != nil {
			return nil, nil, err
		}
		script, err := coder.CreateServiceScript(norm, src, opts)
		if err != nil {
			return nil, nil, err
		}
		if script.SourceContent == "" {
			script.SourceContent = src
		}
		prog := normalizePathKey(toVueProgramPath(norm))
		out[norm] = script.Content
		out[prog] = script.Content
		scripts[norm] = script
		scripts[prog] = script
	}
	return out, scripts, nil
}

func readVueSource(norm, vuePath string, existingOverlays map[string]string) (string, error) {
	if existingOverlays != nil {
		if content, ok := existingOverlays[norm]; ok {
			return content, nil
		}
		if content, ok := existingOverlays[filepath.ToSlash(vuePath)]; ok {
			return content, nil
		}
	}
	src, err := os.ReadFile(vuePath)
	if err != nil {
		return "", fmt.Errorf("typecheck: read vue source %s: %w", vuePath, err)
	}
	return string(src), nil
}

func collectVuePaths(files []string) []string {
	var out []string
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f), ".vue") {
			out = append(out, f)
		}
	}
	return out
}

// collectModulesWebVuePaths returns every modules/<app>/web/**/*.vue path
// (excluding test trees) so cross-app SFC imports receive service-script overlays.
// Walk I/O errors are returned so Check does not continue with a partial overlay set.
func collectModulesWebVuePaths(modulesPath string) ([]string, error) {
	modulesPath = filepath.Clean(modulesPath)
	entries, err := os.ReadDir(modulesPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("typecheck: read modules root %s: %w", modulesPath, err)
	}
	var out []string
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		name := ent.Name()
		if strings.HasPrefix(name, ".") || name == "tmp" {
			continue
		}
		webDir := filepath.Join(modulesPath, name, "web")
		st, err := os.Stat(webDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("typecheck: stat module web dir %s: %w", webDir, err)
		}
		if !st.IsDir() {
			continue
		}
		if err := walkModulesWebVueDir(webDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				base := d.Name()
				if strings.HasPrefix(base, ".") || base == "node_modules" || base == "dist" ||
					base == "tmp" || base == "tests" || base == "__tests__" {
					return fs.SkipDir
				}
				return nil
			}
			if shouldSkipTSFileName(d.Name()) {
				return nil
			}
			if strings.HasSuffix(strings.ToLower(d.Name()), ".vue") {
				out = append(out, normalizePathKey(path))
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("typecheck: walk modules web vue under %s: %w", webDir, err)
		}
	}
	return out, nil
}

// collectVueOverlayPaths returns ScopeAll-eligible .vue paths that exist only
// (or also) in overlays, using the same app/web/test filters as appendOverlayRoots.
func collectVueOverlayPaths(modulesPath, app string, overlays map[string]string, caseSensitive bool) []string {
	if len(overlays) == 0 {
		return nil
	}
	app = strings.TrimSpace(app)
	appRoot := filepath.ToSlash(filepath.Join(modulesPath, app))
	keys := make([]string, 0, len(overlays))
	for k := range overlays {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var out []string
	for _, path := range keys {
		norm := normalizePathKey(path)
		if norm == "" || !strings.HasSuffix(strings.ToLower(norm), ".vue") {
			continue
		}
		if shouldSkipTSFileName(filepath.Base(norm)) {
			continue
		}
		rel, ok := cutPathPrefix(norm, appRoot+"/", caseSensitive)
		if !ok || !strings.Contains(rel, "/") {
			continue
		}
		relLower := rel
		if !caseSensitive {
			relLower = strings.ToLower(rel)
		}
		if !strings.HasPrefix(relLower, "web/") {
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
		out = append(out, norm)
	}
	return out
}

func mergeVuePaths(diskPaths, overlayPaths []string) []string {
	if len(overlayPaths) == 0 {
		return diskPaths
	}
	seen := make(map[string]struct{}, len(diskPaths)+len(overlayPaths))
	out := make([]string, 0, len(diskPaths)+len(overlayPaths))
	for _, p := range diskPaths {
		k := normalizePathKey(p)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, p)
	}
	for _, p := range overlayPaths {
		k := normalizePathKey(p)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, p)
	}
	return out
}

func resolveVueCoder(opts Options) (vue.Coder, error) {
	if opts.Coder != nil {
		return opts.Coder, nil
	}
	dir := strings.TrimSpace(opts.VueGoldenDir)
	if dir != "" {
		abs, err := absPath(dir)
		if err != nil {
			return nil, err
		}
		return vue.NewGoldenCoder(abs), nil
	}
	return vue.NewCachedCoder(vue.NewQuickJSCoder()), nil
}

func closeVueCoder(coder vue.Coder) {
	if cl, ok := coder.(interface{ Close() error }); ok {
		_ = cl.Close()
	}
}

func remapDiagnostics(diags []Diagnostic, scripts map[string]vue.ServiceScript) []Diagnostic {
	if len(scripts) == 0 {
		return diags
	}
	out := make([]Diagnostic, 0, len(diags))
	for _, d := range diags {
		key := normalizePathKey(d.File)
		script, ok := scripts[key]
		if !ok {
			script, ok = scripts[filepath.ToSlash(d.File)]
		}
		vueFile := d.File
		if v, isVueProg := fromVueProgramPath(key); isVueProg {
			vueFile = v
			if !ok {
				script, ok = scripts[normalizePathKey(v)]
			}
		}
		remapped := false
		if ok {
			if srcStart, srcLen, mapped := vue.RemapRange(script.Mappings, d.Start, d.Length); mapped {
				remapped = true
				d.FromVueTemplate = true
				d.Start = srcStart
				d.Length = srcLen
				if line, col, lok := lineColumnFromBytes([]byte(script.SourceContent), srcStart); lok {
					d.Line = line
					d.Column = col
				} else if line, col, lok := lineColumnFromFile(vueFile, srcStart); lok {
					d.Line = line
					d.Column = col
				} else {
					d.Line = 0
					d.Column = 0
				}
			}
		}
		if v, isVueProg := fromVueProgramPath(normalizePathKey(d.File)); isVueProg {
			d.File = v
			if !remapped {
				// Keep the .vue path for attribution, but drop generated coordinates.
				d.Start, d.Length, d.Line, d.Column = 0, 0, 0, 0
			}
		} else if ok {
			d.File = normalizePathKey(vueFile)
		}
		out = append(out, d)
	}
	return out
}

func lineColumnFromFile(path string, pos int) (line, col int, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil || pos < 0 {
		return 0, 0, false
	}
	return lineColumnFromBytes(data, pos)
}

func lineColumnFromBytes(data []byte, pos int) (line, col int, ok bool) {
	if pos < 0 {
		return 0, 0, false
	}
	if len(data) == 0 {
		return 0, 0, false
	}
	if pos > len(data) {
		pos = len(data)
	}
	line, col = 1, 1
	for i := 0; i < pos; i++ {
		if data[i] == '\n' {
			line++
			col = 1
			continue
		}
		col++
	}
	return line, col, true
}

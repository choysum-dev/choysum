// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/internal/typecheck/vue"
)

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

func resolveVueCoder(opts Options) (vue.Coder, error) {
	if opts.Coder != nil {
		return opts.Coder, nil
	}
	dir := strings.TrimSpace(opts.VueGoldenDir)
	if dir == "" {
		return nil, fmt.Errorf("typecheck: ScopeAll requires Options.Coder or Options.VueGoldenDir")
	}
	abs, err := absPath(dir)
	if err != nil {
		return nil, err
	}
	return vue.NewGoldenCoder(abs), nil
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
		if ok {
			if srcStart, srcLen, mapped := vue.RemapRange(script.Mappings, d.Start, d.Length); mapped {
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

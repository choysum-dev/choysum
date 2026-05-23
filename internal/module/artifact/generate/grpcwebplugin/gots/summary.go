// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gots

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"google.golang.org/protobuf/types/pluginpb"
)

var (
	importRegexp            = regexp.MustCompile(`(?m)^\s*import(?:\s+type)?\s+[^;]*\s+from\s+['"]([^'"]+)['"];`)
	exportRegexp            = regexp.MustCompile(`(?m)^\s*export\s+(?:const|type|enum|class|interface|function)\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	constExportRegexp       = regexp.MustCompile(`(?m)^\s*export\s+const\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`)
	serviceDescExportRegexp = regexp.MustCompile(`(?m)^\s*export\s+const\s+([A-Za-z_$][A-Za-z0-9_$]*)(?:\s*:[^=\n]+)?\s*=\s*(?:/\*[^\n]*\*/\s*)?serviceDesc\s*\(`)
	messageTypeBlockRegexp  = regexp.MustCompile(`(?ms)^\s*export\s+type\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*Message<'[^']+'>\s*&\s*\{\s*(.*?)\s*\};`)
	messageFieldLineRegexp  = regexp.MustCompile(`(?m)^\s*([A-Za-z_$][A-Za-z0-9_$]*\??)\s*:\s*([^;]+);\s*$`)
)

// FileSummary is the structured semantic snapshot used by golden/parity tests.
type FileSummary struct {
	Name           string              `json:"name"`
	Imports        []string            `json:"imports"`
	Exports        []string            `json:"exports"`
	SchemaExports  []string            `json:"schemaExports"`
	ServiceExports []string            `json:"serviceExports,omitempty"`
	MessageShapes  map[string][]string `json:"messageShapes,omitempty"`
}

// DiffOptions filters known and acceptable parity differences.
type DiffOptions struct {
	IgnoreFiles          []string
	IgnoreImports        []string
	IgnoreExports        []string
	IgnoreSchemaExports  []string
	IgnoreServiceExports []string
	IgnoreMessageShapes  []string

	// Key: generated file name (e.g. "auth_pb.ts").
	IgnoreImportsByFile        map[string][]string
	IgnoreExportsByFile        map[string][]string
	IgnoreSchemaExportsByFile  map[string][]string
	IgnoreServiceExportsByFile map[string][]string
	IgnoreMessageShapesByFile  map[string][]string
}

func SummarizeGeneratedFile(name, content string) FileSummary {
	s := FileSummary{Name: name}
	constExports := make([]string, 0)

	for _, m := range importRegexp.FindAllStringSubmatch(content, -1) {
		s.Imports = append(s.Imports, m[1])
	}

	for _, m := range exportRegexp.FindAllStringSubmatch(content, -1) {
		ex := m[1]
		s.Exports = append(s.Exports, ex)
		if strings.HasSuffix(ex, "Schema") {
			s.SchemaExports = append(s.SchemaExports, ex)
		}
	}

	for _, m := range constExportRegexp.FindAllStringSubmatch(content, -1) {
		constExports = append(constExports, m[1])
	}

	for _, m := range serviceDescExportRegexp.FindAllStringSubmatch(content, -1) {
		s.ServiceExports = append(s.ServiceExports, m[1])
	}
	if len(s.ServiceExports) == 0 {
		for _, ex := range constExports {
			if strings.HasSuffix(ex, "Schema") {
				continue
			}
			if strings.HasPrefix(ex, "file_") {
				continue
			}
			s.ServiceExports = append(s.ServiceExports, ex)
		}
	}

	s.Imports = uniqueSorted(s.Imports)
	s.Exports = uniqueSorted(s.Exports)
	s.SchemaExports = uniqueSorted(s.SchemaExports)
	s.ServiceExports = uniqueSorted(s.ServiceExports)
	s.MessageShapes = summarizeMessageShapes(content)

	return s
}

func SummarizeResponse(resp *pluginpb.CodeGeneratorResponse) []FileSummary {
	if resp == nil {
		return nil
	}
	out := make([]FileSummary, 0, len(resp.GetFile()))
	for _, f := range resp.GetFile() {
		out = append(out, SummarizeGeneratedFile(f.GetName(), f.GetContent()))
	}
	slices.SortFunc(out, func(a, b FileSummary) int {
		return strings.Compare(a.Name, b.Name)
	})
	return out
}

func DiffFileSummaries(want, got []FileSummary) []string {
	return DiffFileSummariesWithOptions(want, got, DiffOptions{})
}

func DiffFileSummariesWithOptions(want, got []FileSummary, opts DiffOptions) []string {
	diffs := make([]string, 0)
	ignoreFiles := toSet(opts.IgnoreFiles)

	wm := toSummaryMap(want)
	gm := toSummaryMap(got)

	for name, w := range wm {
		if _, ok := ignoreFiles[name]; ok {
			continue
		}
		g, ok := gm[name]
		if !ok {
			diffs = append(diffs, fmt.Sprintf("missing file: %s", name))
			continue
		}
		wantImports := uniqueSorted(filteredWithFile(name, w.Imports, opts.IgnoreImports, opts.IgnoreImportsByFile))
		gotImports := uniqueSorted(filteredWithFile(name, g.Imports, opts.IgnoreImports, opts.IgnoreImportsByFile))
		if !slices.Equal(wantImports, gotImports) {
			diffs = append(diffs, fmt.Sprintf("import mismatch in %s: want=%v got=%v", name, wantImports, gotImports))
		}
		wantExports := uniqueSorted(filteredWithFile(name, w.Exports, opts.IgnoreExports, opts.IgnoreExportsByFile))
		gotExports := uniqueSorted(filteredWithFile(name, g.Exports, opts.IgnoreExports, opts.IgnoreExportsByFile))
		if !slices.Equal(wantExports, gotExports) {
			diffs = append(diffs, fmt.Sprintf("export mismatch in %s: want=%v got=%v", name, wantExports, gotExports))
		}
		wantSchemaExports := uniqueSorted(filteredWithFile(name, w.SchemaExports, opts.IgnoreSchemaExports, opts.IgnoreSchemaExportsByFile))
		gotSchemaExports := uniqueSorted(filteredWithFile(name, g.SchemaExports, opts.IgnoreSchemaExports, opts.IgnoreSchemaExportsByFile))
		if !slices.Equal(wantSchemaExports, gotSchemaExports) {
			diffs = append(diffs, fmt.Sprintf("schema export mismatch in %s: want=%v got=%v", name, wantSchemaExports, gotSchemaExports))
		}
		wantServiceExports := uniqueSorted(filteredWithFile(name, w.ServiceExports, opts.IgnoreServiceExports, opts.IgnoreServiceExportsByFile))
		gotServiceExports := uniqueSorted(filteredWithFile(name, g.ServiceExports, opts.IgnoreServiceExports, opts.IgnoreServiceExportsByFile))
		if !slices.Equal(wantServiceExports, gotServiceExports) {
			diffs = append(diffs, fmt.Sprintf("service export mismatch in %s: want=%v got=%v", name, wantServiceExports, gotServiceExports))
		}

		if len(w.MessageShapes) > 0 && len(g.MessageShapes) > 0 {
			for msgName, wantShape := range w.MessageShapes {
				if shouldIgnoreWithFile(name, msgName, opts.IgnoreMessageShapes, opts.IgnoreMessageShapesByFile) {
					continue
				}
				gotShape, ok := g.MessageShapes[msgName]
				if !ok {
					diffs = append(diffs, fmt.Sprintf("message shape missing in %s: %s", name, msgName))
					continue
				}
				if !slices.Equal(wantShape, gotShape) {
					diffs = append(diffs, fmt.Sprintf("message shape mismatch in %s/%s: want=%v got=%v", name, msgName, wantShape, gotShape))
				}
			}
			for msgName := range g.MessageShapes {
				if shouldIgnoreWithFile(name, msgName, opts.IgnoreMessageShapes, opts.IgnoreMessageShapesByFile) {
					continue
				}
				if _, ok := w.MessageShapes[msgName]; !ok {
					diffs = append(diffs, fmt.Sprintf("unexpected message shape in %s: %s", name, msgName))
				}
			}
		}
	}

	for name := range gm {
		if _, ok := ignoreFiles[name]; ok {
			continue
		}
		if _, ok := wm[name]; !ok {
			diffs = append(diffs, fmt.Sprintf("unexpected file: %s", name))
		}
	}

	slices.Sort(diffs)
	return diffs
}

func toSummaryMap(items []FileSummary) map[string]FileSummary {
	out := make(map[string]FileSummary, len(items))
	for _, it := range items {
		out[it.Name] = it
	}
	return out
}

func uniqueSorted(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	slices.Sort(items)
	out := items[:0]
	for i, v := range items {
		if i == 0 || v != items[i-1] {
			out = append(out, v)
		}
	}
	return out
}

func filtered(items, ignore []string) []string {
	if len(items) == 0 {
		return nil
	}
	if len(ignore) == 0 {
		return append([]string(nil), items...)
	}
	ignoreSet := toSet(ignore)
	out := make([]string, 0, len(items))
	for _, it := range items {
		if _, ok := ignoreSet[it]; ok {
			continue
		}
		out = append(out, it)
	}
	return out
}

func filteredWithFile(fileName string, items, globalIgnore []string, byFile map[string][]string) []string {
	allIgnore := append([]string{}, globalIgnore...)
	if byFile != nil {
		allIgnore = append(allIgnore, byFile[fileName]...)
	}
	return filtered(items, allIgnore)
}

func shouldIgnoreWithFile(fileName, item string, globalIgnore []string, byFile map[string][]string) bool {
	if _, ok := toSet(globalIgnore)[item]; ok {
		return true
	}
	if byFile == nil {
		return false
	}
	_, ok := toSet(byFile[fileName])[item]
	return ok
}

func summarizeMessageShapes(content string) map[string][]string {
	shapes := map[string][]string{}
	for _, m := range messageTypeBlockRegexp.FindAllStringSubmatch(content, -1) {
		msgName := m[1]
		fieldsBlock := m[2]
		fields := make([]string, 0)
		for _, fm := range messageFieldLineRegexp.FindAllStringSubmatch(fieldsBlock, -1) {
			fieldName := strings.TrimSpace(fm[1])
			fieldType := normalizeTypeString(fm[2])
			fields = append(fields, fieldName+": "+fieldType)
		}
		if len(fields) == 0 {
			continue
		}
		shapes[msgName] = uniqueSorted(fields)
	}
	if len(shapes) == 0 {
		return nil
	}
	return shapes
}

func normalizeTypeString(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Join(strings.Fields(v), " ")
	return v
}

func toSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, it := range items {
		out[it] = struct{}{}
	}
	return out
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package coverage

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	tsast "github.com/buke/typescript-go-internal/v7/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/v7/pkg/core"
	tsparser "github.com/buke/typescript-go-internal/v7/pkg/parser"
	"github.com/buke/typescript-go-internal/v7/pkg/scanner"
	tspath "github.com/buke/typescript-go-internal/v7/pkg/tspath"
	xfmt "golang.org/x/exp/errors/fmt"
)

// Istanbul coverage schema id used by istanbul-lib-instrument (kept for consumer compatibility).
const istanbulCoverageSchema = "1a1c01bbd47fc00a2c39e90264f33305004495a9"

var sourceMappingURLRe = regexp.MustCompile(`(?m)\n?//[#@]\s*sourceMappingURL=([^\s]+)\s*$`)

// InstrumentJSFile instruments a single esbuild bundle JS file in place with
// Istanbul-shaped statement and function counters on globalThis.__coverage__.
//
// It inherits an input sourcemap (stored as inputSourceMap on the coverage
// object and written back to path+".map" when present), and appends ";void 0;".
// Statement maps / fnMap live in path+".coverage-meta.json" and are merged when
// coverage JSON is written; the JS preamble only allocates hit arrays.
func InstrumentJSFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return xfmt.Errorf("empty path")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return xfmt.Errorf("abs path: %w", err)
	}
	original, err := os.ReadFile(absPath)
	if err != nil {
		return xfmt.Errorf("read %s: %w", absPath, err)
	}

	code := string(original)
	inputMap := detectSourceMap(code, absPath)
	code = stripSourceMappingURL(code)

	instrumented, meta := instrumentJSSource(absPath, code, inputMap)

	// Write JS (and sourcemap) before the meta sidecar so a failed write cannot
	// leave a stale *.coverage-meta.json paired with an older bundle.
	finalCode := instrumented + "\n;void 0;\n"
	outMapPath := absPath + ".map"
	if inputMap != nil {
		mapBytes, err := json.Marshal(inputMap)
		if err != nil {
			return xfmt.Errorf("marshal sourcemap: %w", err)
		}
		if err := os.WriteFile(outMapPath, mapBytes, 0o644); err != nil {
			return xfmt.Errorf("write sourcemap: %w", err)
		}
		finalCode += "\n//# sourceMappingURL=" + filepath.Base(outMapPath) + "\n"
	}

	if err := os.WriteFile(absPath, []byte(finalCode), 0o644); err != nil {
		return xfmt.Errorf("write instrumented %s: %w", absPath, err)
	}

	metaPath := absPath + ".coverage-meta.json"
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return xfmt.Errorf("marshal coverage meta: %w", err)
	}
	if err := os.WriteFile(metaPath, metaBytes, 0o644); err != nil {
		return xfmt.Errorf("write coverage meta: %w", err)
	}
	return nil
}

func instrumentJSSource(absPath, code string, inputMap *rawSourceMap) (string, *coverageFileData) {
	normalized := tspath.NormalizePath(absPath)
	sf := tsparser.ParseSourceFile(tsast.SourceFileParseOptions{
		FileName: normalized,
		Path:     tspath.ToPath(normalized, "", true),
	}, code, tscore.ScriptKindJS)

	lineMap := sf.ECMALineMap()
	var stmts []instrumentPoint
	var fns []fnPoint

	// Iterative walk: esbuild bundles are large enough that a recursive
	// ForEachChild walk can overflow the goroutine stack.
	stack := []*tsast.Node{sf.AsNode()}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n == nil {
			continue
		}
		if tsast.IsFunctionLike(n) {
			start := scanner.SkipTrivia(code, n.Pos())
			end := n.End()
			if start >= 0 && end >= start && end <= len(code) {
				name := functionCoverageName(n)
				line, col := tscore.PositionToLineAndByteOffset(start, lineMap)
				endLine, endCol := tscore.PositionToLineAndByteOffset(end, lineMap)
				fns = append(fns, fnPoint{
					Name:     name,
					EntryPos: functionBodyEntryPos(code, n),
					Decl: coverageRange{
						Start: coveragePos{Line: line + 1, Column: col},
						End:   coveragePos{Line: line + 1, Column: col + 1},
					},
					Loc: coverageRange{
						Start: coveragePos{Line: line + 1, Column: col},
						End:   coveragePos{Line: endLine + 1, Column: endCol},
					},
					Line: line + 1,
				})
			}
		}
		if shouldInstrumentStatement(n) {
			start := scanner.SkipTrivia(code, n.Pos())
			end := n.End()
			if start >= 0 && end >= start && end <= len(code) {
				line, col := tscore.PositionToLineAndByteOffset(start, lineMap)
				endLine, endCol := tscore.PositionToLineAndByteOffset(end, lineMap)
				stmts = append(stmts, instrumentPoint{
					InsertPos: start,
					EndPos:    end,
					BlockWrap: statementNeedsBlockWrap(n),
					Range: coverageRange{
						Start: coveragePos{Line: line + 1, Column: col},
						End:   coveragePos{Line: endLine + 1, Column: endCol},
					},
				})
			}
		}
		n.ForEachChild(func(child *tsast.Node) bool {
			if child != nil {
				stack = append(stack, child)
			}
			return false
		})
	}
	// Stable order by insert position; assign ids in source order.
	sort.SliceStable(stmts, func(i, j int) bool {
		return stmts[i].InsertPos < stmts[j].InsertPos
	})

	statementMap := map[string]coverageRange{}
	sHits := map[string]int{}
	for i, st := range stmts {
		id := strconv.Itoa(i)
		statementMap[id] = st.Range
		sHits[id] = 0
		stmts[i].ID = i
	}

	fnMap := map[string]coverageFn{}
	fHits := map[string]int{}
	for i, fn := range fns {
		id := strconv.Itoa(i)
		fnMap[id] = coverageFn{Name: fn.Name, Decl: fn.Decl, Loc: fn.Loc, Line: fn.Line}
		fHits[id] = 0
		fns[i].ID = i
	}

	hash := sha1.Sum([]byte(code))
	meta := &coverageFileData{
		Path:           absPath,
		StatementMap:   statementMap,
		FnMap:          fnMap,
		BranchMap:      map[string]coverageBranch{},
		S:              hitMap(sHits),
		F:              hitMap(fHits),
		B:              map[string][]int{},
		InputSourceMap: inputMap,
		Hash:           hex.EncodeToString(hash[:]),
		CoverageSchema: istanbulCoverageSchema,
	}

	// Runtime preamble only carries counters; maps live in *.coverage-meta.json
	// so QuickJS does not parse multi-MB statementMap JSON on init.
	covName := coverageFunctionName(absPath, meta.Hash)
	preamble := buildCoveragePreamble(covName, absPath, meta.Hash, len(stmts), len(fns))

	type textEdit struct {
		pos     int
		text    string
		closing bool
	}
	edits := make([]textEdit, 0, len(stmts)*2+len(fns))
	for _, st := range stmts {
		inc := covName + "().s[" + strconv.Itoa(st.ID) + "]++;"
		if st.BlockWrap {
			edits = append(edits,
				textEdit{pos: st.EndPos, text: "}", closing: true},
				textEdit{pos: st.InsertPos, text: "{" + inc},
			)
			continue
		}
		edits = append(edits, textEdit{pos: st.InsertPos, text: inc})
	}
	for _, fn := range fns {
		if fn.EntryPos < 0 || fn.EntryPos > len(code) {
			continue
		}
		edits = append(edits, textEdit{
			pos:  fn.EntryPos,
			text: covName + "().f[" + strconv.Itoa(fn.ID) + "]++;",
		})
	}
	sort.SliceStable(edits, func(i, j int) bool {
		if edits[i].pos != edits[j].pos {
			return edits[i].pos < edits[j].pos
		}
		// At the same offset, emit closing braces before opening inserts so
		// trailing statements after a braced if-body are not nested inside it.
		if edits[i].closing != edits[j].closing {
			return edits[i].closing
		}
		return edits[i].text < edits[j].text
	})
	var b strings.Builder
	b.Grow(len(code) + len(edits)*24)
	prev := 0
	for _, ed := range edits {
		if ed.pos < prev || ed.pos > len(code) {
			continue
		}
		b.WriteString(code[prev:ed.pos])
		b.WriteString(ed.text)
		prev = ed.pos
	}
	b.WriteString(code[prev:])
	return preamble + b.String(), meta
}

type instrumentPoint struct {
	ID        int
	InsertPos int
	EndPos    int
	BlockWrap bool
	Range     coverageRange
}

type fnPoint struct {
	ID       int
	EntryPos int // byte offset after `{` of a block body; -1 if no block body
	Name     string
	Decl     coverageRange
	Loc      coverageRange
	Line     int
}

// functionBodyEntryPos returns the insert position for a function hit counter
// (immediately after the opening `{` of a block body). Expression-bodied
// arrows and body-less signatures return -1.
func functionBodyEntryPos(code string, n *tsast.Node) int {
	if n == nil {
		return -1
	}
	body := n.Body()
	if body == nil || body.Kind != tsast.KindBlock {
		return -1
	}
	pos := scanner.SkipTrivia(code, body.Pos())
	if pos < 0 || pos >= len(code) || code[pos] != '{' {
		return -1
	}
	return pos + 1
}

func shouldInstrumentStatement(n *tsast.Node) bool {
	if n == nil || !tsast.IsStatement(n) {
		return false
	}
	switch n.Kind {
	case tsast.KindBlock, tsast.KindEmptyStatement, tsast.KindDebuggerStatement:
		return false
	case tsast.KindModuleBlock:
		return false
	default:
		return true
	}
}

func statementNeedsBlockWrap(n *tsast.Node) bool {
	if n == nil {
		return false
	}
	switch n.Kind {
	case tsast.KindVariableStatement,
		tsast.KindFunctionDeclaration,
		tsast.KindClassDeclaration,
		tsast.KindImportDeclaration,
		tsast.KindImportEqualsDeclaration,
		tsast.KindExportAssignment,
		tsast.KindExportDeclaration,
		tsast.KindMissingDeclaration:
		return false
	default:
		return true
	}
}

func functionCoverageName(n *tsast.Node) string {
	if n == nil {
		return "(anonymous)"
	}
	name := n.Name()
	if name == nil {
		return "(anonymous)"
	}
	// Identifier.Text panics on some name kinds (e.g. ComputedPropertyName).
	if name.Kind != tsast.KindIdentifier && name.Kind != tsast.KindPrivateIdentifier && name.Kind != tsast.KindStringLiteral {
		return "(anonymous)"
	}
	text := strings.TrimSpace(name.Text())
	if text == "" {
		return "(anonymous)"
	}
	return text
}

func coverageFunctionName(path, hash string) string {
	sum := sha1.Sum([]byte(path + "\n" + hash))
	return "cov_" + hex.EncodeToString(sum[:8])
}

func buildCoveragePreamble(covName, path, hash string, statementCount, functionCount int) string {
	var b strings.Builder
	b.WriteString("function ")
	b.WriteString(covName)
	b.WriteString("(){\n")
	b.WriteString("  var global = globalThis;\n")
	b.WriteString("  var gcv = \"__coverage__\";\n")
	b.WriteString("  var path = ")
	b.WriteString(strconv.Quote(path))
	b.WriteString(";\n")
	b.WriteString("  var hash = ")
	b.WriteString(strconv.Quote(hash))
	b.WriteString(";\n")
	b.WriteString("  var coverage = global[gcv] || (global[gcv] = {});\n")
	b.WriteString("  if (!coverage[path] || coverage[path].hash !== hash) {\n")
	b.WriteString("    coverage[path] = {\n")
	b.WriteString("      path: path,\n")
	b.WriteString("      hash: hash,\n")
	b.WriteString("      statementMap: {},\n")
	b.WriteString("      fnMap: {},\n")
	b.WriteString("      branchMap: {},\n")
	b.WriteString("      s: Array(")
	b.WriteString(strconv.Itoa(statementCount))
	b.WriteString(").fill(0),\n")
	b.WriteString("      f: Array(")
	b.WriteString(strconv.Itoa(functionCount))
	b.WriteString(").fill(0),\n")
	b.WriteString("      b: {},\n")
	b.WriteString("      _coverageSchema: ")
	b.WriteString(strconv.Quote(istanbulCoverageSchema))
	b.WriteString("\n")
	b.WriteString("    };\n")
	b.WriteString("  }\n")
	b.WriteString("  return coverage[path];\n")
	b.WriteString("}\n")
	b.WriteString(covName)
	b.WriteString("();\n")
	return b.String()
}

func stripSourceMappingURL(code string) string {
	return sourceMappingURLRe.ReplaceAllString(code, "")
}

func detectSourceMap(code, inFilePath string) *rawSourceMap {
	m := sourceMappingURLRe.FindStringSubmatch(code)
	if m == nil {
		return nil
	}
	url := strings.TrimSpace(m[1])
	const prefix = "data:application/json;base64,"
	if strings.HasPrefix(url, prefix) {
		raw, err := decodeBase64(url[len(prefix):])
		if err != nil {
			return nil
		}
		var sm rawSourceMap
		if err := json.Unmarshal(raw, &sm); err != nil {
			return nil
		}
		return &sm
	}
	mapPath := filepath.Join(filepath.Dir(inFilePath), url)
	raw, err := os.ReadFile(mapPath)
	if err != nil {
		// Also try sibling .map next to the JS file.
		raw, err = os.ReadFile(inFilePath + ".map")
		if err != nil {
			return nil
		}
	}
	var sm rawSourceMap
	if err := json.Unmarshal(raw, &sm); err != nil {
		return nil
	}
	return &sm
}

func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

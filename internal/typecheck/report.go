// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/buke/typescript-go-internal/v7/pkg/ast"
	"github.com/buke/typescript-go-internal/v7/pkg/core"
	"github.com/buke/typescript-go-internal/v7/pkg/locale"
)

func mapASTDiagnostic(d *ast.Diagnostic) Diagnostic {
	out := Diagnostic{
		Code:     d.Code(),
		Category: normalizeCategory(d.Category()),
		Message:  d.Localize(locale.Default),
		Start:    d.Pos(),
		Length:   d.Len(),
	}
	if f := d.File(); f != nil {
		out.File = filepath.ToSlash(f.FileName())
		line, col := positionToLineColumn(f, d.Pos())
		out.Line = line
		out.Column = col
	}
	return out
}

func toResult(diags []*ast.Diagnostic) Result {
	out := Result{Diagnostics: make([]Diagnostic, 0, len(diags))}
	for _, d := range diags {
		if d == nil {
			continue
		}
		out.Diagnostics = append(out.Diagnostics, mapASTDiagnostic(d))
	}
	return out
}

// FormatStderr writes tsc-like diagnostics to w.
func FormatStderr(w io.Writer, diags []Diagnostic) {
	if w == nil {
		return
	}
	for _, d := range diags {
		loc := d.File
		if d.Line > 0 && d.Column > 0 {
			loc = fmt.Sprintf("%s:%d:%d", d.File, d.Line, d.Column)
		}
		if loc == "" {
			loc = "<unknown>"
		}
		fmt.Fprintf(w, "%s - %s TS%d: %s\n", loc, d.Category, d.Code, d.Message)
	}
}

func positionToLineColumn(file *ast.SourceFile, pos int) (line, column int) {
	if file == nil || pos < 0 {
		return 0, 0
	}
	starts := fileLineMap(file)
	if len(starts) == 0 {
		return 1, pos + 1
	}
	line0, byteOffset := core.PositionToLineAndByteOffset(pos, starts)
	return line0 + 1, byteOffset + 1
}

// Test hook: SourceFile.ECMALineMap never returns an empty slice for real files.
var fileLineMap = func(file *ast.SourceFile) []core.TextPos {
	return file.ECMALineMap()
}

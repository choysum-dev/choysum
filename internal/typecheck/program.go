// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"

	"github.com/buke/typescript-go-internal/v7/pkg/ast"
	"github.com/buke/typescript-go-internal/v7/pkg/compiler"
	"github.com/buke/typescript-go-internal/v7/pkg/core"
	"github.com/buke/typescript-go-internal/v7/pkg/tsoptions"
)

func buildProgram(host compiler.CompilerHost, fileNames []string, opts *core.CompilerOptions) *compiler.Program {
	return compiler.NewProgram(compiler.ProgramOptions{
		Host: host,
		Config: &tsoptions.ParsedCommandLine{
			ParsedConfig: &core.ParsedOptions{
				FileNames:       fileNames,
				CompilerOptions: opts,
			},
		},
		SingleThreaded: core.TSTrue,
	})
}

func collectDiagnostics(ctx context.Context, program *compiler.Program) []*ast.Diagnostic {
	if ctx == nil {
		ctx = context.Background()
	}
	return compiler.GetDiagnosticsOfAnyProgram(
		ctx,
		program,
		nil,
		true,
		program.GetBindDiagnostics,
		program.GetSemanticDiagnostics,
	)
}

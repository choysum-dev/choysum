// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

//go:build ignore

// gen.go generates ExportHub TypeScript stubs via gots.
//
// Invoke via: go generate ./internal/export/web/...

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/bufbuild/protocompile"
	"github.com/choysum-dev/choysum/internal/module/artifact/generate/grpcwebplugin/gots"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintf(os.Stderr, "export_web_generate: cannot resolve caller path\n")
		os.Exit(1)
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	if err := generateExportProto(repoRoot); err != nil {
		fmt.Fprintf(os.Stderr, "export_web_generate: %v\n", err)
		os.Exit(1)
	}
}

func generateExportProto(repoRoot string) error {
	protoRelPath := filepath.ToSlash(filepath.Join("internal", "export", "proto", "export.proto"))
	protoDir := filepath.Join(repoRoot, "internal", "export", "proto")

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{repoRoot, protoDir},
		}),
	}

	fds, err := compiler.Compile(context.Background(), protoRelPath)
	if err != nil {
		return fmt.Errorf("compile %s: %w", protoRelPath, err)
	}

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{protoRelPath},
		Parameter:      proto.String("target=ts"),
	}
	for _, fd := range fds {
		req.ProtoFile = append(req.ProtoFile, protodesc.ToFileDescriptorProto(fd))
	}

	gen := gots.NewGenerator()
	resp, err := gen.Generate(req)
	if err != nil {
		return fmt.Errorf("gots generate: %w", err)
	}

	outDir := filepath.Join(repoRoot, "modules", "core", "web", "export", "pb")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	spdxHeader := `// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

`

	wrote := 0
	for _, file := range resp.GetFile() {
		if file.GetName() == "" {
			continue
		}
		outPath := filepath.Join(outDir, filepath.Base(file.GetName()))
		content := spdxHeader + file.GetContent()
		if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", outPath, err)
		}
		fmt.Printf("export_web_generate: wrote %s\n", outPath)
		wrote++
	}
	if wrote == 0 {
		return fmt.Errorf("gots generated no files")
	}
	return nil
}

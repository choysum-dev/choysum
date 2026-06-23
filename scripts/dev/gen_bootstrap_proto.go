// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

//go:build ignore

// gen_bootstrap_proto.go generates TypeScript type definitions from
// internal/bootstrap/proto/bootstrap.proto using the Go-native gots
// generator. No protoc or protoc-gen-es required.
//
// Invoke via: go run ./scripts/dev/gen_bootstrap_proto.go

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bufbuild/protocompile"
	"github.com/choysum-dev/choysum/internal/module/artifact/generate/grpcwebplugin/gots"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	repoRoot := "."
	protoRelPath := "internal/bootstrap/proto/bootstrap.proto"
	protoDir := "internal/bootstrap/proto"

	// Set up protocompile with the repo root as the import path so the
	// file descriptor records the canonical proto file name.
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{repoRoot, protoDir},
		}),
	}

	fds, err := compiler.Compile(context.Background(), protoRelPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen_bootstrap_proto: compile: %v\n", err)
		os.Exit(1)
	}

	// Build CodeGeneratorRequest.
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{protoRelPath},
		Parameter:      proto.String("target=ts"),
	}
	for _, fd := range fds {
		req.ProtoFile = append(req.ProtoFile, protodesc.ToFileDescriptorProto(fd))
	}

	// Generate TypeScript using gots.
	gen := gots.NewGenerator()
	resp, err := gen.Generate(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen_bootstrap_proto: generate: %v\n", err)
		os.Exit(1)
	}

	// Write output to the canonical location consumed by bootstrap web.
	outDir := filepath.Join(
		repoRoot,
		"internal/bootstrap/web/src/gen/bootstrap/internal/bootstrap/proto",
	)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "gen_bootstrap_proto: mkdir: %v\n", err)
		os.Exit(1)
	}

	spdxHeader := `// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

`

	for _, file := range resp.GetFile() {
		// Use only the base filename so output matches the existing layout.
		outPath := filepath.Join(outDir, filepath.Base(file.GetName()))
		content := spdxHeader + file.GetContent()
		if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "gen_bootstrap_proto: write %s: %v\n", outPath, err)
			os.Exit(1)
		}
		fmt.Printf("gen_bootstrap_proto: wrote %s\n", outPath)
	}
}

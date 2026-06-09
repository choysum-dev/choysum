// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/choysum-dev/choysum/internal/module/artifact/generate/grpcwebplugin"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/pkg/scope"

	"github.com/choysum-dev/choysum/pkg/meta"
	xfmt "golang.org/x/exp/errors/fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/pluginpb"
)

type webGrpcGenerator struct {
	runtimeScope scope.Scope
	module       *meta.IrModule
	plugins      []GrpcPlugin

	// Optional override for pipeline-managed staging.
	modulesProtoDir string
	modulesWebDir   string
}

func (p *webGrpcGenerator) generate(ctx context.Context, results []*module.GeneratorResult) ([]*module.GeneratorResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runtimeOpts := runtimeOptionsFromScope(p.runtimeScope)
	// 1. Find every generated protobuf file.
	var protoFiles []string
	for _, result := range results {
		if result.Name == "protobuf" {
			for _, filePath := range result.OutPaths {
				if strings.HasSuffix(filePath, ".proto") {
					protoFiles = append(protoFiles, filePath)
				}
			}
		}
	}

	// Return the original results when no proto files exist.
	if len(protoFiles) == 0 {
		return results, nil
	}

	// 2. Parse proto files with the buf API and build the full request.
	req, err := p.buildCodeGeneratorRequest(protoFiles)
	if err != nil {
		return results, err
	}

	// 3. Run every plugin.
	var allGenResults []*module.GeneratorResult

	// Resolve the output directory.
	modulesWebDir := p.modulesWebDir
	if modulesWebDir == "" {
		_, webDir, _, err := WorkspaceGeneratedAPITargets(runtimeOpts.modulesPath, p.module.ApplicationStr, runtimeOpts.defaultChoysumPath)
		if err != nil {
			return results, xfmt.Errorf("resolve workspace generated api targets: %w", err)
		}
		modulesWebDir = webDir
	}
	outDir, _ := filepath.Abs(filepath.Join(modulesWebDir, "pb"))

	writeTo := func(dir string) error {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		// Generate code for each plugin.
		for _, plugin := range p.plugins {
			// Invoke the plugin.
			resp, err := plugin.Generate(req)
			if err != nil {
				return err
			}

			// Handle generated files.
			for _, file := range resp.GetFile() {
				fileName := file.GetName()
				content := file.GetContent()

				// Build the output file path.
				outPathStage := filepath.Join(dir, fileName)

				// Ensure the parent directory exists.
				parentDir := filepath.Dir(outPathStage)
				if err := os.MkdirAll(parentDir, 0755); err != nil {
					return err
				}

				// Write the file content.
				if err := os.WriteFile(outPathStage, []byte(content), 0644); err != nil {
					return err
				}

				// Add the generated output using its target path.
				allGenResults = append(allGenResults, &module.GeneratorResult{
					Name:     plugin.Name(),
					OutPaths: []string{filepath.Join(outDir, fileName)},
				})
			}
		}
		return nil
	}

	if p.modulesWebDir != "" {
		if err := writeTo(outDir); err != nil {
			return results, err
		}
	} else {
		if err := staging.WithStagingDir(ctx, outDir, func(stagingDir string) error {
			return writeTo(stagingDir)
		}); err != nil {
			return results, err
		}
	}

	// Return the original results plus the new generated outputs.
	return append(results, allGenResults...), nil
}

// buildRequestWithBuf builds a CodeGeneratorRequest with the protocompile library.
func (p *webGrpcGenerator) buildCodeGeneratorRequest(protoFiles []string) (*pluginpb.CodeGeneratorRequest, error) {
	runtimeOpts := runtimeOptionsFromScope(p.runtimeScope)
	modulesProtoDir := p.modulesProtoDir
	if modulesProtoDir == "" {
		protoDir, _, _, err := WorkspaceGeneratedAPITargets(runtimeOpts.modulesPath, p.module.ApplicationStr, runtimeOpts.defaultChoysumPath)
		if err != nil {
			return nil, xfmt.Errorf("resolve workspace generated api targets: %w", err)
		}
		modulesProtoDir = protoDir
	}
	protoDir, _ := filepath.Abs(modulesProtoDir)
	protoFileNames := make([]string, len(protoFiles))
	for i, file := range protoFiles {
		relPath, _ := filepath.Rel(protoDir, file)
		protoFileNames[i] = relPath
	}

	// Create the compiler.
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{
				".",
				protoDir,
			},
		}),
	}

	// Compile the proto files.
	fileDescriptors, err := compiler.Compile(p.runtimeScope.Context(), protoFileNames...)
	if err != nil {
		return nil, err
	}

	// Create the request object.
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: protoFileNames,
		Parameter:      proto.String(p.buildParameters()),
	}

	// Convert file descriptors into FileDescriptorProto values.
	for _, fileDescriptor := range fileDescriptors {
		fdProto := protodesc.ToFileDescriptorProto(fileDescriptor)
		req.ProtoFile = append(req.ProtoFile, fdProto)
	}

	return req, nil
}

func (p *webGrpcGenerator) buildParameters() string {
	params := []string{
		"target=ts",
	}
	return strings.Join(params, ",")
}

func NewWebGrpcGenerator(runtimeScope scope.Scope, module *meta.IrModule) *webGrpcGenerator {
	gen := &webGrpcGenerator{
		runtimeScope: runtimeScope,
		module:       module,
		plugins: []GrpcPlugin{
			grpcwebplugin.NewGrpcWebPlugin(runtimeScope),
		},
	}
	return gen
}

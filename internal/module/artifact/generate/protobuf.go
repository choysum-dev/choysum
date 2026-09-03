// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"bytes"
	"context"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

//go:embed protobuf_tpl.proto
var tplStr string

//go:embed assets
var protoFS embed.FS

type protobufGenerator struct {
	runtimeScope scope.Scope
	module       *meta.Module

	// Optional overrides for pipeline-managed staging.
	// When set, generator writes directly into these directories and does not commit.
	modulesProtoDir string
	distAppDir      string
}

func (g *protobufGenerator) generate(ctx context.Context, app *meta.Application) ([]*module.GeneratorResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runtimeOpts := runtimeOptionsFromScope(g.runtimeScope)
	var OutPaths []string
	if len(app.Models) == 0 {
		return nil, nil
	}

	modulesProtoDir := g.modulesProtoDir
	if modulesProtoDir == "" {
		protoDir, _, _, err := WorkspaceGeneratedAPITargets(runtimeOpts.modulesPath, g.module.ApplicationStr, runtimeOpts.defaultChoysumPath)
		if err != nil {
			return nil, xfmt.Errorf("resolve workspace generated api targets: %w", err)
		}
		modulesProtoDir = protoDir
	}
	modulesProtoDir, _ = filepath.Abs(modulesProtoDir)

	tpl, err := template.New(app.Name).Funcs(template.FuncMap{
		"index": func(i int, start int) int {
			return i + start
		},
		"needsTimestamp": protobufNeedsTimestamp,
	}).Parse(tplStr)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if err := tpl.Execute(buf, app); err != nil {
		return nil, xfmt.Errorf("error executing template: %w", err)
	}

	writeAll := func(protoDir string) error {
		if err := os.MkdirAll(protoDir, 0o755); err != nil {
			return err
		}
		// Copy embedded proto files into the modules proto directory for generators and frontend code.
		embeddedProtoFiles, err := g.copyEmbeddedProtoFiles(protoDir, protobufNeedsTimestamp(app))
		if err != nil {
			return xfmt.Errorf("error copying embedded proto files: %w", err)
		}
		for _, p := range embeddedProtoFiles {
			rel, err := filepath.Rel(protoDir, p)
			if err != nil {
				return err
			}
			OutPaths = append(OutPaths, filepath.Join(modulesProtoDir, rel))
		}

		protoFilePath := filepath.Join(protoDir, app.Name+".proto")
		if err := os.WriteFile(protoFilePath, buf.Bytes(), 0o644); err != nil {
			return xfmt.Errorf("error writing file: %w", err)
		}
		OutPaths = append(OutPaths, filepath.Join(modulesProtoDir, app.Name+".proto"))
		return nil
	}

	if g.modulesProtoDir != "" {
		if err := writeAll(modulesProtoDir); err != nil {
			return nil, err
		}
		// Sync proto source files to the dist directory for ApplicationService runtime use.
		if err := g.syncProtoToDistDirect(ctx, modulesProtoDir); err != nil {
			return nil, err
		}
	} else {
		if err := staging.WithStagingDir(ctx, modulesProtoDir, func(stagingDir string) error {
			if err := writeAll(stagingDir); err != nil {
				return err
			}
			return g.syncProtoToDist(ctx, stagingDir)
		}); err != nil {
			return nil, err
		}
	}

	return []*module.GeneratorResult{
		{
			Name:     "protobuf",
			OutPaths: OutPaths,
		}}, nil
}

func protobufNeedsTimestamp(app *meta.Application) bool {
	if app == nil {
		return false
	}
	for _, model := range app.Models {
		if model == nil {
			continue
		}
		for _, service := range model.Services {
			if service == nil {
				continue
			}
			if strings.Contains(service.ProtobufType, "google.protobuf.Timestamp") {
				return true
			}
			for _, param := range service.Parameters {
				if param != nil && strings.Contains(param.ProtobufType, "google.protobuf.Timestamp") {
					return true
				}
			}
		}
	}
	return false
}

// copyEmbeddedProtoFiles copies embedded proto assets into destDir.
// timestamp.proto is copied only when includeTimestamp is true.
func (g *protobufGenerator) copyEmbeddedProtoFiles(destDir string, includeTimestamp bool) ([]string, error) {
	var outPaths []string
	err := fs.WalkDir(protoFS, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel("assets", path)
		if err != nil {
			return err
		}
		if !includeTimestamp && filepath.ToSlash(relPath) == "google/protobuf/timestamp.proto" {
			return nil
		}

		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, os.ModePerm)
		}

		data, err := protoFS.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return err
		}

		outPaths = append(outPaths, destPath)
		return nil
	})
	return outPaths, err
}

func (g *protobufGenerator) syncProtoToDist(ctx context.Context, srcDir string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	runtimeOpts := runtimeOptionsFromScope(g.runtimeScope)

	// Bundle mode: runtime protobuf assets are published under .choysum/api/<app>/proto
	// by the module pipeline. Per-app dist sync must be disabled to avoid fallback artifacts.
	if runtimeOpts.isBundleMode() {
		return nil
	}

	distAppDir := g.distAppDir
	if distAppDir == "" {
		distAppDir = filepath.Join(runtimeOpts.distPath, "apps", g.module.ApplicationStr)
	}
	distProtoDir, _ := filepath.Abs(filepath.Join(distAppDir, "assets"))
	return staging.WithStagingDir(ctx, distProtoDir, func(stagingDir string) error {
		return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(srcDir, path)
			if err != nil {
				return err
			}
			targetPath := filepath.Join(stagingDir, relPath)

			if d.IsDir() {
				return os.MkdirAll(targetPath, os.ModePerm)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
				return err
			}
			return os.WriteFile(targetPath, data, 0644)
		})
	})
}

func (g *protobufGenerator) syncProtoToDistDirect(ctx context.Context, srcDir string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	runtimeOpts := runtimeOptionsFromScope(g.runtimeScope)
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Bundle mode: runtime protobuf assets are published under .choysum/api/<app>/proto
	// by the module pipeline. Per-app dist sync must be disabled to avoid fallback artifacts.
	if runtimeOpts.isBundleMode() {
		return nil
	}

	distAppDir := g.distAppDir
	if distAppDir == "" {
		distAppDir = filepath.Join(runtimeOpts.distPath, "apps", g.module.ApplicationStr)
	}
	distProtoDir, _ := filepath.Abs(filepath.Join(distAppDir, "assets"))
	if err := os.MkdirAll(distProtoDir, 0o755); err != nil {
		return err
	}

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(distProtoDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, os.ModePerm)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, 0o644)
	})
}

func newProtobufGenerator(runtimeScope scope.Scope, module *meta.Module) *protobufGenerator {
	return &protobufGenerator{
		runtimeScope: runtimeScope,
		module:       module,
	}
}

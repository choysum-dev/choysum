// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/ettle/strcase"
	xfmt "golang.org/x/exp/errors/fmt"
)

//go:embed serviceclient.ts.tpl
var serviceClientTplStr string

type serviceClientGenerator struct {
	runtimeScope scope.Scope
	module       *meta.Module

	// Optional override for pipeline-managed staging.
	modulesProtoDir   string
	modulesServiceDir string
}

type serviceClientTemplateData struct {
	App        *meta.Application
	ProtoFiles []*protoFilePayload
}

type protoFilePayload struct {
	RegisterPath   string
	EncodedContent string
}

func (g *serviceClientGenerator) generate(ctx context.Context, app *meta.Application) ([]*module.GeneratorResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runtimeOpts := runtimeOptionsFromScope(g.runtimeScope)
	var OutPaths []string
	if len(app.Models) == 0 {
		return nil, nil
	}

	protoFiles, err := g.collectProtoFiles(app.Name)
	if err != nil {
		return nil, err
	}
	if len(protoFiles) == 0 {
		return nil, xfmt.Errorf("no proto files found for app %s", app.Name)
	}

	moduleSpecPath, referenceIdent := meta.BaseModelModuleSpec(g.runtimeScope)
	funcMap := template.FuncMap{
		"ConvertPath": func(path string) string {
			p := strings.ReplaceAll(path, runtimeOpts.modulesPath, "@")
			return strings.TrimSuffix(p, ".ts")
		},
		"ConvertTypeParam": func(model *meta.Model, service *meta.Service) string {
			if len(service.TypeParameters) > 0 {
				typeParams := make([]string, 0, len(service.TypeParameters))
				for _, tp := range service.TypeParameters {
					if tp.ReferenceIdent == referenceIdent && tp.ModuleSpecPath == moduleSpecPath {
						typeParams = append(typeParams, model.Name)
					} else {
						typeParams = append(typeParams, "any")
					}
				}
				return "<" + strings.Join(typeParams, ", ") + ">"
			}
			return ""
		},
		"ToCamel": func(s string) string {
			return strcase.ToCamel(s)
		},
		"ConvertArgs": func(service *meta.Service) string {
			args := make([]string, 0, len(service.Parameters))
			for i, param := range service.Parameters {
				args = append(args, fmt.Sprintf("{ name: '%s', type: '%s', value: args[%d]}", strcase.ToCamel(param.Name), param.ProtobufType, i))
			}
			return "[" + strings.Join(args, ", ") + "]"
		},
		"ConvertReturnType": func(service *meta.Service) string {
			if service.ProtobufType == "google.protobuf.Empty" {
				return "undefined"
			} else {
				return fmt.Sprintf("{ name: '%s', type: '%s' }", "result", service.ProtobufType)
			}
		},
	}

	// Generate Server Client (service/client.ts)
	tpl, err := template.New(app.Name + "_service_client").Funcs(funcMap).Parse(serviceClientTplStr)
	if err != nil {
		return nil, err
	}

	data := &serviceClientTemplateData{
		App:        app,
		ProtoFiles: protoFiles,
	}
	buf := new(bytes.Buffer)
	if err := tpl.Execute(buf, data); err != nil {
		return nil, xfmt.Errorf("error executing service client template: %w", err)
	}

	modulesServiceDir := g.modulesServiceDir
	if modulesServiceDir == "" {
		_, _, serviceDir, err := WorkspaceGeneratedAPITargets(runtimeOpts.modulesPath, g.module.ApplicationStr, runtimeOpts.defaultChoysumPath)
		if err != nil {
			return nil, xfmt.Errorf("resolve workspace generated api targets: %w", err)
		}
		modulesServiceDir = serviceDir
	}
	outDir, _ := filepath.Abs(modulesServiceDir)

	writeTo := func(dir string) error {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		clientTsPath := filepath.Join(dir, "service.ts")
		if err := os.WriteFile(clientTsPath, buf.Bytes(), 0o644); err != nil {
			return err
		}
		OutPaths = append(OutPaths, filepath.Join(outDir, "service.ts"))

		indexContent := "export * from './service';\n"
		indexPath := filepath.Join(dir, "index.ts")
		if err := os.WriteFile(indexPath, []byte(indexContent), 0o644); err != nil {
			return err
		}
		OutPaths = append(OutPaths, filepath.Join(outDir, "index.ts"))
		return nil
	}

	if g.modulesServiceDir != "" {
		if err := writeTo(outDir); err != nil {
			return nil, err
		}
	} else {
		if err := staging.WithStagingDir(ctx, outDir, func(stagingDir string) error {
			return writeTo(stagingDir)
		}); err != nil {
			return nil, err
		}
	}

	return []*module.GeneratorResult{
		{
			Name:     "serviceclient",
			OutPaths: OutPaths,
		}}, nil
}

func (g *serviceClientGenerator) collectProtoFiles(appName string) ([]*protoFilePayload, error) {
	runtimeOpts := runtimeOptionsFromScope(g.runtimeScope)
	modulesProtoDir := g.modulesProtoDir
	if modulesProtoDir == "" {
		protoDir, _, _, err := WorkspaceGeneratedAPITargets(runtimeOpts.modulesPath, g.module.ApplicationStr, runtimeOpts.defaultChoysumPath)
		if err != nil {
			return nil, xfmt.Errorf("resolve workspace generated api targets: %w", err)
		}
		modulesProtoDir = protoDir
	}
	protoDir := modulesProtoDir
	entries := make([]*protoFilePayload, 0)
	err := filepath.WalkDir(protoDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".proto" {
			return nil
		}
		relPath, err := filepath.Rel(protoDir, path)
		if err != nil {
			return err
		}
		registerPath := resolveProtoRegisterPath(appName, relPath)
		if registerPath == "" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		encoded, err := json.Marshal(string(content))
		if err != nil {
			return err
		}
		entries = append(entries, &protoFilePayload{
			RegisterPath:   registerPath,
			EncodedContent: string(encoded),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].RegisterPath < entries[j].RegisterPath
	})
	return entries, nil
}

func resolveProtoRegisterPath(appName, relPath string) string {
	rel := filepath.ToSlash(relPath)
	if strings.HasPrefix(rel, "google/") {
		return rel
	}
	return filepath.ToSlash(filepath.Join(appName, rel))
}

func NewServiceClientGenerator(runtimeScope scope.Scope, module *meta.Module) *serviceClientGenerator {
	return &serviceClientGenerator{
		runtimeScope: runtimeScope,
		module:       module,
	}
}

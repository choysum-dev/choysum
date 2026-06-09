// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/ettle/strcase"
	xfmt "golang.org/x/exp/errors/fmt"
)

//go:embed webservice.ts.tpl
var serviceTplStr string

type webServiceGenerator struct {
	runtimeScope scope.Scope
	module       *meta.IrModule

	// Optional override for pipeline-managed staging.
	modulesWebDir string
}

func (g *webServiceGenerator) generate(app *meta.IrApplication) ([]*module.GeneratorResult, error) {
	runtimeOpts := runtimeOptionsFromScope(g.runtimeScope)
	var OutPaths []string
	if len(app.Models) == 0 {
		return nil, nil
	}

	moduleSpecPath, referenceIdent := meta.BaseModelModuleSpec(g.runtimeScope)
	funcMap := template.FuncMap{
		"ConvertPath": func(path string) string {
			p := strings.ReplaceAll(path, runtimeOpts.modulesPath, "@")
			return strings.TrimSuffix(p, ".ts")
		},
		"ConvertTypeParam": func(model *meta.IrModel, service *meta.IrService) string {
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
		"ConvertArgs": func(service *meta.IrService) string {
			args := make([]string, 0, len(service.Parameters))
			for i, param := range service.Parameters {
				args = append(args, fmt.Sprintf("{ name: '%s', type: '%s', value: args[%d]}", strcase.ToCamel(param.Name), param.ProtobufType, i))
			}
			return "[" + strings.Join(args, ", ") + "]"
		},
		"ConvertReturnType": func(service *meta.IrService) string {
			if service.ProtobufType == "google.protobuf.Empty" {
				return "undefined"
			} else {
				return fmt.Sprintf("{ name: '%s', type: '%s' }", "result", service.ProtobufType)
			}
		},
	}

	// Generate Web Client (web/index.ts)
	webTpl, err := template.New(app.Name + "_web").Funcs(funcMap).Parse(serviceTplStr)
	if err != nil {
		return nil, err
	}

	webBuf := new(bytes.Buffer)
	if err := webTpl.Execute(webBuf, app); err != nil {
		return nil, xfmt.Errorf("error executing web template: %w", err)
	}

	modulesWebDir := g.modulesWebDir
	if modulesWebDir == "" {
		_, webDir, _, err := WorkspaceGeneratedAPITargets(runtimeOpts.modulesPath, g.module.ApplicationStr, runtimeOpts.defaultChoysumPath)
		if err != nil {
			return nil, xfmt.Errorf("resolve workspace generated api targets: %w", err)
		}
		modulesWebDir = webDir
	}
	webOutDir, _ := filepath.Abs(modulesWebDir)
	if err := os.MkdirAll(webOutDir, 0755); err != nil {
		return nil, err
	}

	webServiceTsPath := filepath.Join(webOutDir, "service.ts")
	if err := staging.WriteFileAtomic(webServiceTsPath, webBuf.Bytes(), 0o644); err != nil {
		return nil, err
	}
	OutPaths = append(OutPaths, webServiceTsPath)

	// Generate index.ts
	indexContent := "export * from './service';\nexport * from './stores';\n"
	indexPath := filepath.Join(webOutDir, "index.ts")
	if err := staging.WriteFileAtomic(indexPath, []byte(indexContent), 0o644); err != nil {
		return nil, err
	}
	OutPaths = append(OutPaths, indexPath)

	return []*module.GeneratorResult{
		{
			Name:     "webservice",
			OutPaths: OutPaths,
		}}, nil
}

func NewWebServiceGenerator(runtimeScope scope.Scope, module *meta.IrModule) *webServiceGenerator {
	return &webServiceGenerator{
		runtimeScope: runtimeScope,
		module:       module,
	}
}

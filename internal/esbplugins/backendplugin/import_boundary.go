// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendplugin

import (
	"strings"

	"github.com/choysum-dev/choysum/internal/module/policy"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func (p *BackendPlugin) enforceServiceImportBoundary(parserResults []*parser.ParserResult) error {
	if p == nil || p.Module == nil {
		return nil
	}
	sourceApp := strings.TrimSpace(p.Module.ApplicationStr)
	moduleRoot := strings.TrimSpace(p.Module.Path)
	if sourceApp == "" || moduleRoot == "" {
		return nil
	}

	runtimeOptions := p.resolvedRuntimeOptions()
	lookup := p.moduleApplicationLookup(parserResults, runtimeOptions.modulesPath)
	violations := policy.CheckServiceImportBoundary(policy.ServiceImportBoundaryInput{
		ModulesPath:       runtimeOptions.modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: sourceApp,
		ParserResults:     parserResults,
		Lookup:            lookup,
	})
	return policy.FormatImportBoundaryError(violations)
}

func (p *BackendPlugin) moduleApplicationLookup(parserResults []*parser.ParserResult, modulesPath string) policy.ModuleApplicationLookup {
	appByModule := make(map[string]string)
	if p.Module != nil {
		name := strings.TrimSpace(p.Module.Name)
		app := strings.TrimSpace(p.Module.ApplicationStr)
		if name != "" && app != "" {
			appByModule[name] = app
		}
	}

	moduleNames := make(map[string]struct{})
	for name := range appByModule {
		moduleNames[name] = struct{}{}
	}
	for _, result := range parserResults {
		policy.CollectModuleNamesFromParserResult(modulesPath, result, moduleNames)
	}

	if session := p.Env.Session(); session != nil && len(moduleNames) > 0 {
		names := make([]string, 0, len(moduleNames))
		for name := range moduleNames {
			names = append(names, name)
		}
		var modules []meta.Module
		if err := session.Where("name IN ?", names).Find(&modules).Error; err == nil {
			for _, mod := range modules {
				name := strings.TrimSpace(mod.Name)
				app := strings.TrimSpace(mod.ApplicationStr)
				if name != "" && app != "" {
					appByModule[name] = app
				}
			}
		}
	}

	baseLookup := policy.ModuleApplicationLookupFromMap(appByModule)
	return func(moduleName string) (string, bool) {
		if app, ok := baseLookup(moduleName); ok {
			return app, true
		}
		return policy.ResolveModuleApplication(moduleName, nil)
	}
}

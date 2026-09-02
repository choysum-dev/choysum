// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendplugin

import (
	"os"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/policy"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func (p *BackendPlugin) enforceServiceImportBoundary(parserResults []*parser.ParserResult) error {
	if p == nil || p.BasePlugin == nil || p.Module == nil {
		return nil
	}
	moduleRoot := strings.TrimSpace(p.Module.Path)
	if moduleRoot == "" {
		return nil
	}

	runtimeOptions := p.resolvedRuntimeOptions()
	lookup := p.moduleApplicationLookup(parserResults, runtimeOptions.modulesPath)
	sourceApp := resolveSourceApplication(p.Module, runtimeOptions.modulesPath, lookup)
	if sourceApp == "" {
		return nil
	}
	violations := policy.CheckServiceImportBoundary(policy.ServiceImportBoundaryInput{
		ModulesPath:       runtimeOptions.modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: sourceApp,
		ParserResults:     parserResults,
		Lookup:            lookup,
	})
	return policy.FormatImportBoundaryError(violations)
}

func resolveSourceApplication(module *meta.Module, modulesPath string, lookup policy.ModuleApplicationLookup) string {
	if module == nil {
		return ""
	}
	name := strings.TrimSpace(module.Name)
	if name != "" {
		if app, ok := readModuleApplicationFromExistingPackageJSON(modulesPath, name); ok {
			return app
		}
		if app := strings.TrimSpace(module.ApplicationStr); app != "" {
			return app
		}
		if lookup != nil {
			if app, ok := lookup(name); ok && strings.TrimSpace(app) != "" {
				return strings.TrimSpace(app)
			}
		}
		if app, ok := policy.ResolveModuleApplication(name, nil); ok {
			return app
		}
	}
	return strings.TrimSpace(module.ApplicationStr)
}

func readModuleApplicationFromExistingPackageJSON(modulesPath, moduleName string) (string, bool) {
	modulesPath = strings.TrimSpace(modulesPath)
	moduleName = strings.TrimSpace(moduleName)
	if modulesPath == "" || moduleName == "" {
		return "", false
	}
	app, ok, err := policy.ReadExplicitModuleApplicationFromPackageJSON(modulesPath, moduleName)
	if err != nil || !ok {
		return "", false
	}
	return strings.TrimSpace(app), true
}

func (p *BackendPlugin) moduleApplicationLookup(parserResults []*parser.ParserResult, modulesPath string) policy.ModuleApplicationLookup {
	appByModule := make(map[string]string)
	mergeModuleApplicationsFromDisk(modulesPath, appByModule)

	moduleNames := make(map[string]struct{})
	for name := range appByModule {
		moduleNames[name] = struct{}{}
	}
	if p.Module != nil {
		name := strings.TrimSpace(p.Module.Name)
		if name != "" {
			moduleNames[name] = struct{}{}
		}
	}
	for _, result := range parserResults {
		policy.CollectModuleNamesFromParserResult(modulesPath, result, moduleNames)
	}

	if p.Env != nil && len(moduleNames) > 0 {
		session := p.Env.Session()
		if session != nil {
			names := make([]string, 0, len(moduleNames))
			for name := range moduleNames {
				names = append(names, name)
			}
			var modules []meta.Module
			if err := session.Where("name IN ?", names).Find(&modules).Error; err == nil {
				for _, mod := range modules {
					name := strings.TrimSpace(mod.Name)
					app := strings.TrimSpace(mod.ApplicationStr)
					if name == "" || app == "" {
						continue
					}
					if _, exists := appByModule[name]; exists {
						continue
					}
					appByModule[name] = app
				}
			}
		}
	}

	return policy.ModuleApplicationLookupWithDefault(policy.ModuleApplicationLookupFromMap(appByModule))
}

func mergeModuleApplicationsFromDisk(modulesPath string, appByModule map[string]string) {
	modulesPath = strings.TrimSpace(modulesPath)
	if modulesPath == "" || appByModule == nil {
		return
	}
	entries, err := os.ReadDir(modulesPath)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") || name == "tmp" || name == "node_modules" || name == ".choysum" {
			continue
		}
		app, err := policy.ReadModuleApplicationFromPackageJSON(modulesPath, name)
		if err != nil || strings.TrimSpace(app) == "" {
			continue
		}
		appByModule[name] = strings.TrimSpace(app)
	}
}

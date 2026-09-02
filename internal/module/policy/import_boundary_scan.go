// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/origin/contract"
	"github.com/choysum-dev/choysum/internal/parser"
	xfmt "golang.org/x/exp/errors/fmt"
)

// ServiceImportBoundaryScanInput configures a filesystem scan of one module service tree.
type ServiceImportBoundaryScanInput struct {
	ModulesPath       string
	ModuleName        string
	SourceApplication string
	ModuleRoot        string
	PathAlias         map[string]string
	Lookup            ModuleApplicationLookup
}

var statPath = os.Stat

var walkServiceTree = func(root string, walkFn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, walkFn)
}

// BuildModuleApplicationLookupFromModulesDir indexes choysum.application from each module package.json.
func BuildModuleApplicationLookupFromModulesDir(modulesPath string) (ModuleApplicationLookup, error) {
	modulesPath = strings.TrimSpace(modulesPath)
	if modulesPath == "" {
		return nil, xfmt.Errorf("modules path is required")
	}
	appByModule := make(map[string]string)
	entries, err := os.ReadDir(modulesPath)
	if err != nil {
		return nil, xfmt.Errorf("read modules dir: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		moduleName := strings.TrimSpace(entry.Name())
		if moduleName == "" || shouldSkipModulesDirEntry(moduleName) {
			continue
		}
		app, err := ReadModuleApplicationFromPackageJSON(modulesPath, moduleName)
		if err != nil {
			return nil, err
		}
		if app != "" {
			appByModule[moduleName] = app
		}
	}
	baseLookup := ModuleApplicationLookupFromMap(appByModule)
	return ModuleApplicationLookupWithDefault(baseLookup), nil
}

func shouldSkipModulesDirEntry(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "tmp", "node_modules":
		return true
	default:
		return false
	}
}

// ReadExplicitModuleApplicationFromPackageJSON returns choysum.application when set in package.json.
// ok is false when package.json is missing or the application field is absent/empty.
func ReadExplicitModuleApplicationFromPackageJSON(modulesPath, moduleName string) (application string, ok bool, err error) {
	modulesPath = strings.TrimSpace(modulesPath)
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return "", false, xfmt.Errorf("module name is required")
	}
	if modulesPath == "" {
		return "", false, nil
	}
	packageJSONPath := filepath.Join(modulesPath, moduleName, "package.json")
	data, readErr := os.ReadFile(packageJSONPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return "", false, nil
		}
		return "", false, xfmt.Errorf("read %s package.json: %w", moduleName, readErr)
	}
	pkg, decodeErr := contract.DecodePackageJSON(data)
	if decodeErr != nil {
		return "", false, xfmt.Errorf("decode %s package.json: %w", moduleName, decodeErr)
	}
	app := strings.TrimSpace(pkg.Choysum.Application)
	if app == "" {
		return "", false, nil
	}
	return app, true, nil
}

// ReadModuleApplicationFromPackageJSON reads choysum.application for a module directory.
func ReadModuleApplicationFromPackageJSON(modulesPath, moduleName string) (string, error) {
	modulesPath = strings.TrimSpace(modulesPath)
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return "", xfmt.Errorf("module name is required")
	}
	if modulesPath == "" {
		return DefaultApplicationForModuleName(moduleName), nil
	}
	packageJSONPath := filepath.Join(modulesPath, moduleName, "package.json")
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultApplicationForModuleName(moduleName), nil
		}
		return "", xfmt.Errorf("read %s package.json: %w", moduleName, err)
	}
	pkg, err := contract.DecodePackageJSON(data)
	if err != nil {
		return "", xfmt.Errorf("decode %s package.json: %w", moduleName, err)
	}
	app := strings.TrimSpace(pkg.Choysum.Application)
	if app == "" {
		return DefaultApplicationForModuleName(moduleName), nil
	}
	return app, nil
}

// ParseServiceSourceFile parses imports/exports from one service .ts file.
func ParseServiceSourceFile(pathAlias map[string]string, filePath string, content []byte) (*parser.ParserResult, error) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return nil, xfmt.Errorf("file path is required")
	}
	tsParser := &parser.TsParser{
		Path:      filePath,
		Content:   string(content),
		PathAlias: pathAlias,
	}
	if err := tsParser.ParseImport(nil); err != nil {
		return nil, xfmt.Errorf("parse imports in %s: %w", filePath, err)
	}
	return &parser.ParserResult{
		Path:           filePath,
		RawContent:     string(content),
		Imports:        tsParser.ImportsMap,
		DynamicImports: tsParser.DynamicImports,
		Exports:        tsParser.ExportsMap,
	}, nil
}

// ScanServiceImportBoundaryOnDisk walks moduleRoot/service and applies import boundary rules.
func ScanServiceImportBoundaryOnDisk(input ServiceImportBoundaryScanInput) ([]ImportBoundaryViolation, error) {
	modulesPath := strings.TrimSpace(input.ModulesPath)
	moduleRoot := strings.TrimSpace(input.ModuleRoot)
	sourceApp := strings.TrimSpace(input.SourceApplication)
	if modulesPath == "" || moduleRoot == "" || sourceApp == "" {
		return nil, nil
	}
	serviceRoot := filepath.Join(moduleRoot, "service")
	st, err := statPath(serviceRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, xfmt.Errorf("stat service dir: %w", err)
	}
	if !st.IsDir() {
		return nil, nil
	}

	lookup := input.Lookup
	if lookup == nil {
		var lookupErr error
		lookup, lookupErr = BuildModuleApplicationLookupFromModulesDir(modulesPath)
		if lookupErr != nil {
			return nil, lookupErr
		}
	}

	var parserResults []*parser.ParserResult
	walkErr := walkServiceTree(serviceRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipServiceScanDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if !isServiceTypeScriptSource(path) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return xfmt.Errorf("read %s: %w", path, err)
		}
		result, err := ParseServiceSourceFile(input.PathAlias, path, content)
		if err != nil {
			return err
		}
		parserResults = append(parserResults, result)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	return CheckServiceImportBoundary(ServiceImportBoundaryInput{
		ModulesPath:       modulesPath,
		ModuleRoot:        moduleRoot,
		SourceApplication: sourceApp,
		ParserResults:     parserResults,
		Lookup:            lookup,
	}), nil
}

func shouldSkipServiceScanDir(name string) bool {
	switch name {
	case "node_modules", "dist", ".choysum", "tmp":
		return true
	default:
		return false
	}
}

func isServiceTypeScriptSource(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	if !strings.HasSuffix(name, ".ts") {
		return false
	}
	if strings.HasSuffix(name, ".d.ts") {
		return false
	}
	return true
}

// CheckServiceImportBoundaryOnDisk is the filesystem entry point used by typecheck.
func CheckServiceImportBoundaryOnDisk(modulesPath, moduleName string, pathAlias map[string]string) error {
	modulesPath = strings.TrimSpace(modulesPath)
	moduleName = strings.TrimSpace(moduleName)
	if modulesPath == "" || moduleName == "" {
		return nil
	}
	moduleRoot := filepath.Join(modulesPath, moduleName)
	sourceApp, err := ReadModuleApplicationFromPackageJSON(modulesPath, moduleName)
	if err != nil {
		return err
	}
	violations, err := ScanServiceImportBoundaryOnDisk(ServiceImportBoundaryScanInput{
		ModulesPath:       modulesPath,
		ModuleName:        moduleName,
		SourceApplication: sourceApp,
		ModuleRoot:        moduleRoot,
		PathAlias:         pathAlias,
	})
	if err != nil {
		return err
	}
	return FormatImportBoundaryError(violations)
}

// ModulePathAliasForBoundary returns the @/* alias map used by service import scans.
func ModulePathAliasForBoundary(modulesPath string) map[string]string {
	modulesPath = strings.TrimSpace(modulesPath)
	if modulesPath == "" {
		return nil
	}
	return map[string]string{
		"@/*": filepath.Join(modulesPath, "*"),
	}
}

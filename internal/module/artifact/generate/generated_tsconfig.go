// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"encoding/json"
	"path/filepath"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type workspaceGeneratedTSConfig struct {
	Extends         string                              `json:"extends"`
	CompilerOptions workspaceGeneratedTSCompilerOptions `json:"compilerOptions"`
	Include         []string                            `json:"include"`
}

type workspaceGeneratedTSCompilerOptions struct {
	NoEmit bool `json:"noEmit"`
}

func buildWorkspaceGeneratedTSConfig(modulesPath string, defaultChoysumPath string) ([]byte, error) {
	generatedRoot, err := WorkspaceGeneratedAPIRoot(modulesPath, defaultChoysumPath)
	if err != nil {
		return nil, err
	}
	if absGeneratedRoot, err := filepath.Abs(generatedRoot); err == nil {
		generatedRoot = absGeneratedRoot
	}
	modulesTSConfigPath := filepath.Join(filepath.Clean(modulesPath), "tsconfig.json")
	if absModulesTSConfigPath, err := filepath.Abs(modulesTSConfigPath); err == nil {
		modulesTSConfigPath = absModulesTSConfigPath
	}
	extendsPath, err := filepath.Rel(generatedRoot, modulesTSConfigPath)
	if err != nil {
		return nil, err
	}

	cfg := workspaceGeneratedTSConfig{
		Extends: filepath.ToSlash(extendsPath),
		CompilerOptions: workspaceGeneratedTSCompilerOptions{
			NoEmit: true,
		},
		Include: []string{"./**/*.ts", "./**/*.d.ts"},
	}

	content, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}

func ensureWorkspaceGeneratedTSConfig(runtimeScope scope.Scope) (*module.GeneratorResult, error) {
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	generatedRoot, err := WorkspaceGeneratedAPIRoot(runtimeOpts.modulesPath, runtimeOpts.defaultChoysumPath)
	if err != nil {
		return nil, err
	}
	outPath, err := filepath.Abs(filepath.Join(generatedRoot, "tsconfig.json"))
	if err != nil {
		return nil, err
	}

	content, err := buildWorkspaceGeneratedTSConfig(runtimeOpts.modulesPath, runtimeOpts.defaultChoysumPath)
	if err != nil {
		return nil, err
	}

	if err := staging.WriteFileAtomic(outPath, content, 0o644); err != nil {
		return nil, err
	}

	return &module.GeneratorResult{
		Name:     "generated-tsconfig",
		OutPaths: []string{outPath},
	}, nil
}

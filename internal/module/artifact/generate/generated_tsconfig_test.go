// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

type parsedWorkspaceGeneratedTSConfig struct {
	Extends         string `json:"extends"`
	CompilerOptions struct {
		NoEmit bool `json:"noEmit"`
	} `json:"compilerOptions"`
	Include []string `json:"include"`
}

func TestBuildWorkspaceGeneratedTSConfig_UsesModulesPath(t *testing.T) {
	root := t.TempDir()
	modulesPath := filepath.Join(root, "custom-modules")
	defaultChoysumPath := t.TempDir()

	content, err := buildWorkspaceGeneratedTSConfig(modulesPath, defaultChoysumPath)
	if err != nil {
		t.Fatalf("buildWorkspaceGeneratedTSConfig() error = %v", err)
	}

	var cfg parsedWorkspaceGeneratedTSConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		t.Fatalf("unmarshal generated tsconfig: %v", err)
	}

	generatedRoot, err := WorkspaceGeneratedAPIRoot(modulesPath, defaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPIRoot() error = %v", err)
	}
	if absGeneratedRoot, err := filepath.Abs(generatedRoot); err == nil {
		generatedRoot = absGeneratedRoot
	}
	modulesTSConfigPath := filepath.Join(modulesPath, "tsconfig.json")
	if absModulesTSConfigPath, err := filepath.Abs(modulesTSConfigPath); err == nil {
		modulesTSConfigPath = absModulesTSConfigPath
	}
	wantExtendsRel, err := filepath.Rel(generatedRoot, modulesTSConfigPath)
	if err != nil {
		t.Fatalf("filepath.Rel() error = %v", err)
	}
	wantExtends := filepath.ToSlash(wantExtendsRel)
	if cfg.Extends != wantExtends {
		t.Fatalf("extends = %q, want %q", cfg.Extends, wantExtends)
	}

	if !cfg.CompilerOptions.NoEmit {
		t.Fatal("compilerOptions.noEmit = false, want true")
	}

	wantInclude := []string{"./**/*.ts", "./**/*.d.ts"}
	if !reflect.DeepEqual(cfg.Include, wantInclude) {
		t.Fatalf("include = %#v, want %#v", cfg.Include, wantInclude)
	}
}

func TestBuildWorkspaceGeneratedTSConfig_RelativeModulesPath(t *testing.T) {
	defaultChoysumPath := t.TempDir()
	content, err := buildWorkspaceGeneratedTSConfig("modules", defaultChoysumPath)
	if err != nil {
		t.Fatalf("buildWorkspaceGeneratedTSConfig() error = %v", err)
	}

	var cfg parsedWorkspaceGeneratedTSConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		t.Fatalf("unmarshal generated tsconfig: %v", err)
	}

	generatedRoot, err := WorkspaceGeneratedAPIRoot("modules", defaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPIRoot() error = %v", err)
	}
	if absGeneratedRoot, err := filepath.Abs(generatedRoot); err == nil {
		generatedRoot = absGeneratedRoot
	}
	modulesTSConfigPath := filepath.Join("modules", "tsconfig.json")
	if absModulesTSConfigPath, err := filepath.Abs(modulesTSConfigPath); err == nil {
		modulesTSConfigPath = absModulesTSConfigPath
	}
	wantExtendsRel, err := filepath.Rel(generatedRoot, modulesTSConfigPath)
	if err != nil {
		t.Fatalf("filepath.Rel() error = %v", err)
	}
	wantExtends := filepath.ToSlash(wantExtendsRel)
	if cfg.Extends != wantExtends {
		t.Fatalf("extends = %q, want %q", cfg.Extends, wantExtends)
	}
}

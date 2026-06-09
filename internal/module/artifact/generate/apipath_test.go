// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceGeneratedAPIRoot(t *testing.T) {
	choysumRoot := t.TempDir()
	modulesPath := filepath.Join("tmp", "repo", "modules")
	root, err := WorkspaceGeneratedAPIRoot(modulesPath, choysumRoot)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPIRoot() error = %v", err)
	}
	if root != filepath.Join(choysumRoot, "generated") {
		t.Fatalf("workspace generated root = %q, want %q", root, filepath.Join(choysumRoot, "generated"))
	}
	recomputed, err := WorkspaceGeneratedAPIRoot(modulesPath, choysumRoot)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPIRoot(recompute) error = %v", err)
	}
	if root != recomputed {
		t.Fatalf("workspace generated root should be deterministic")
	}
}

func TestWorkspaceGeneratedAPITargets(t *testing.T) {
	choysumRoot := t.TempDir()
	modulesPath := filepath.Join("tmp", "repo", "modules")
	root, err := WorkspaceGeneratedAPIRoot(modulesPath, choysumRoot)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPIRoot() error = %v", err)
	}
	protoDir, webDir, serviceDir, err := WorkspaceGeneratedAPITargets(modulesPath, "crm", choysumRoot)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPITargets() error = %v", err)
	}

	if protoDir != filepath.Join(root, "proto", "crm") {
		t.Fatalf("proto dir = %q", protoDir)
	}
	if webDir != filepath.Join(root, "web", "crm") {
		t.Fatalf("web dir = %q", webDir)
	}
	if serviceDir != filepath.Join(root, "service", "crm") {
		t.Fatalf("service dir = %q", serviceDir)
	}
}

func TestWorkspaceGeneratedAPIRoot_RequiresDefaultChoysumPath(t *testing.T) {
	if _, err := WorkspaceGeneratedAPIRoot(filepath.Join("tmp", "repo", "modules"), ""); err == nil {
		t.Fatal("expected WorkspaceGeneratedAPIRoot to reject empty defaultChoysumPath")
	}
}

func TestWorkspaceGeneratedAPIRoot_RejectsRootAndDotLikePaths(t *testing.T) {
	tests := []string{string(filepath.Separator), " " + string(filepath.Separator) + " "}
	for _, input := range tests {
		t.Run(strings.ReplaceAll(input, " ", "_"), func(t *testing.T) {
			if _, err := WorkspaceGeneratedAPIRoot(filepath.Join("tmp", "repo", "modules"), input); err == nil {
				t.Fatalf("expected WorkspaceGeneratedAPIRoot to reject %q", input)
			}
		})
	}
}

func TestWorkspaceGeneratedAPIDirs(t *testing.T) {
	choysumRoot := t.TempDir()
	modulesPath := filepath.Join(t.TempDir(), "modules")
	appName := "auth"

	protoDir, err := workspaceGeneratedAPIProtoDir(modulesPath, appName, choysumRoot)
	if err != nil {
		t.Fatalf("workspaceGeneratedAPIProtoDir() error = %v", err)
	}
	webDir, err := workspaceGeneratedAPIWebDir(modulesPath, appName, choysumRoot)
	if err != nil {
		t.Fatalf("workspaceGeneratedAPIWebDir() error = %v", err)
	}
	serviceDir, err := workspaceGeneratedAPIServiceDir(modulesPath, appName, choysumRoot)
	if err != nil {
		t.Fatalf("workspaceGeneratedAPIServiceDir() error = %v", err)
	}

	root := filepath.Join(choysumRoot, "generated")
	if protoDir != filepath.Join(root, "proto", appName) {
		t.Fatalf("proto dir = %q", protoDir)
	}
	if webDir != filepath.Join(root, "web", appName) {
		t.Fatalf("web dir = %q", webDir)
	}
	if serviceDir != filepath.Join(root, "service", appName) {
		t.Fatalf("service dir = %q", serviceDir)
	}
}

func TestWorkspaceGeneratedAPIRoot_UsesContractNamespaceOnly(t *testing.T) {
	choysumRoot := t.TempDir()
	modulesPath := filepath.Join(t.TempDir(), "modules")

	root, err := WorkspaceGeneratedAPIRoot(modulesPath, choysumRoot)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPIRoot() error = %v", err)
	}
	if strings.Contains(filepath.ToSlash(root), "/workspaces/") {
		t.Fatalf("expected generated root without workspaces segment, got %q", root)
	}
}

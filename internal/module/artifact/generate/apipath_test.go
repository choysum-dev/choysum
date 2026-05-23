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
	addonsPath := filepath.Join("tmp", "repo", "addons")
	root, err := WorkspaceGeneratedAPIRoot(addonsPath, choysumRoot)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPIRoot() error = %v", err)
	}
	if root != filepath.Join(choysumRoot, "generated") {
		t.Fatalf("workspace generated root = %q, want %q", root, filepath.Join(choysumRoot, "generated"))
	}
	recomputed, err := WorkspaceGeneratedAPIRoot(addonsPath, choysumRoot)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPIRoot(recompute) error = %v", err)
	}
	if root != recomputed {
		t.Fatalf("workspace generated root should be deterministic")
	}
}

func TestWorkspaceGeneratedAPITargets(t *testing.T) {
	choysumRoot := t.TempDir()
	addonsPath := filepath.Join("tmp", "repo", "addons")
	root, err := WorkspaceGeneratedAPIRoot(addonsPath, choysumRoot)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPIRoot() error = %v", err)
	}
	protoDir, webDir, serviceDir, err := WorkspaceGeneratedAPITargets(addonsPath, "crm", choysumRoot)
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
	if _, err := WorkspaceGeneratedAPIRoot(filepath.Join("tmp", "repo", "addons"), ""); err == nil {
		t.Fatal("expected WorkspaceGeneratedAPIRoot to reject empty defaultChoysumPath")
	}
}

func TestWorkspaceGeneratedAPIRoot_UsesContractNamespaceOnly(t *testing.T) {
	choysumRoot := t.TempDir()
	addonsPath := filepath.Join(t.TempDir(), "addons")

	root, err := WorkspaceGeneratedAPIRoot(addonsPath, choysumRoot)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPIRoot() error = %v", err)
	}
	if strings.Contains(filepath.ToSlash(root), "/workspaces/") {
		t.Fatalf("expected generated root without workspaces segment, got %q", root)
	}
}

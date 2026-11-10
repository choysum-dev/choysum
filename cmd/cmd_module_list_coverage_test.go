// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"bytes"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	cliruntime "github.com/choysum-dev/choysum/internal/cli/runtime"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/spf13/cobra"
)

func TestRunModuleListLocal_PrintsSourceSnapshotViews(t *testing.T) {
	runtimeScope, db := newModuleQueryTestScope(t)
	if err := db.AutoMigrate(&modmeta.ModuleSourceSnapshot{}, &meta.Module{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "auth",
		SourceType: "local",
		SourceRef:  "modules/auth",
		Available:  true,
	}).Error; err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runModuleListLocal(cmd, runtimeScope, cliruntime.Options{}); err != nil {
		t.Fatalf("runModuleListLocal() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "auth") || !strings.Contains(out, "local") || !strings.Contains(out, "modules/auth") {
		t.Fatalf("unexpected list output: %q", out)
	}
}

func TestQueryModuleIndexViews_QueryError(t *testing.T) {
	runtimeScope, db := newModuleQueryTestScope(t)
	if err := db.Exec(`CREATE TABLE meta_module_source_snapshot (id TEXT)`).Error; err != nil {
		t.Fatalf("create incomplete table: %v", err)
	}

	_, _, err := queryModuleIndexViews(runtimeScope, "")
	if err == nil || !strings.Contains(err.Error(), "query module source snapshot failed") {
		t.Fatalf("error = %v, want query module source snapshot failed", err)
	}
}

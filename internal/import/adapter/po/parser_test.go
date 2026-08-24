// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package po

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	termplan "github.com/choysum-dev/choysum/internal/import/plan/term"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestBuildTermPlan_EnumeratesPOFiles(t *testing.T) {
	root := t.TempDir()
	i18nDir := filepath.Join(root, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"zh_CN.po", "en_US.po", "readme.txt"} {
		if err := os.WriteFile(filepath.Join(i18nDir, name), []byte("msgid \"\"\nmsgstr \"\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	p, err := BuildTermPlan(root, "", "auth", "demo")
	if err != nil {
		t.Fatalf("BuildTermPlan: %v", err)
	}
	if len(p.Units) != 2 {
		t.Fatalf("units = %d, want 2", len(p.Units))
	}
	u0 := p.Units[0].(termplan.Unit)
	u1 := p.Units[1].(termplan.Unit)
	if u0.Lang != "en_US" || u1.Lang != "zh_CN" {
		t.Fatalf("langs = %q,%q", u0.Lang, u1.Lang)
	}
}

func TestBuildTermPlan_MissingI18nDir(t *testing.T) {
	p, err := BuildTermPlan(t.TempDir(), "", "auth", "demo")
	if err != nil {
		t.Fatalf("BuildTermPlan missing dir: %v", err)
	}
	if len(p.Units) != 0 {
		t.Fatalf("units = %d, want 0", len(p.Units))
	}
}

func TestBuilder_BuildRequiresTerminology(t *testing.T) {
	_, err := (Builder{}).Build(context.Background(), importpkg.Spec{
		Profile: importpkg.ProfileInitdata,
		Source:  importpkg.Source{Format: Format, Path: t.TempDir()},
		Module:  "demo",
	})
	if err == nil {
		t.Fatal("expected profile error")
	}
}

func TestBuildTermPlan_FilterLang(t *testing.T) {
	root := t.TempDir()
	i18nDir := filepath.Join(root, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"zh_CN.po", "en_US.po"} {
		if err := os.WriteFile(filepath.Join(i18nDir, name), []byte("msgid \"\"\nmsgstr \"\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p, err := BuildTermPlan(root, "zh_CN", "auth", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Units) != 1 || p.Units[0].(termplan.Unit).Lang != "zh_CN" {
		t.Fatalf("units = %+v", p.Units)
	}
}

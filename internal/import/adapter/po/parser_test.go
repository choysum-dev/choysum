// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package po

import (
	"context"
	"errors"
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

func TestBuildTermPlan_EmptyModuleRoot(t *testing.T) {
	_, err := BuildTermPlan("  ", "", "auth", "demo")
	if err == nil {
		t.Fatal("expected empty module root error")
	}
}

func TestBuildTermPlan_ReadDirError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "i18n"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := BuildTermPlan(root, "", "auth", "demo")
	if err == nil {
		t.Fatal("expected read i18n dir error")
	}
}

func TestBuildTermPlan_SkipsSubdirAndEmptyLang(t *testing.T) {
	root := t.TempDir()
	i18nDir := filepath.Join(root, "i18n")
	if err := os.MkdirAll(filepath.Join(i18nDir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, ".po"), []byte("msgid \"\"\nmsgstr \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "fr.po"), []byte("msgid \"\"\nmsgstr \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := BuildTermPlan(root, "", "auth", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Units) != 1 || p.Units[0].(termplan.Unit).Lang != "fr" {
		t.Fatalf("units = %+v", p.Units)
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

func TestBuilder_BuildRequiresPathModuleApplication(t *testing.T) {
	base := importpkg.Spec{
		Profile:     importpkg.ProfileTerminology,
		Application: "auth",
		Module:      "demo",
		Source:      importpkg.Source{Format: Format, Path: t.TempDir()},
	}
	for _, spec := range []importpkg.Spec{
		{Profile: base.Profile, Application: base.Application, Module: base.Module, Source: importpkg.Source{Format: Format}},
		{Profile: base.Profile, Application: base.Application, Source: base.Source},
		{Profile: base.Profile, Module: base.Module, Source: base.Source},
	} {
		if _, err := (Builder{}).Build(context.Background(), spec); err == nil {
			t.Fatalf("expected validation error for spec %+v", spec)
		}
	}
}

func TestBuilder_BuildSuccess(t *testing.T) {
	root := t.TempDir()
	i18nDir := filepath.Join(root, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte("msgid \"\"\nmsgstr \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := (Builder{}).Build(context.Background(), importpkg.Spec{
		Profile:     importpkg.ProfileTerminology,
		Application: "auth",
		Module:      "demo",
		Source:      importpkg.Source{Format: Format, Path: root},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(p.Units) != 1 {
		t.Fatalf("units = %d", len(p.Units))
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

func TestBuildTermPlan_trimsInputs(t *testing.T) {
	root := t.TempDir()
	i18nDir := filepath.Join(root, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "de.po"), []byte("msgid \"\"\nmsgstr \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := BuildTermPlan("  "+root+"  ", " de ", " auth ", " demo ")
	if err != nil {
		t.Fatal(err)
	}
	u := p.Units[0].(termplan.Unit)
	if u.Application != "auth" || u.ModuleName != "demo" || u.Lang != "de" {
		t.Fatalf("unit = %+v", u)
	}
}

func TestBuilder_BuildWrapsBuildTermPlanError(t *testing.T) {
	_, err := (Builder{}).Build(context.Background(), importpkg.Spec{
		Profile:     importpkg.ProfileTerminology,
		Application: "auth",
		Module:      "demo",
		Source:      importpkg.Source{Format: Format, Path: "  "},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) {
		t.Fatalf("error = %T", err)
	}
}

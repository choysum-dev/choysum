// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestToTermSpecNilRequest(t *testing.T) {
	_, err := toTermSpec(nil)
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckTerminologyExportAccessRequiresFields(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	ctx := authCtx(t)
	cases := []struct {
		app, module, lang string
	}{
		{"", "base", "zh_CN"},
		{"auth", "", "zh_CN"},
		{"auth", "base", ""},
	}
	for _, tc := range cases {
		err := checkTerminologyExportAccess(ctx, runtimeScope, tc.app, tc.module, tc.lang)
		if err == nil || status.Code(err) != codes.InvalidArgument {
			t.Fatalf("checkTerminologyExportAccess(%q,%q,%q) err = %v", tc.app, tc.module, tc.lang, err)
		}
	}
}

func TestCheckTerminologyExportAccessModuleNotInApp(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedInstalledModule(t, runtimeScope, "auth", "base")
	ctx := authCtx(t)
	err := checkTerminologyExportAccess(ctx, runtimeScope, "auth", "web", "zh_CN")
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckTerminologyExportAccessDenied(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedInstalledModule(t, runtimeScope, "auth", "base")
	ctx := authCtxWithServer(t, denyAuthServer{})
	err := checkTerminologyExportAccess(ctx, runtimeScope, "auth", "base", "zh_CN")
	if err == nil || status.Code(err) != codes.PermissionDenied {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckTerminologyExportAccessAuthRPCError(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedInstalledModule(t, runtimeScope, "auth", "base")
	ctx := authCtxWithServer(t, errAuthServer{})
	err := checkTerminologyExportAccess(ctx, runtimeScope, "auth", "base", "zh_CN")
	if err == nil || status.Code(err) != codes.PermissionDenied {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckTerminologyExportAccessAllowed(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedInstalledModule(t, runtimeScope, "auth", "base")
	ctx := authCtx(t)
	if err := checkTerminologyExportAccess(ctx, runtimeScope, "auth", "base", "zh_CN"); err != nil {
		t.Fatalf("checkTerminologyExportAccess: %v", err)
	}
}

func TestInstalledModulesByAppNilScope(t *testing.T) {
	got, err := installedModulesByApp(nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}

func TestInstalledModulesByAppNoModuleTable(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	got, err := installedModulesByApp(runtimeScope)
	if err != nil || len(got) != 0 {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}

func TestInstalledModulesByAppFrameworkModuleInjection(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	for _, mod := range []meta.Module{
		{Name: "core", ApplicationStr: "core", Status: meta.Installed, Path: "/tmp/core"},
		{Name: "base", ApplicationStr: "auth", Status: meta.Installed, Path: "/tmp/base"},
	} {
		if err := db.Create(&mod).Error; err != nil {
			t.Fatalf("seed module: %v", err)
		}
	}

	got, err := installedModulesByApp(runtimeScope)
	if err != nil {
		t.Fatalf("installedModulesByApp: %v", err)
	}
	modules := got["auth"]
	foundCore := false
	for _, name := range modules {
		if name == frameworkModuleName {
			foundCore = true
		}
	}
	if !foundCore {
		t.Fatalf("modules = %#v, want core injected", modules)
	}
}

func TestInstalledModulesByAppSkipsEmptyApplicationAndName(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	for _, mod := range []meta.Module{
		{Name: "base", ApplicationStr: "auth", Status: meta.Installed, Path: "/tmp/base"},
		{Name: "", ApplicationStr: "auth", Status: meta.Installed, Path: "/tmp/empty"},
		{Name: "web", ApplicationStr: "", Status: meta.Installed, Path: "/tmp/web"},
		{Name: "core", ApplicationStr: "core", Status: meta.Installed, Path: "/tmp/core"},
	} {
		if err := db.Create(&mod).Error; err != nil {
			t.Fatalf("seed module: %v", err)
		}
	}

	got, err := installedModulesByApp(runtimeScope)
	if err != nil {
		t.Fatalf("installedModulesByApp: %v", err)
	}
	if len(got["auth"]) != 2 {
		t.Fatalf("auth modules = %#v, want base+core", got["auth"])
	}
	if len(got[""]) != 0 && len(got["core"]) != 0 {
		t.Fatalf("unexpected apps = %#v", got)
	}
}

func TestInstalledModulesByAppDBError(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	if err := runtimeScope.Session().DB.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	prev := installedModulesByAppDBHook
	installedModulesByAppDBHook = func(_ *gorm.DB, _ *[]meta.Module) error {
		return errors.New("db down")
	}
	t.Cleanup(func() { installedModulesByAppDBHook = prev })

	_, err := installedModulesByApp(runtimeScope)
	if err == nil {
		t.Fatal("expected db error")
	}
}

func TestModuleBelongsToApp(t *testing.T) {
	if moduleBelongsToApp([]string{"base"}, "base") != true {
		t.Fatal("expected match")
	}
	if moduleBelongsToApp([]string{"base"}, " ") != false {
		t.Fatal("expected empty module to fail")
	}
	if moduleBelongsToApp(nil, "base") != false {
		t.Fatal("expected missing module to fail")
	}
}

func TestInstalledModulesByAppSkipsDuplicateNames(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	if err := runtimeScope.Session().DB.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	prev := installedModulesByAppDBHook
	installedModulesByAppDBHook = func(_ *gorm.DB, dest *[]meta.Module) error {
		*dest = []meta.Module{
			{Name: "base", ApplicationStr: "auth", Status: meta.Installed},
			{Name: "base", ApplicationStr: "auth", Status: meta.Installed},
			{Name: "core", ApplicationStr: "core", Status: meta.Installed},
		}
		return nil
	}
	t.Cleanup(func() { installedModulesByAppDBHook = prev })

	got, err := installedModulesByApp(runtimeScope)
	if err != nil {
		t.Fatalf("installedModulesByApp: %v", err)
	}
	if len(got["auth"]) != 2 {
		t.Fatalf("auth modules = %#v, want base+core", got["auth"])
	}
}

func TestCheckTerminologyExportAccessCatalogLookupError(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	if err := runtimeScope.Session().DB.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	prev := installedModulesByAppDBHook
	installedModulesByAppDBHook = func(_ *gorm.DB, _ *[]meta.Module) error {
		return errors.New("catalog down")
	}
	t.Cleanup(func() { installedModulesByAppDBHook = prev })

	ctx := authCtx(t)
	err := checkTerminologyExportAccess(ctx, runtimeScope, "auth", "base", "zh_CN")
	if err == nil || status.Code(err) != codes.Internal {
		t.Fatalf("err = %v", err)
	}
}

func TestToTermSpecTrimsFields(t *testing.T) {
	spec, err := toTermSpec(&exportpb.ExportRunRequest{
		Application: " auth ",
		Module:      " base ",
		Lang:        " zh_CN ",
	})
	if err != nil {
		t.Fatalf("toTermSpec: %v", err)
	}
	if spec.Application != "auth" || spec.Module != "base" || spec.Lang != "zh_CN" {
		t.Fatalf("spec = %+v", spec)
	}
}

type nilSessionScope struct {
	scope.Scope
}

func (nilSessionScope) Session() *scope.Session { return nil }

func TestInstalledModulesByAppNilSession(t *testing.T) {
	got, err := installedModulesByApp(nilSessionScope{})
	if err != nil || len(got) != 0 {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}

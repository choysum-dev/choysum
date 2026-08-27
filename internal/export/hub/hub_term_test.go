// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	"github.com/choysum-dev/choysum/internal/export/registry"
	"github.com/choysum-dev/choysum/pkg/auth"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func seedInstalledModule(t *testing.T, runtimeScope scope.Scope, app, name string) {
	t.Helper()
	if err := runtimeScope.Session().DB.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("AutoMigrate modules: %v", err)
	}
	if err := runtimeScope.Session().DB.Create(&meta.Module{
		Name:           name,
		ApplicationStr: app,
		Status:         meta.Installed,
		Path:           "/tmp/" + name,
	}).Error; err != nil {
		t.Fatalf("seed module %s/%s: %v", app, name, err)
	}
}

func TestToTermSpec(t *testing.T) {
	spec, err := toTermSpec(&exportpb.ExportRunRequest{
		Application: "auth",
		Module:      "base",
		Lang:        "zh_CN",
	})
	if err != nil {
		t.Fatalf("toTermSpec: %v", err)
	}
	if spec.Profile != exportpkg.ProfileTerminology || spec.Caller != exportpkg.CallerUser {
		t.Fatalf("spec = %+v", spec)
	}
	if spec.Application != "auth" || spec.Module != "base" || spec.Lang != "zh_CN" || spec.Format != "po" {
		t.Fatalf("spec = %+v", spec)
	}
}

func TestToSpecRejectsUnknownProfile(t *testing.T) {
	_, err := toSpec(&exportpb.ExportRunRequest{Profile: "initdata", Model: "base.Country"}, false)
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestRun_Terminology_AssemblesSpec(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedInstalledModule(t, runtimeScope, "auth", "base")
	ctx := auth.ContextWithAccessToken(authCtx(t), "user-token")

	var captured exportpkg.Spec
	h := New(Deps{
		RuntimeScope: runtimeScope,
		Run: func(_ context.Context, _ scope.Scope, spec exportpkg.Spec) (importpkg.Report, error) {
			captured = spec
			return importpkg.Report{
				Profile: importpkg.ProfileTerminology,
				Stats:   importpkg.Stats{Ok: 2, Total: 2},
			}, nil
		},
	})

	_, err := h.Run(ctx, &exportpb.ExportRunRequest{
		Profile:     "terminology",
		Application: "auth",
		Module:      "base",
		Lang:        "zh_CN",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if captured.Profile != exportpkg.ProfileTerminology || captured.Caller != exportpkg.CallerUser {
		t.Fatalf("captured = %+v", captured)
	}
	if captured.Application != "auth" || captured.Module != "base" || captured.Lang != "zh_CN" {
		t.Fatalf("captured = %+v", captured)
	}
}

func TestRun_TerminologyReturnsPOBytes(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedInstalledModule(t, runtimeScope, "auth", "base")
	ctx := auth.ContextWithAccessToken(authCtx(t), "user-token")
	prev := runExportWithResultFn
	runExportWithResultFn = func(_ context.Context, _ scope.Scope, _ exportpkg.Spec) (importpkg.Report, registry.Result, error) {
		return importpkg.Report{
			Profile: importpkg.ProfileTerminology,
			Stats:   importpkg.Stats{Ok: 1, Total: 1},
		}, registry.Result{POBytes: []byte(`msgid "Hello"` + "\n")}, nil
	}
	t.Cleanup(func() { runExportWithResultFn = prev })

	resp, err := runExport(ctx, Deps{
		RuntimeScope: runtimeScope,
		JSExecutor:   stubJSExecutor{},
	}, &exportpb.ExportRunRequest{
		Profile:     "terminology",
		Application: "auth",
		Module:      "base",
		Lang:        "zh_CN",
	}, false)
	if err != nil {
		t.Fatalf("runExport: %v", err)
	}
	if string(resp.GetPoData()) != `msgid "Hello"`+"\n" {
		t.Fatalf("po_data = %q", resp.GetPoData())
	}
}

func TestPreview_TerminologyUnsupported(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedInstalledModule(t, runtimeScope, "auth", "base")
	ctx := auth.ContextWithAccessToken(authCtx(t), "user-token")
	h := New(Deps{RuntimeScope: runtimeScope})
	_, err := h.Preview(ctx, &exportpb.ExportRunRequest{
		Profile:     "terminology",
		Application: "auth",
		Module:      "base",
		Lang:        "zh_CN",
	})
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckTerminologyExportAccessUnknownApplication(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	err := checkTerminologyExportAccess(runtimeScope, "missing", "base", "zh_CN")
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckTerminologyExportAccessInvalidLang(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedInstalledModule(t, runtimeScope, "auth", "base")
	err := checkTerminologyExportAccess(runtimeScope, "auth", "base", "bad lang!")
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestAttachInlinePO(t *testing.T) {
	resp := &exportpb.ExportRunResponse{}
	large := make([]byte, maxInlinePOBytes+1)
	attachInlinePO(resp, large)
	if len(resp.GetPoData()) != 0 {
		t.Fatalf("po_data len = %d, want 0", len(resp.GetPoData()))
	}
	attachInlinePO(resp, []byte("msgid \"x\"\n"))
	if string(resp.GetPoData()) != "msgid \"x\"\n" {
		t.Fatalf("po_data = %q", resp.GetPoData())
	}
}

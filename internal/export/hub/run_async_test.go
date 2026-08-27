// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubExportEnqueueExecutor struct {
	stubJSExecutor
	result map[string]any
	raw    any
	err    error
}

func (s stubExportEnqueueExecutor) Execute(_ context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	if req != nil && req.Service == enqueueExportDataTransferJobService {
		if s.err != nil {
			return nil, s.err
		}
		if s.raw != nil {
			return &jsengine.JsResponse{Result: s.raw}, nil
		}
		return &jsengine.JsResponse{Result: s.result}, nil
	}
	return s.stubJSExecutor.Execute(context.Background(), req)
}

type exportFailMarshalResult struct{}

func (exportFailMarshalResult) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal failed")
}

func TestRunAsync_CreatesDataTransferJob_DirectionExport(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	h := New(Deps{
		RuntimeScope: runtimeScope,
		JSExecutor: stubExportEnqueueExecutor{
			result: map[string]any{
				"dataTransferJobId": "ej-1",
				"taskJobId":         "tj-1",
			},
		},
	})

	resp, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{
		Run: &exportpb.ExportRunRequest{
			Model:     "base.Country",
			CompanyId: "cmp_test",
		},
	})
	if err != nil {
		t.Fatalf("RunAsync: %v", err)
	}
	if resp.GetDataTransferJobId() != "ej-1" || resp.GetTaskJobId() != "tj-1" {
		t.Fatalf("response = %+v", resp)
	}
	if resp.GetReport() == nil || resp.GetReport().GetMeta().GetTargetModel() != "base.Country" {
		t.Fatalf("report = %+v", resp.GetReport())
	}
}

func TestRunAsync_RequiresExecutor(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)
	h := New(Deps{RuntimeScope: runtimeScope})
	_, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{
		Run: &exportpb.ExportRunRequest{Model: "base.Country"},
	})
	if err == nil || status.Code(err) != codes.Unavailable {
		t.Fatalf("RunAsync err = %v", err)
	}
}

func TestRunAsync_ErrorPaths(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	ctx := authCtx(t)

	t.Run("terminology unsupported", func(t *testing.T) {
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubExportEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{
			Run: &exportpb.ExportRunRequest{
				Profile:     "terminology",
				Application: "auth",
				Module:      "base",
				Lang:        "zh_CN",
			},
		})
		if err == nil || status.Code(err) != codes.InvalidArgument {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("missing model", func(t *testing.T) {
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubExportEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{
			Run: &exportpb.ExportRunRequest{},
		})
		if err == nil || status.Code(err) != codes.InvalidArgument {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("enqueue incomplete ids", func(t *testing.T) {
		h := New(Deps{
			RuntimeScope: runtimeScope,
			JSExecutor: stubExportEnqueueExecutor{result: map[string]any{
				"dataTransferJobId": "ej-only",
			}},
		})
		_, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{
			Run: &exportpb.ExportRunRequest{Model: "base.Country"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("job record from spec error", func(t *testing.T) {
		orig := exportDataTransferJobRecordFromSpec
		t.Cleanup(func() { exportDataTransferJobRecordFromSpec = orig })
		exportDataTransferJobRecordFromSpec = func(exportpkg.Spec) (exportpkg.DataTransferJobRecord, error) {
			return exportpkg.DataTransferJobRecord{}, errors.New("snapshot failed")
		}
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubExportEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{
			Run: &exportpb.ExportRunRequest{Model: "base.Country"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestToAsyncRecordSpec(t *testing.T) {
	spec, err := toAsyncRecordSpec(&exportpb.ExportRunRequest{Model: "base.Country"})
	if err != nil || !spec.Async || spec.Model != "base.Country" {
		t.Fatalf("spec=%+v err=%v", spec, err)
	}
	if _, err := toAsyncRecordSpec(&exportpb.ExportRunRequest{
		Profile:     "terminology",
		Application: "auth",
		Module:      "base",
		Lang:        "zh_CN",
	}); err == nil {
		t.Fatal("expected terminology async error")
	}
}

func TestExportSchedulerUserID(t *testing.T) {
	if got := exportSchedulerUserID(context.Background()); got != "" {
		t.Fatalf("exportSchedulerUserID = %q", got)
	}
	if got := exportSchedulerUserID(authCtx(t)); got == "" {
		t.Fatal("expected user id")
	}
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"github.com/choysum-dev/choysum/pkg/auth"
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

	t.Run("unsupported profile", func(t *testing.T) {
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubExportEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{
			Run: &exportpb.ExportRunRequest{
				Profile: "initdata",
				Model:   "base.Country",
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

	t.Run("nil scope", func(t *testing.T) {
		h := New(Deps{JSExecutor: stubExportEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{
			Run: &exportpb.ExportRunRequest{Model: "base.Country"},
		})
		if err == nil || status.Code(err) != codes.Unavailable {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("nil request", func(t *testing.T) {
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubExportEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, nil)
		if err == nil || status.Code(err) != codes.InvalidArgument {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("invalid spec", func(t *testing.T) {
		orig := validateExportSpec
		t.Cleanup(func() { validateExportSpec = orig })
		validateExportSpec = func(exportpkg.Spec) error { return errors.New("bad spec") }
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubExportEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{
			Run: &exportpb.ExportRunRequest{Model: "base.Country"},
		})
		if err == nil || status.Code(err) != codes.InvalidArgument {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("spec snapshot decode error", func(t *testing.T) {
		orig := exportJSONUnmarshal
		t.Cleanup(func() { exportJSONUnmarshal = orig })
		exportJSONUnmarshal = func([]byte, any) error { return errors.New("decode failed") }
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubExportEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{
			Run: &exportpb.ExportRunRequest{Model: "base.Country"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("enqueue execute error", func(t *testing.T) {
		h := New(Deps{
			RuntimeScope: runtimeScope,
			JSExecutor:   stubExportEnqueueExecutor{err: errors.New("exec failed")},
		})
		_, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{
			Run: &exportpb.ExportRunRequest{Model: "base.Country"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("enqueue empty result", func(t *testing.T) {
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: exportNilResponseExecutor{}})
		_, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{
			Run: &exportpb.ExportRunRequest{Model: "base.Country"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("enqueue marshal failure", func(t *testing.T) {
		h := New(Deps{
			RuntimeScope: runtimeScope,
			JSExecutor:   stubExportEnqueueExecutor{raw: exportFailMarshalResult{}},
		})
		_, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{
			Run: &exportpb.ExportRunRequest{Model: "base.Country"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("enqueue decode failure", func(t *testing.T) {
		h := New(Deps{
			RuntimeScope: runtimeScope,
			JSExecutor:   stubExportEnqueueExecutor{raw: 42},
		})
		_, err := h.RunAsync(ctx, &exportpb.ExportRunAsyncRequest{
			Run: &exportpb.ExportRunRequest{Model: "base.Country"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("access denied", func(t *testing.T) {
		seedPartnerModelMeta(t, runtimeScope.Session().DB)
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubExportEnqueueExecutor{}})
		noCompany := auth.ContextWithIdentity(context.Background(), exportNoCompanyIdentity{})
		_, err := h.RunAsync(noCompany, &exportpb.ExportRunAsyncRequest{
			Run: &exportpb.ExportRunRequest{Model: "partner.Partner"},
		})
		if err == nil {
			t.Fatal("expected company_id required")
		}
	})
}

type exportNoCompanyIdentity struct{}

func (exportNoCompanyIdentity) GetUserID() string                   { return "test-user" }
func (exportNoCompanyIdentity) GetTokenID() string                  { return "test-token" }
func (exportNoCompanyIdentity) GetMetadata() map[string]interface{} { return map[string]interface{}{} }
func (exportNoCompanyIdentity) IsValid() bool                       { return true }

type exportNilResponseExecutor struct {
	stubJSExecutor
}

func (exportNilResponseExecutor) Execute(_ context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	if req != nil && req.Service == enqueueExportDataTransferJobService {
		return nil, nil
	}
	return &jsengine.JsResponse{}, nil
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
	if _, err := toAsyncRecordSpec(&exportpb.ExportRunRequest{
		Profile: "initdata",
		Model:   "base.Country",
	}); err == nil {
		t.Fatal("expected unsupported profile error")
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

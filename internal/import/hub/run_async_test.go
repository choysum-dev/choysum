// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
	"github.com/choysum-dev/choysum/pkg/auth"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubEnqueueExecutor struct {
	stubJSExecutor
	result map[string]any
	raw    any
	err    error
}

func (s stubEnqueueExecutor) Execute(_ context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	if req != nil && req.Service == enqueueDataTransferJobService {
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

type failMarshalResult struct{}

func (failMarshalResult) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal failed")
}

func TestRunAsync_CreatesDataTransferJobAndTaskJob(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	ctx := authCtx(t)
	h := New(Deps{
		RuntimeScope: runtimeScope,
		JSExecutor: stubEnqueueExecutor{
			result: map[string]any{
				"dataTransferJobId": "ij-1",
				"taskJobId":         "tj-1",
			},
		},
	})

	resp, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
		Run: &importpb.ImportRunRequest{
			TargetModel: "base.Country",
			SourceRef:   "src-async",
			CompanyId:   "cmp_test",
		},
	})
	if err != nil {
		t.Fatalf("RunAsync: %v", err)
	}
	if resp.GetDataTransferJobId() != "ij-1" || resp.GetTaskJobId() != "tj-1" {
		t.Fatalf("response = %+v", resp)
	}
	if resp.GetReport() == nil || resp.GetReport().GetMeta().GetTargetModel() != "base.Country" {
		t.Fatalf("report = %+v", resp.GetReport())
	}
}

func TestRunAsync_RequiresExecutor(t *testing.T) {
	ctx := authCtx(t)
	h := New(Deps{RuntimeScope: newHubTestScope(t)})
	_, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
		Run: &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src"},
	})
	if err == nil || status.Code(err) != codes.Unavailable {
		t.Fatalf("RunAsync err = %v", err)
	}
}

func TestRunAsync_ErrorPaths(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	ctx := authCtx(t)

	t.Run("unauthenticated", func(t *testing.T) {
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubEnqueueExecutor{}})
		_, err := h.RunAsync(context.Background(), &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src"},
		})
		if err == nil || status.Code(err) != codes.Unauthenticated {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("nil scope", func(t *testing.T) {
		h := New(Deps{JSExecutor: stubEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src"},
		})
		if err == nil || status.Code(err) != codes.Unavailable {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("nil request", func(t *testing.T) {
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, nil)
		if err == nil || status.Code(err) != codes.InvalidArgument {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("nil run", func(t *testing.T) {
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{})
		if err == nil || status.Code(err) != codes.InvalidArgument {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("missing model", func(t *testing.T) {
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{SourceRef: "src"},
		})
		if err == nil || status.Code(err) != codes.InvalidArgument {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("unsupported policy", func(t *testing.T) {
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{
				TargetModel: "base.Country",
				SourceRef:   "src",
				Policy:      importpb.ImportPolicy(99),
			},
		})
		if err == nil || status.Code(err) != codes.InvalidArgument {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("uses active company when missing", func(t *testing.T) {
		h := New(Deps{
			RuntimeScope: runtimeScope,
			JSExecutor: stubEnqueueExecutor{result: map[string]any{
				"dataTransferJobId": "ij-active",
				"taskJobId":         "tj-active",
			}},
		})
		resp, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{
				TargetModel: "base.Country",
				SourceRef:   "src-active",
				Policy:      importpb.ImportPolicy_IMPORT_POLICY_BEST_EFFORT,
			},
		})
		if err != nil {
			t.Fatalf("RunAsync: %v", err)
		}
		if resp.GetDataTransferJobId() != "ij-active" {
			t.Fatalf("resp = %+v", resp)
		}
	})

	t.Run("access denied", func(t *testing.T) {
		seedPartnerModelMeta(t, runtimeScope.Session().DB)
		h := New(Deps{
			RuntimeScope: runtimeScope,
			JSExecutor:   stubEnqueueExecutor{},
		})
		noCompany := auth.ContextWithIdentity(context.Background(), noCompanyIdentity{})
		_, err := h.RunAsync(noCompany, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{
				TargetModel: "partner.Partner",
				SourceRef:   "src",
			},
		})
		if err == nil {
			t.Fatal("expected company_id required")
		}
	})

	t.Run("invalid spec", func(t *testing.T) {
		orig := validateImportSpec
		t.Cleanup(func() { validateImportSpec = orig })
		validateImportSpec = func(importpkg.Spec) error {
			return errors.New("bad spec")
		}
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src"},
		})
		if err == nil || status.Code(err) != codes.InvalidArgument {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("enqueue execute error", func(t *testing.T) {
		h := New(Deps{
			RuntimeScope: runtimeScope,
			JSExecutor:   stubEnqueueExecutor{err: errors.New("exec failed")},
		})
		_, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("enqueue empty result", func(t *testing.T) {
		h := New(Deps{
			RuntimeScope: runtimeScope,
			JSExecutor:   stubEnqueueExecutor{},
		})
		_, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("enqueue marshal failure", func(t *testing.T) {
		h := New(Deps{
			RuntimeScope: runtimeScope,
			JSExecutor:   stubEnqueueExecutor{raw: failMarshalResult{}},
		})
		_, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("enqueue decode failure", func(t *testing.T) {
		h := New(Deps{
			RuntimeScope: runtimeScope,
			JSExecutor:   stubEnqueueExecutor{raw: 42},
		})
		_, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("enqueue incomplete ids", func(t *testing.T) {
		h := New(Deps{
			RuntimeScope: runtimeScope,
			JSExecutor: stubEnqueueExecutor{result: map[string]any{
				"dataTransferJobId": "ij-only",
			}},
		})
		_, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("enqueue nil response", func(t *testing.T) {
		h := New(Deps{
			RuntimeScope: runtimeScope,
			JSExecutor:   nilResponseExecutor{},
		})
		_, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("job record from spec error", func(t *testing.T) {
		orig := dataTransferJobRecordFromSpec
		t.Cleanup(func() { dataTransferJobRecordFromSpec = orig })
		dataTransferJobRecordFromSpec = func(importpkg.Spec) (importpkg.DataTransferJobRecord, error) {
			return importpkg.DataTransferJobRecord{}, errors.New("snapshot failed")
		}
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("spec snapshot decode error", func(t *testing.T) {
		orig := jsonUnmarshal
		t.Cleanup(func() { jsonUnmarshal = orig })
		jsonUnmarshal = func([]byte, any) error {
			return errors.New("decode failed")
		}
		h := New(Deps{RuntimeScope: runtimeScope, JSExecutor: stubEnqueueExecutor{}})
		_, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{TargetModel: "base.Country", SourceRef: "src"},
		})
		if err == nil || status.Code(err) != codes.Internal {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("stop_keep policy", func(t *testing.T) {
		h := New(Deps{
			RuntimeScope: runtimeScope,
			JSExecutor: stubEnqueueExecutor{result: map[string]any{
				"dataTransferJobId": "ij-sk",
				"taskJobId":         "tj-sk",
			}},
		})
		resp, err := h.RunAsync(ctx, &importpb.ImportRunAsyncRequest{
			Run: &importpb.ImportRunRequest{
				TargetModel:   "base.Country",
				SourceRef:     "src-sk",
				CompanyId:     "cmp_test",
				Policy:        importpb.ImportPolicy_IMPORT_POLICY_STOP_KEEP,
				ColumnMapping: map[string]string{"Name": "Name"},
			},
		})
		if err != nil {
			t.Fatalf("RunAsync: %v", err)
		}
		if resp.GetTaskJobId() != "tj-sk" {
			t.Fatalf("resp = %+v", resp)
		}
	})
}

type noCompanyIdentity struct{}

func (noCompanyIdentity) GetUserID() string                   { return "test-user" }
func (noCompanyIdentity) GetTokenID() string                  { return "test-token" }
func (noCompanyIdentity) GetMetadata() map[string]interface{} { return map[string]interface{}{} }
func (noCompanyIdentity) IsValid() bool                       { return true }

type nilResponseExecutor struct {
	stubJSExecutor
}

func (nilResponseExecutor) Execute(_ context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	if req != nil && req.Service == enqueueDataTransferJobService {
		return nil, nil
	}
	return &jsengine.JsResponse{}, nil
}

func TestToAsyncRecordSpecAndPolicyHelpers(t *testing.T) {
	if _, err := toAsyncRecordSpec(nil); err == nil {
		t.Fatal("expected nil request error")
	}
	spec, err := toAsyncRecordSpec(&importpb.ImportRunRequest{
		TargetModel: "base.Country",
		SourceRef:   "src",
		Policy:      importpb.ImportPolicy_IMPORT_POLICY_UNSPECIFIED,
	})
	if err != nil || spec.Policy != importpkg.PolicyAtomic || !spec.Async {
		t.Fatalf("spec=%+v err=%v", spec, err)
	}
	if _, err := asyncPolicyFromProto(importpb.ImportPolicy(99)); err == nil {
		t.Fatal("expected unsupported policy")
	}
	if got := schedulerUserID(context.Background()); got != "" {
		t.Fatalf("schedulerUserID = %q", got)
	}
	if got := schedulerUserID(auth.ContextWithIdentity(context.Background(), stubIdentity{})); got == "" {
		t.Fatal("expected user id")
	}
}

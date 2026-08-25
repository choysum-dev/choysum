// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubEnqueueExecutor struct {
	stubJSExecutor
	result map[string]any
}

func (s stubEnqueueExecutor) Execute(_ context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	if req != nil && req.Service == enqueueImportJobService {
		return &jsengine.JsResponse{Result: s.result}, nil
	}
	return s.stubJSExecutor.Execute(context.Background(), req)
}

func TestRunAsync_CreatesImportJobAndTaskJob(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	ctx := authCtx(t)
	h := New(Deps{
		RuntimeScope: runtimeScope,
		JSExecutor: stubEnqueueExecutor{
			result: map[string]any{
				"importJobId": "ij-1",
				"taskJobId":   "tj-1",
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
	if resp.GetImportJobId() != "ij-1" || resp.GetTaskJobId() != "tj-1" {
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

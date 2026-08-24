// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/adapter/csv"
	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
	"github.com/choysum-dev/choysum/pkg/auth"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubIdentity struct{}

func (stubIdentity) GetUserID() string  { return "test-user" }
func (stubIdentity) GetTokenID() string { return "test-token" }
func (stubIdentity) GetMetadata() map[string]interface{} {
	return map[string]interface{}{"activeCompanyId": "cmp_test"}
}
func (stubIdentity) IsValid() bool { return true }

func TestParseHeaders_BOM(t *testing.T) {
	csvBytes := []byte("\xef\xbb\xbfName,Code,IsActive\nAcme,ACME001,true\n")
	h := New(Deps{
		SourceReader: StubSourceReader{"src-bom": csvBytes},
	})

	ctx := auth.ContextWithIdentity(context.Background(), stubIdentity{})
	resp, err := h.ParseHeaders(ctx, &importpb.ParseHeadersRequest{SourceRef: "src-bom"})
	if err != nil {
		t.Fatalf("ParseHeaders() error = %v", err)
	}
	if len(resp.GetHeaders()) != 3 || resp.GetHeaders()[0] != "Name" {
		t.Fatalf("headers = %#v", resp.GetHeaders())
	}
	if resp.GetDelimiter() != "," || resp.GetHeaderRow() != 1 {
		t.Fatalf("delimiter/header_row = %q / %d", resp.GetDelimiter(), resp.GetHeaderRow())
	}

	table, err := csv.ReadTable(csvBytes)
	if err != nil {
		t.Fatalf("ReadTable() error = %v", err)
	}
	if table.Headers[0] != "Name" {
		t.Fatalf("BOM not stripped: %#v", table.Headers)
	}
}

func TestRun_RejectsNonAtomicPolicy(t *testing.T) {
	_, err := toRecordSpec(&importpb.ImportRunRequest{
		TargetModel: "partner.Partner",
		SourceRef:   "src-1",
		Policy:      importpb.ImportPolicy_IMPORT_POLICY_BEST_EFFORT,
		CompanyId:   "cmp_test",
	}, false)
	if err == nil {
		t.Fatal("toRecordSpec() expected error for non-atomic policy")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("toRecordSpec() code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestToRecordSpec_PreviewDryRun(t *testing.T) {
	spec, err := toRecordSpec(&importpb.ImportRunRequest{
		TargetModel: "partner.Partner",
		SourceRef:   "src-1",
		CompanyId:   "cmp_test",
	}, true)
	if err != nil {
		t.Fatalf("toRecordSpec() error = %v", err)
	}
	if !spec.DryRun {
		t.Fatal("dry_run = false, want true")
	}
	if spec.Policy != importpkg.PolicyAtomic {
		t.Fatalf("policy = %q, want atomic", spec.Policy)
	}
	if spec.Profile != importpkg.ProfileRecord || spec.Caller != importpkg.CallerUser {
		t.Fatalf("profile/caller = %q / %q", spec.Profile, spec.Caller)
	}
	if spec.Source.DocumentRef != "src-1" || spec.Source.Format != "csv" {
		t.Fatalf("source = %#v", spec.Source)
	}
	if spec.Options.CompanyID != "cmp_test" {
		t.Fatalf("company_id = %q", spec.Options.CompanyID)
	}
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package proto

import (
	"testing"

	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestReportProtoRoundTrip(t *testing.T) {
	original := importpkg.Report{
		Profile: importpkg.ProfileRecord,
		Policy:  importpkg.PolicyAtomic,
		DryRun:  true,
		Stats: importpkg.Stats{
			Total:   3,
			Ok:      2,
			Error:   1,
			Skip:    0,
			Warning: 0,
		},
		Messages: []importpkg.Message{
			{
				Type:  importpkg.MessageError,
				Row:   4,
				Field: "Code",
				Code:  "duplicate_key",
				Text:  "duplicate",
			},
		},
		Meta: &importpkg.ReportMeta{
			SourceRef:   "src-1",
			TargetModel: "partner.Partner",
		},
	}

	pb := ReportToProto(original)
	got := ReportFromProto(pb)

	if got.Profile != original.Profile {
		t.Fatalf("profile = %q, want %q", got.Profile, original.Profile)
	}
	if got.Policy != original.Policy {
		t.Fatalf("policy = %q, want %q", got.Policy, original.Policy)
	}
	if got.DryRun != original.DryRun {
		t.Fatalf("dry_run = %v, want %v", got.DryRun, original.DryRun)
	}
	if got.Stats != original.Stats {
		t.Fatalf("stats = %#v, want %#v", got.Stats, original.Stats)
	}
	if len(got.Messages) != 1 || got.Messages[0].Code != "duplicate_key" {
		t.Fatalf("messages = %#v", got.Messages)
	}
	if got.Meta == nil || got.Meta.TargetModel != "partner.Partner" {
		t.Fatalf("meta = %#v", got.Meta)
	}
}

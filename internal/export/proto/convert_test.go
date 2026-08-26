// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package proto

import (
	"testing"

	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestExportReportProtoRoundTrip(t *testing.T) {
	original := importpkg.Report{
		Profile: importpkg.ProfileRecord,
		Policy:  importpkg.PolicyUnspecified,
		DryRun:  false,
		Stats: importpkg.Stats{
			Total: 3,
			Ok:    3,
		},
		Messages: []importpkg.Message{
			{
				Type: importpkg.MessageWarning,
				Row:  2,
				Text: "truncated",
			},
		},
		ArtifactRef: "content-1",
		Meta: &importpkg.ReportMeta{
			TargetModel: "partner.Partner",
		},
	}

	pb := ReportToProto(original)
	got := ReportFromProto(pb)

	if got.Profile != original.Profile {
		t.Fatalf("profile = %q, want %q", got.Profile, original.Profile)
	}
	if got.Stats != original.Stats {
		t.Fatalf("stats = %#v, want %#v", got.Stats, original.Stats)
	}
	if len(got.Messages) != 1 || got.Messages[0].Text != "truncated" {
		t.Fatalf("messages = %#v", got.Messages)
	}
	if got.ArtifactRef != "content-1" {
		t.Fatalf("artifact_ref = %q", got.ArtifactRef)
	}
	if got.Meta == nil || got.Meta.TargetModel != "partner.Partner" {
		t.Fatalf("meta = %#v", got.Meta)
	}
}

func TestExportReportFromProtoNil(t *testing.T) {
	if got := ReportFromProto(nil); got.Profile != "" || len(got.Messages) != 0 {
		t.Fatalf("nil proto = %#v", got)
	}
}

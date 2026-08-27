// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package proto

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
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

func TestExportMessageTypeConversionBranches(t *testing.T) {
	cases := []struct {
		domain importpkg.MessageType
		pb     exportpb.ExportMessageType
	}{
		{importpkg.MessageError, exportpb.ExportMessageType_EXPORT_MESSAGE_TYPE_ERROR},
		{importpkg.MessageWarning, exportpb.ExportMessageType_EXPORT_MESSAGE_TYPE_WARNING},
		{importpkg.MessageSkip, exportpb.ExportMessageType_EXPORT_MESSAGE_TYPE_SKIP},
		{importpkg.MessageType(""), exportpb.ExportMessageType_EXPORT_MESSAGE_TYPE_UNSPECIFIED},
	}
	for _, tc := range cases {
		if got := messageTypeToProto(tc.domain); got != tc.pb {
			t.Fatalf("messageTypeToProto(%q) = %v", tc.domain, got)
		}
		if got := messageTypeFromProto(tc.pb); got != tc.domain {
			t.Fatalf("messageTypeFromProto(%v) = %q", tc.pb, got)
		}
	}
	if got := messageTypeFromProto(exportpb.ExportMessageType_EXPORT_MESSAGE_TYPE_UNSPECIFIED); got != importpkg.MessageType("") {
		t.Fatalf("unspecified = %q", got)
	}
}

func TestExportMessagesConversionNilAndEmpty(t *testing.T) {
	if messagesToProto(nil) != nil {
		t.Fatal("nil messages to proto")
	}
	if messagesFromProto(nil) != nil {
		t.Fatal("nil messages from proto")
	}
	if len(messagesFromProto([]*exportpb.ExportMessage{nil, {Type: exportpb.ExportMessageType_EXPORT_MESSAGE_TYPE_ERROR, Text: "x"}})) != 1 {
		t.Fatal("skip nil message")
	}
	if statsFromProto(nil) != (importpkg.Stats{}) {
		t.Fatal("nil stats")
	}
}

func TestExportReportToProtoWithoutMeta(t *testing.T) {
	pb := ReportToProto(importpkg.Report{
		Profile: importpkg.ProfileRecord,
		Stats:   importpkg.Stats{Total: 1},
	})
	if pb.GetMeta() != nil {
		t.Fatal("expected nil meta")
	}
}

func TestExportReportToProtoWithMeta(t *testing.T) {
	pb := ReportToProto(importpkg.Report{
		Profile: importpkg.ProfileRecord,
		Meta:    &importpkg.ReportMeta{Lang: "en", TargetModel: "base.Country"},
	})
	if pb.GetMeta().GetLang() != "en" || pb.GetMeta().GetTargetModel() != "base.Country" {
		t.Fatalf("meta = %#v", pb.GetMeta())
	}
}

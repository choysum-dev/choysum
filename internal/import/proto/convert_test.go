// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package proto

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
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

func TestReportFromProtoNil(t *testing.T) {
	if got := ReportFromProto(nil); got.Profile != "" || len(got.Messages) != 0 {
		t.Fatalf("nil proto = %#v", got)
	}
}

func TestPolicyConversionBranches(t *testing.T) {
	cases := []struct {
		policy importpkg.Policy
		pb     importpb.ImportPolicy
	}{
		{importpkg.PolicyAtomic, importpb.ImportPolicy_IMPORT_POLICY_ATOMIC},
		{importpkg.PolicyStopKeep, importpb.ImportPolicy_IMPORT_POLICY_STOP_KEEP},
		{importpkg.PolicyBestEffort, importpb.ImportPolicy_IMPORT_POLICY_BEST_EFFORT},
		{importpkg.PolicyUnspecified, importpb.ImportPolicy_IMPORT_POLICY_UNSPECIFIED},
		{importpkg.Policy("unknown"), importpb.ImportPolicy_IMPORT_POLICY_UNSPECIFIED},
	}
	for _, tc := range cases {
		if got := policyToProto(tc.policy); got != tc.pb {
			t.Fatalf("policyToProto(%q) = %v, want %v", tc.policy, got, tc.pb)
		}
		if got := policyFromProto(tc.pb); got != tc.policy && tc.policy != importpkg.Policy("unknown") {
			t.Fatalf("policyFromProto(%v) = %q, want %q", tc.pb, got, tc.policy)
		}
	}
	if got := policyFromProto(importpb.ImportPolicy_IMPORT_POLICY_UNSPECIFIED); got != importpkg.PolicyUnspecified {
		t.Fatalf("unspecified = %q", got)
	}
}

func TestMessageTypeConversionBranches(t *testing.T) {
	cases := []struct {
		domain importpkg.MessageType
		pb     importpb.ImportMessageType
	}{
		{importpkg.MessageError, importpb.ImportMessageType_IMPORT_MESSAGE_TYPE_ERROR},
		{importpkg.MessageWarning, importpb.ImportMessageType_IMPORT_MESSAGE_TYPE_WARNING},
		{importpkg.MessageSkip, importpb.ImportMessageType_IMPORT_MESSAGE_TYPE_SKIP},
		{importpkg.MessageType(""), importpb.ImportMessageType_IMPORT_MESSAGE_TYPE_UNSPECIFIED},
	}
	for _, tc := range cases {
		if got := messageTypeToProto(tc.domain); got != tc.pb {
			t.Fatalf("messageTypeToProto(%q) = %v", tc.domain, got)
		}
		if got := messageTypeFromProto(tc.pb); got != tc.domain {
			t.Fatalf("messageTypeFromProto(%v) = %q", tc.pb, got)
		}
	}
	if got := messageTypeFromProto(importpb.ImportMessageType_IMPORT_MESSAGE_TYPE_UNSPECIFIED); got != importpkg.MessageType("") {
		t.Fatalf("unspecified message type = %q", got)
	}
}

func TestMessagesConversionNilAndEmpty(t *testing.T) {
	if messagesToProto(nil) != nil {
		t.Fatal("nil messages to proto")
	}
	if messagesFromProto(nil) != nil {
		t.Fatal("nil messages from proto")
	}
	if len(messagesFromProto([]*importpb.ImportMessage{nil, {Type: importpb.ImportMessageType_IMPORT_MESSAGE_TYPE_ERROR, Text: "x"}})) != 1 {
		t.Fatal("skip nil message")
	}
	if statsFromProto(nil) != (importpkg.Stats{}) {
		t.Fatal("nil stats")
	}
}

func TestReportToProtoWithoutMeta(t *testing.T) {
	pb := ReportToProto(importpkg.Report{
		Profile: importpkg.ProfileRecord,
		Policy:  importpkg.PolicyBestEffort,
		Stats:   importpkg.Stats{Total: 1},
	})
	if pb.GetMeta() != nil {
		t.Fatal("expected nil meta")
	}
	if pb.GetPolicy() != importpb.ImportPolicy_IMPORT_POLICY_BEST_EFFORT {
		t.Fatalf("policy = %v", pb.GetPolicy())
	}
}

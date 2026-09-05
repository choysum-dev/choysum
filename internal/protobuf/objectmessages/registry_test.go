// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package objectmessages

import (
	"strings"
	"testing"
)

func TestLookupAndCollectUsed(t *testing.T) {
	fr, ok := Lookup("FieldRuleSpec")
	if !ok || fr.ProtoName != "FieldRuleSpec" {
		t.Fatalf("Lookup FieldRuleSpec: %+v ok=%v", fr, ok)
	}
	ce, ok := Lookup("ConditionEnvelope")
	if !ok || ce.ProtoName != "ConditionEnvelope" {
		t.Fatalf("Lookup ConditionEnvelope: %+v ok=%v", ce, ok)
	}
	if _, ok := Lookup("Unknown"); ok {
		t.Fatal("expected Unknown miss")
	}
	if _, ok := Lookup("  "); ok {
		t.Fatal("expected blank miss")
	}

	if name, ok := ProtoNameForTS("FieldRuleSpec"); !ok || name != "FieldRuleSpec" {
		t.Fatalf("ProtoNameForTS FieldRuleSpec: %q %v", name, ok)
	}
	if _, ok := ProtoNameForTS("Nope"); ok {
		t.Fatal("expected ProtoNameForTS miss")
	}

	if !IsRegisteredProtoName("FieldRuleSpec") || !IsRegisteredProtoName("ConditionEnvelope") {
		t.Fatal("expected registered proto names")
	}
	if IsRegisteredProtoName("google.protobuf.Value") || IsRegisteredProtoName("  ") {
		t.Fatal("expected unregistered proto names")
	}

	used := CollectUsed("string", "FieldRuleSpec", "repeated ConditionEnvelope", "google.protobuf.Value", "", "  ")
	if len(used) != 2 {
		t.Fatalf("CollectUsed len=%d want 2: %+v", len(used), used)
	}
	if used[0].ProtoName != "ConditionEnvelope" || used[1].ProtoName != "FieldRuleSpec" {
		t.Fatalf("CollectUsed order: %+v", used)
	}
	if got := CollectUsed(); len(got) != 0 {
		t.Fatalf("CollectUsed() empty = %+v", got)
	}
	if got := CollectUsed("Nope", "also-nope"); len(got) != 0 {
		t.Fatalf("CollectUsed unknown = %+v", got)
	}

	byProto, ok := lookupByProtoName("ConditionEnvelope")
	if !ok || byProto.TSName != "ConditionEnvelope" {
		t.Fatalf("lookupByProtoName: %+v ok=%v", byProto, ok)
	}
	if _, ok := lookupByProtoName("missing"); ok {
		t.Fatal("expected lookupByProtoName miss")
	}

	// Cover CollectUsed fallback when ProtoName differs from TSName.
	origRegistry := registry
	origByTS := byTSName
	t.Cleanup(func() {
		registry = origRegistry
		byTSName = origByTS
	})
	registry = append(append([]Def{}, origRegistry...), Def{
		TSName:    "InternalAlias",
		ProtoName: "ExternalMsg",
		Body:      "  string x = 1;",
	})
	byTSName = make(map[string]Def, len(registry))
	for _, def := range registry {
		byTSName[def.TSName] = def
	}
	fallback := CollectUsed("ExternalMsg")
	if len(fallback) != 1 || fallback[0].ProtoName != "ExternalMsg" {
		t.Fatalf("CollectUsed proto-name fallback: %+v", fallback)
	}

	all := All()
	if len(all) != 3 || all[0].ProtoName != "ConditionEnvelope" || all[1].ProtoName != "ExternalMsg" || all[2].ProtoName != "FieldRuleSpec" {
		t.Fatalf("All: %+v", all)
	}

	src := MessageSource(fr)
	for _, part := range []string{"message FieldRuleSpec {", "denyReadFields = 1", "hitRuleIds = 4"} {
		if !strings.Contains(src, part) {
			t.Fatalf("MessageSource missing %q:\n%s", part, src)
		}
	}
}

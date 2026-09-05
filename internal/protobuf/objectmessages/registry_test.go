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

	used := CollectUsed("string", "FieldRuleSpec", "repeated ConditionEnvelope", "google.protobuf.Value")
	if len(used) != 2 {
		t.Fatalf("CollectUsed len=%d want 2: %+v", len(used), used)
	}
	if used[0].ProtoName != "ConditionEnvelope" || used[1].ProtoName != "FieldRuleSpec" {
		t.Fatalf("CollectUsed order: %+v", used)
	}

	src := MessageSource(fr)
	for _, part := range []string{"message FieldRuleSpec {", "denyReadFields = 1", "hitRuleIds = 4"} {
		if !strings.Contains(src, part) {
			t.Fatalf("MessageSource missing %q:\n%s", part, src)
		}
	}
}

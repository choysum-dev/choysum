// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/module/evolution/schema"
)

func TestScopeSchemaIntents_RestoresOuterBag(t *testing.T) {
	ctx := newOpContext()
	outer := ctx.schemaIntents
	outer.Add(schema.Intent{Kind: schema.IntentDropColumn, Table: "t", Name: "c"})

	m := &moduleUpgrader{ctx: ctx}
	restore := m.scopeSchemaIntents()
	inner := m.schemaIntents()
	if inner == nil || inner == outer {
		t.Fatal("expected a fresh bag for this upgrade")
	}
	inner.Add(schema.Intent{Kind: schema.IntentDropColumn, Table: "t", Name: "inner"})
	restore()

	if m.schemaIntents() != outer {
		t.Fatal("expected outer bag restored")
	}
	if got := outer.List(); len(got) != 1 || got[0].Name != "c" {
		t.Fatalf("outer intents corrupted: %#v", got)
	}

	nilUpgrader := (*moduleUpgrader)(nil)
	restoreNil := nilUpgrader.scopeSchemaIntents()
	restoreNil()

	noCtx := &moduleUpgrader{}
	restoreNoCtx := noCtx.scopeSchemaIntents()
	restoreNoCtx()
}

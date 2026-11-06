// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"testing"
)

func TestResolveWithDemo_DefaultFalseThenSetTrue(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}

	if o.ResolveWithDemo(context.Background()) {
		t.Fatal("expected default false")
	}
	if err := o.SetWithDemo(context.Background(), true); err != nil {
		t.Fatalf("SetWithDemo: %v", err)
	}
	if !o.ResolveWithDemo(context.Background()) {
		t.Fatal("expected true after SetWithDemo")
	}
	// Idempotent upsert.
	if err := o.SetWithDemo(context.Background(), true); err != nil {
		t.Fatalf("SetWithDemo again: %v", err)
	}
	if !o.ResolveWithDemo(context.Background()) {
		t.Fatal("expected still true")
	}
}

func TestSetWithDemo_RejectsFalse(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	if err := o.SetWithDemo(context.Background(), false); err == nil {
		t.Fatal("expected error for false")
	}
}

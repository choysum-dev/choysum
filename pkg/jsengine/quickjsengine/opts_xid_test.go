// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"testing"

	"github.com/rs/xid"
)

func TestWithXidExposesGenerator(t *testing.T) {
	engine := newTestQuickjsEngine(t, WithXid())

	idA := evalString(t, engine, `$choysum.xid.New()`)
	idB := evalString(t, engine, `$choysum.xid.New()`)
	if idA == idB {
		t.Fatalf("expected unique xid values, got %q", idA)
	}
	if _, err := xid.FromString(idA); err != nil {
		t.Fatalf("invalid xid %q: %v", idA, err)
	}
	if _, err := xid.FromString(idB); err != nil {
		t.Fatalf("invalid xid %q: %v", idB, err)
	}
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package model

import "testing"

func TestEntitiesAndLockLeaseTableName(t *testing.T) {
	entities := Entities()
	if len(entities) != 1 {
		t.Fatalf("Entities() len = %d, want 1", len(entities))
	}
	if _, ok := entities[0].(*LockLease); !ok {
		t.Fatalf("Entities()[0] type = %T, want *LockLease", entities[0])
	}
	if got := (&LockLease{}).TableName(); got != "meta_lock_lease" {
		t.Fatalf("LockLease.TableName() = %q, want meta_lock_lease", got)
	}
}

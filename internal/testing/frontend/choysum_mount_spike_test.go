// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import "testing"

func TestPlannedVTUSubsetAPIsAlias(t *testing.T) {
	if len(PlannedVTUSubsetAPIs) == 0 || len(PlannedVTUSubsetAPIs) != len(FrozenVTUSubsetAPIs) {
		t.Fatalf("PlannedVTUSubsetAPIs alias broken: %v vs %v", PlannedVTUSubsetAPIs, FrozenVTUSubsetAPIs)
	}
}

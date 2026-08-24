// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan_test

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/import/plan"
	planstub "github.com/choysum-dev/choysum/internal/import/plan/stub"
)

func TestPlan_Len(t *testing.T) {
	p := plan.Plan{Units: []plan.Unit{planstub.Unit{Index: 1}, planstub.Unit{Index: 2}}}
	if p.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", p.Len())
	}
}

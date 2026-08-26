// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	"errors"
	"testing"

	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

func TestPlanFromSpec_validationError(t *testing.T) {
	_, err := PlanFromSpec(exportpkg.Spec{})
	if !errors.Is(err, exportpkg.ErrProfileNotApproved) {
		t.Fatalf("PlanFromSpec() error = %v, want ErrProfileNotApproved", err)
	}
}

func TestPlanFromSpec_recordDefaults(t *testing.T) {
	p, err := PlanFromSpec(exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Mode:    exportpkg.ModeTemplate,
		Fields:  []string{"Name"},
	})
	if err != nil {
		t.Fatalf("PlanFromSpec: %v", err)
	}
	if p.Format != "csv" || p.Mode != exportpkg.ModeTemplate || p.Model != "base.Country" {
		t.Fatalf("plan = %+v", p)
	}
	if len(p.Fields) != 1 || p.Fields[0] != "Name" {
		t.Fatalf("fields = %+v", p.Fields)
	}
}

func TestPlanFromSpec_terminologyDefaults(t *testing.T) {
	p, err := PlanFromSpec(exportpkg.Spec{
		Profile: exportpkg.ProfileTerminology,
		Caller:  exportpkg.CallerUser,
		Module:  "base",
		Lang:    "zh_CN",
	})
	if err != nil {
		t.Fatalf("PlanFromSpec: %v", err)
	}
	if p.Format != "po" || p.Mode != exportpkg.ModeUnspecified || p.Module != "base" || p.Lang != "zh_CN" {
		t.Fatalf("plan = %+v", p)
	}
}

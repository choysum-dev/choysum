// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	"strings"

	"github.com/choysum-dev/choysum/internal/export/plan"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

// PlanFromSpec builds the minimal Plan used by Runner/Readers.
func PlanFromSpec(spec exportpkg.Spec) (plan.Plan, error) {
	if err := exportpkg.ValidateSpec(spec); err != nil {
		return plan.Plan{}, err
	}
	if err := validatePlanInputs(spec); err != nil {
		return plan.Plan{}, err
	}

	p := plan.Plan{
		Profile:           spec.Profile,
		Caller:            spec.Caller,
		Mode:              exportpkg.EffectiveMode(spec.Mode),
		Format:            strings.TrimSpace(spec.Format),
		Model:             strings.TrimSpace(spec.Model),
		Fields:            append([]string(nil), spec.Fields...),
		Domain:            strings.TrimSpace(spec.Domain),
		Ids:               append([]string(nil), spec.Ids...),
		Limit:             spec.Limit,
		Offset:            spec.Offset,
		Application:       strings.TrimSpace(spec.Application),
		Module:            strings.TrimSpace(spec.Module),
		Lang:              strings.TrimSpace(spec.Lang),
		CompanyID:         strings.TrimSpace(spec.Options.CompanyID),
		StubUnitCount:     spec.Options.StubUnitCount,
		StubFailUnitIndex: spec.Options.StubFailUnitIndex,
	}
	if p.Format == "" {
		switch spec.Profile {
		case exportpkg.ProfileRecord:
			p.Format = "csv"
		case exportpkg.ProfileTerminology:
			p.Format = "po"
		}
	}
	if spec.Profile == exportpkg.ProfileTerminology {
		p.Mode = exportpkg.ModeUnspecified
	}
	return p, nil
}

func validatePlanInputs(spec exportpkg.Spec) error {
	// Caller×Profile and Mode-only-for-record are enforced in ValidateSpec;
	// keep this hook for runner-local plan constraints.
	_ = spec
	return nil
}

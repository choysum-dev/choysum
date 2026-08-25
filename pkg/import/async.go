// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg

import (
	"encoding/json"
	"strings"
)

var marshalSpecSnapshot = json.Marshal

// JobRecordFromSpec builds the lean async domain DTO from a validated record Spec.
func JobRecordFromSpec(spec Spec) (JobRecord, error) {
	companyID := strings.TrimSpace(spec.Options.CompanyID)
	snapshot, err := SpecSnapshotJSON(spec)
	if err != nil {
		return JobRecord{}, err
	}
	return JobRecord{
		Profile:          spec.Profile,
		Policy:           EffectivePolicy(spec),
		DryRun:           spec.DryRun,
		TargetModel:      strings.TrimSpace(spec.Model),
		SourceRef:        strings.TrimSpace(spec.Source.DocumentRef),
		CompanyID:        companyID,
		SpecSnapshotJSON: snapshot,
		Direction:        "import",
	}, nil
}

// SpecSnapshotJSON freezes the import Spec for async worker replay.
func SpecSnapshotJSON(spec Spec) ([]byte, error) {
	return marshalSpecSnapshot(spec)
}

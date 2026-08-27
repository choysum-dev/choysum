// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportpkg

import (
	"encoding/json"
	"strings"
)

var marshalSpecSnapshot = json.Marshal

// ExportJobSourceRef is the lean domain SourceRef for async record exports.
func ExportJobSourceRef(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "export"
	}
	return "export:" + model
}

// DataTransferJobRecordFromSpec builds the lean async domain DTO from a validated record Spec.
func DataTransferJobRecordFromSpec(spec Spec) (DataTransferJobRecord, error) {
	companyID := strings.TrimSpace(spec.Options.CompanyID)
	snapshot, err := SpecSnapshotJSON(spec)
	if err != nil {
		return DataTransferJobRecord{}, err
	}
	return DataTransferJobRecord{
		Profile:          spec.Profile,
		TargetModel:      strings.TrimSpace(spec.Model),
		SourceRef:        ExportJobSourceRef(spec.Model),
		CompanyID:        companyID,
		SpecSnapshotJSON: snapshot,
		Direction:        "export",
	}, nil
}

// SpecSnapshotJSON freezes the export Spec for async worker replay.
func SpecSnapshotJSON(spec Spec) ([]byte, error) {
	return marshalSpecSnapshot(spec)
}

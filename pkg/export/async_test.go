// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportpkg_test

import (
	"encoding/json"
	"testing"

	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

func TestDataTransferJobRecordFromSpec(t *testing.T) {
	spec := exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Mode:    exportpkg.ModeData,
		Format:  "csv",
		Model:   "base.Country",
		Async:   true,
		Options: exportpkg.Options{
			CompanyID: "cmp-1",
		},
	}
	record, err := exportpkg.DataTransferJobRecordFromSpec(spec)
	if err != nil {
		t.Fatalf("DataTransferJobRecordFromSpec: %v", err)
	}
	if record.Profile != exportpkg.ProfileRecord {
		t.Fatalf("record = %+v", record)
	}
	if record.TargetModel != "base.Country" || record.SourceRef != "export:base.Country" || record.CompanyID != "cmp-1" {
		t.Fatalf("unexpected record fields: %+v", record)
	}
	if record.Direction != "export" || len(record.SpecSnapshotJSON) == 0 {
		t.Fatalf("unexpected snapshot/direction: %+v", record)
	}
	var roundTrip exportpkg.Spec
	if err := json.Unmarshal(record.SpecSnapshotJSON, &roundTrip); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if roundTrip.Model != "base.Country" || !roundTrip.Async {
		t.Fatalf("roundTrip = %+v", roundTrip)
	}
}

func TestExportJobSourceRef(t *testing.T) {
	if got := exportpkg.ExportJobSourceRef(""); got != "export" {
		t.Fatalf("ExportJobSourceRef(\"\") = %q", got)
	}
	if got := exportpkg.ExportJobSourceRef(" base.Country "); got != "export:base.Country" {
		t.Fatalf("ExportJobSourceRef = %q", got)
	}
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg_test

import (
	"encoding/json"
	"testing"

	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestJobRecordFromSpec(t *testing.T) {
	spec := importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyBestEffort,
		Model:   "base.Country",
		Source: importpkg.Source{
			Format:      "csv",
			DocumentRef: "doc-ref-1",
		},
		Options: importpkg.Options{
			CompanyID:     "cmp-1",
			ColumnMapping: map[string]string{"Name": "Name"},
		},
	}
	record := importpkg.JobRecordFromSpec(spec)
	if record.Profile != importpkg.ProfileRecord || record.Policy != importpkg.PolicyBestEffort {
		t.Fatalf("record = %+v", record)
	}
	if record.TargetModel != "base.Country" || record.SourceRef != "doc-ref-1" || record.CompanyID != "cmp-1" {
		t.Fatalf("unexpected record fields: %+v", record)
	}
	if record.Direction != "import" || len(record.SpecSnapshotJSON) == 0 {
		t.Fatalf("unexpected snapshot/direction: %+v", record)
	}
	var roundTrip importpkg.Spec
	if err := json.Unmarshal(record.SpecSnapshotJSON, &roundTrip); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if roundTrip.Model != "base.Country" {
		t.Fatalf("roundTrip model = %q", roundTrip.Model)
	}
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestDataTransferJobRecordFromSpec_SnapshotError(t *testing.T) {
	orig := marshalSpecSnapshot
	t.Cleanup(func() { marshalSpecSnapshot = orig })
	marshalSpecSnapshot = func(any) ([]byte, error) {
		return nil, errors.New("marshal boom")
	}
	_, err := DataTransferJobRecordFromSpec(Spec{Profile: ProfileRecord, Model: "base.Country"})
	if err == nil {
		t.Fatal("expected snapshot error")
	}
}

func TestSpecSnapshotJSON_RoundTrip(t *testing.T) {
	raw, err := SpecSnapshotJSON(Spec{Model: "base.Country", Profile: ProfileRecord})
	if err != nil {
		t.Fatal(err)
	}
	var got Spec
	if err := json.Unmarshal(raw, &got); err != nil || got.Model != "base.Country" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner_test

import (
	"encoding/json"
	"testing"

	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestReport_JSONGolden(t *testing.T) {
	report := importpkg.Report{
		Profile: importpkg.ProfileRecord,
		Policy:  importpkg.PolicyBestEffort,
		DryRun:  false,
		Stats: importpkg.Stats{
			Total: 100,
			Ok:    97,
			Error: 2,
			Skip:  1,
		},
		Messages: []importpkg.Message{
			{Type: importpkg.MessageError, Row: 42, Field: "email", Code: "constraint", Text: "duplicate email"},
			{Type: importpkg.MessageSkip, Row: 7, Code: "empty_required", Text: "name is required"},
		},
	}

	got, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	const want = `{"profile":"record","policy":"best_effort","dry_run":false,"stats":{"total":100,"ok":97,"error":2,"skip":1},"messages":[{"type":"error","row":42,"field":"email","code":"constraint","text":"duplicate email"},{"type":"skip","row":7,"code":"empty_required","text":"name is required"}]}`
	if string(got) != want {
		t.Fatalf("JSON = %s, want %s", got, want)
	}
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/module/evolution/scripts"
)

func TestSameNormalizedVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		from string
		to   string
		want bool
	}{
		{from: "1.0.0", to: "1.0.0", want: true},
		{from: "v1.0.0", to: "1.0.0", want: true},
		{from: "1.0.0", to: "1.0.1", want: false},
		{from: "", to: "", want: true},
		{from: "1.0.0", to: "", want: false},
		// Non-semver tags must not collapse via semver.Compare(invalid,invalid)==0.
		{from: "0.1", to: "0.2", want: false},
		{from: "2024.1", to: "2024.2", want: false},
		{from: "0.1", to: "0.1", want: true},
	}
	for _, tt := range tests {
		if got := scripts.SameNormalizedVersion(tt.from, tt.to); got != tt.want {
			t.Fatalf("SameNormalizedVersion(%q,%q)=%v want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestShouldSkipSameVersionMigrationScripts(t *testing.T) {
	t.Parallel()
	if !shouldSkipSameVersionMigrationScripts("0.1.0", "0.1.0") {
		t.Fatal("same version should skip")
	}
	if shouldSkipSameVersionMigrationScripts("0.1.0", "0.2.0") {
		t.Fatal("version bump must not skip")
	}
}

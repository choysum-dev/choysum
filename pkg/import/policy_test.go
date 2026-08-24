// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg_test

import (
	"testing"

	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestAllowedPolicies_Matrix(t *testing.T) {
	tests := []struct {
		profile importpkg.Profile
		want    []importpkg.Policy
	}{
		{importpkg.ProfileInitdata, []importpkg.Policy{importpkg.PolicyAtomic}},
		{importpkg.ProfileTerminology, []importpkg.Policy{importpkg.PolicyAtomic}},
		{importpkg.ProfileRecord, []importpkg.Policy{importpkg.PolicyAtomic, importpkg.PolicyStopKeep, importpkg.PolicyBestEffort}},
	}
	for _, tc := range tests {
		got := importpkg.AllowedPolicies(tc.profile)
		if len(got) != len(tc.want) {
			t.Fatalf("AllowedPolicies(%q) = %v, want %v", tc.profile, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("AllowedPolicies(%q) = %v, want %v", tc.profile, got, tc.want)
			}
		}
	}
}

func TestAllowsPolicy_Matrix(t *testing.T) {
	if importpkg.AllowsPolicy(importpkg.ProfileInitdata, importpkg.PolicyBestEffort) {
		t.Fatal("initdata must not allow best_effort")
	}
	if !importpkg.AllowsPolicy(importpkg.ProfileRecord, importpkg.PolicyStopKeep) {
		t.Fatal("record must allow stop_keep")
	}
}

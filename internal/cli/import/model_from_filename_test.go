// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importcli_test

import (
	"testing"

	importcli "github.com/choysum-dev/choysum/internal/cli/import"
)

func TestModelFromFilename(t *testing.T) {
	tests := []struct {
		path    string
		want    string
		wantErr bool
	}{
		{path: "./base.Country.csv", want: "base.Country"},
		{path: "imports/base_Country.csv", want: "base.Country"},
		{path: "partner-Partner.csv", want: "partner.Partner"},
		{path: "country_import_ok.csv", wantErr: true},
		{path: "base.country.csv", wantErr: true},
		{path: "readme.txt", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got, err := importcli.ModelFromFilename(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ModelFromFilename(%q) error = nil, want error", tc.path)
				}
				return
			}
			if err != nil {
				t.Fatalf("ModelFromFilename(%q): %v", tc.path, err)
			}
			if got != tc.want {
				t.Fatalf("ModelFromFilename(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportcli

import (
	"errors"
	"testing"

	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

func TestTerminologySpecFromOptions(t *testing.T) {
	spec, err := terminologySpecFromOptions(TerminologyOptions{
		Application: "auth",
		Module:      "base",
		Lang:        "zh_CN",
	})
	if err != nil {
		t.Fatalf("terminologySpecFromOptions: %v", err)
	}
	if spec.Profile != exportpkg.ProfileTerminology || spec.Caller != exportpkg.CallerCLI {
		t.Fatalf("spec = %+v", spec)
	}
	if spec.Format != "po" {
		t.Fatalf("format = %q", spec.Format)
	}
}

func TestTerminologySpecFromOptionsValidation(t *testing.T) {
	cases := []struct {
		name string
		opts TerminologyOptions
	}{
		{name: "missing application", opts: TerminologyOptions{Module: "base", Lang: "zh_CN"}},
		{name: "missing module", opts: TerminologyOptions{Application: "auth", Lang: "zh_CN"}},
		{name: "missing lang", opts: TerminologyOptions{Application: "auth", Module: "base"}},
		{name: "invalid lang", opts: TerminologyOptions{Application: "auth", Module: "base", Lang: "bad lang!"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := terminologySpecFromOptions(tc.opts); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestTerminologySpecRejectsE2ECallerMatrix(t *testing.T) {
	spec, err := terminologySpecFromOptions(TerminologyOptions{
		Application: "auth",
		Module:      "base",
		Lang:        "zh_CN",
	})
	if err != nil {
		t.Fatalf("terminologySpecFromOptions: %v", err)
	}
	spec.Caller = exportpkg.CallerE2E
	if err := exportpkg.ValidateSpec(spec); !errors.Is(err, exportpkg.ErrCallerProfileDenied) {
		t.Fatalf("ValidateSpec() err = %v, want ErrCallerProfileDenied", err)
	}
}

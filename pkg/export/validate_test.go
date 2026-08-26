// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportpkg_test

import (
	"errors"
	"testing"

	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

func TestValidateSpec_RejectsInitdataProfile(t *testing.T) {
	err := exportpkg.ValidateSpec(exportpkg.Spec{
		Profile: exportpkg.Profile("initdata"),
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	})
	if !errors.Is(err, exportpkg.ErrProfileNotApproved) {
		t.Fatalf("ValidateSpec() error = %v, want ErrProfileNotApproved", err)
	}
}

func TestValidateSpec_CallerMatrix(t *testing.T) {
	type tc struct {
		name    string
		profile exportpkg.Profile
		caller  exportpkg.Caller
		ok      bool
	}
	cases := []tc{
		{name: "record/user", profile: exportpkg.ProfileRecord, caller: exportpkg.CallerUser, ok: true},
		{name: "record/cli", profile: exportpkg.ProfileRecord, caller: exportpkg.CallerCLI, ok: true},
		{name: "record/e2e", profile: exportpkg.ProfileRecord, caller: exportpkg.CallerE2E, ok: true},
		{name: "terminology/user", profile: exportpkg.ProfileTerminology, caller: exportpkg.CallerUser, ok: true},
		{name: "terminology/cli", profile: exportpkg.ProfileTerminology, caller: exportpkg.CallerCLI, ok: true},
		{name: "terminology/e2e", profile: exportpkg.ProfileTerminology, caller: exportpkg.CallerE2E, ok: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			spec := exportpkg.Spec{Profile: c.profile, Caller: c.caller}
			switch c.profile {
			case exportpkg.ProfileRecord:
				spec.Model = "base.Country"
				spec.Format = "csv"
			case exportpkg.ProfileTerminology:
				spec.Module = "base"
				spec.Lang = "zh_CN"
				spec.Format = "po"
			}
			err := exportpkg.ValidateSpec(spec)
			if c.ok && err != nil {
				t.Fatalf("ValidateSpec() unexpected error = %v", err)
			}
			if !c.ok && !errors.Is(err, exportpkg.ErrCallerProfileDenied) {
				t.Fatalf("ValidateSpec() error = %v, want ErrCallerProfileDenied", err)
			}
		})
	}
}

func TestValidateSpec_RecordRequiresModel(t *testing.T) {
	err := exportpkg.ValidateSpec(exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Format:  "csv",
	})
	var expErr *exportpkg.Error
	if !errors.As(err, &expErr) || expErr.Code != exportpkg.CodeModelNotFound {
		t.Fatalf("error = %v, want model required", err)
	}
}

func TestValidateSpec_TerminologyRejectsModeAndFields(t *testing.T) {
	err := exportpkg.ValidateSpec(exportpkg.Spec{
		Profile: exportpkg.ProfileTerminology,
		Caller:  exportpkg.CallerUser,
		Mode:    exportpkg.ModeData,
		Module:  "base",
		Lang:    "en_US",
		Format:  "po",
	})
	var expErr *exportpkg.Error
	if !errors.As(err, &expErr) || expErr.Code != exportpkg.CodeInvalidMode {
		t.Fatalf("error = %v, want CodeInvalidMode", err)
	}

	err = exportpkg.ValidateSpec(exportpkg.Spec{
		Profile: exportpkg.ProfileTerminology,
		Caller:  exportpkg.CallerUser,
		Module:  "base",
		Lang:    "en_US",
		Format:  "po",
		Fields:  []string{"Name"},
	})
	if !errors.As(err, &expErr) || expErr.Code != exportpkg.CodeInvalidSpec {
		t.Fatalf("error = %v, want CodeInvalidSpec", err)
	}
}

func TestValidateSpec_CallerRequired(t *testing.T) {
	err := exportpkg.ValidateSpec(exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Model:   "base.Country",
		Format:  "csv",
	})
	var expErr *exportpkg.Error
	if !errors.As(err, &expErr) || expErr.Code != exportpkg.CodeCallerProfileDenied {
		t.Fatalf("error = %v, want caller required", err)
	}
}

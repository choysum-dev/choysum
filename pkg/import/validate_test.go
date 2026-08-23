// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg_test

import (
	"errors"
	"testing"

	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestValidateSpec_CallerProfileDenied_InitdataUser(t *testing.T) {
	err := importpkg.ValidateSpec(importpkg.Spec{
		Profile: importpkg.ProfileInitdata,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Module:  "base",
		Source:  importpkg.Source{Format: "json"},
	})
	if !errors.Is(err, importpkg.ErrCallerProfileDenied) {
		t.Fatalf("ValidateSpec() error = %v, want ErrCallerProfileDenied", err)
	}
}

func TestValidateSpec_PolicyDenied_StopKeepOnInitdata(t *testing.T) {
	err := importpkg.ValidateSpec(importpkg.Spec{
		Profile: importpkg.ProfileInitdata,
		Caller:  importpkg.CallerLifecycle,
		Policy:  importpkg.PolicyStopKeep,
		Module:  "base",
		Source:  importpkg.Source{Format: "json"},
	})
	if !errors.Is(err, importpkg.ErrPolicyDenied) {
		t.Fatalf("ValidateSpec() error = %v, want ErrPolicyDenied", err)
	}
}

func TestValidateSpec_DryRunRequiresAtomic(t *testing.T) {
	err := importpkg.ValidateSpec(importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyBestEffort,
		DryRun:  true,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "csv"},
	})
	if !errors.Is(err, importpkg.ErrDryRunRequiresAtomic) {
		t.Fatalf("ValidateSpec() error = %v, want ErrDryRunRequiresAtomic", err)
	}
}

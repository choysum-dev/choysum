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

func TestValidateSpec_InvalidProfile(t *testing.T) {
	err := importpkg.ValidateSpec(importpkg.Spec{
		Profile: importpkg.ProfileUnspecified,
		Caller:  importpkg.CallerUser,
		Source:  importpkg.Source{Format: "csv"},
	})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeModelNotFound {
		t.Fatalf("error = %v, want CodeModelNotFound", err)
	}
}

func TestValidateSpec_InvalidCaller(t *testing.T) {
	err := importpkg.ValidateSpec(importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUnspecified,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "csv"},
	})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeCallerProfileDenied {
		t.Fatalf("error = %v, want caller required", err)
	}
}

func TestValidateSpec_InvalidPolicy(t *testing.T) {
	err := importpkg.ValidateSpec(importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.Policy("bogus"),
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "csv"},
	})
	if !errors.Is(err, importpkg.ErrPolicyDenied) {
		t.Fatalf("error = %v, want ErrPolicyDenied", err)
	}
}

func TestValidateSpec_AsyncOnlyRecord(t *testing.T) {
	err := importpkg.ValidateSpec(importpkg.Spec{
		Profile: importpkg.ProfileInitdata,
		Caller:  importpkg.CallerLifecycle,
		Policy:  importpkg.PolicyAtomic,
		Module:  "base",
		Source:  importpkg.Source{Format: "json"},
		Async:   true,
	})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodePolicyDenied {
		t.Fatalf("error = %v, want async denied for initdata", err)
	}
}

func TestValidateSpec_RecordRequiresModel(t *testing.T) {
	err := importpkg.ValidateSpec(importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Source:  importpkg.Source{Format: "csv"},
	})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeModelNotFound {
		t.Fatalf("error = %v, want model required", err)
	}
}

func TestValidateSpec_InitdataRequiresModule(t *testing.T) {
	err := importpkg.ValidateSpec(importpkg.Spec{
		Profile: importpkg.ProfileInitdata,
		Caller:  importpkg.CallerLifecycle,
		Policy:  importpkg.PolicyAtomic,
		Source:  importpkg.Source{Format: "json"},
	})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("error = %v, want module required", err)
	}
}

func TestValidateSpec_TerminologyRequiresModule(t *testing.T) {
	err := importpkg.ValidateSpec(importpkg.Spec{
		Profile: importpkg.ProfileTerminology,
		Caller:  importpkg.CallerCLI,
		Policy:  importpkg.PolicyAtomic,
		Source:  importpkg.Source{Format: "po"},
	})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("error = %v, want module required", err)
	}
}

func TestValidateSpec_SourceFormatRequired(t *testing.T) {
	err := importpkg.ValidateSpec(importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Model:   "base.Country",
	})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("error = %v, want source format required", err)
	}
}

func TestValidateSpec_PolicyUnspecifiedDefaultsAtomic(t *testing.T) {
	err := importpkg.ValidateSpec(importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "csv"},
	})
	if err != nil {
		t.Fatalf("ValidateSpec() error = %v, want nil", err)
	}
}

func TestEffectivePolicy(t *testing.T) {
	if got := importpkg.EffectivePolicy(importpkg.Spec{}); got != importpkg.PolicyAtomic {
		t.Fatalf("EffectivePolicy(empty) = %q, want atomic", got)
	}
	if got := importpkg.EffectivePolicy(importpkg.Spec{Policy: importpkg.PolicyStopKeep}); got != importpkg.PolicyStopKeep {
		t.Fatalf("EffectivePolicy(stop_keep) = %q", got)
	}
}

func TestProfileCallerPolicyValid(t *testing.T) {
	if importpkg.Profile("bogus").Valid() {
		t.Fatal("bogus profile should be invalid")
	}
	if importpkg.Caller("bogus").Valid() {
		t.Fatal("bogus caller should be invalid")
	}
	if importpkg.Policy("bogus").Valid() {
		t.Fatal("bogus policy should be invalid")
	}
	if importpkg.AllowedPolicies(importpkg.Profile("bogus")) != nil {
		t.Fatal("unknown profile should have no allowed policies")
	}
}

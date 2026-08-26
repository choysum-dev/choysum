// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportpkg

import "strings"

// ValidateSpec checks caller×profile and profile-specific fields (EX9).
func ValidateSpec(spec Spec) error {
	if !spec.Profile.Valid() {
		return ErrProfileNotApproved
	}
	if !spec.Caller.Valid() {
		return Errorf(CodeCallerProfileDenied, "caller is required")
	}
	if !AllowsCallerProfile(spec.Profile, spec.Caller) {
		return ErrCallerProfileDenied
	}

	if spec.Async && spec.Profile != ProfileRecord {
		return Errorf(CodeAsyncNotSupported, "async is only allowed for record profile")
	}

	switch spec.Profile {
	case ProfileRecord:
		if strings.TrimSpace(spec.Model) == "" {
			return Errorf(CodeModelNotFound, "model is required for record profile")
		}
		mode := EffectiveMode(spec.Mode)
		if !mode.Valid() {
			return Errorf(CodeInvalidMode, "unsupported export mode "+string(spec.Mode))
		}
		format := strings.TrimSpace(spec.Format)
		if format == "" {
			format = "csv"
		}
		if format != "csv" {
			return Errorf(CodeInvalidFormat, "record export format must be csv")
		}
	case ProfileTerminology:
		if spec.Mode != ModeUnspecified {
			return Errorf(CodeInvalidMode, "mode is only allowed for record profile")
		}
		if len(spec.Fields) > 0 || len(spec.Ids) > 0 {
			return Errorf(CodeInvalidSpec, "fields and ids are only allowed for record profile")
		}
		if strings.TrimSpace(spec.Module) == "" {
			return Errorf(CodeInvalidSpec, "module is required for terminology profile")
		}
		if strings.TrimSpace(spec.Lang) == "" {
			return Errorf(CodeInvalidSpec, "lang is required for terminology profile")
		}
		format := strings.TrimSpace(spec.Format)
		if format == "" {
			format = "po"
		}
		if format != "po" {
			return Errorf(CodeInvalidFormat, "terminology export format must be po")
		}
	}

	return nil
}

// AllowsCallerProfile reports whether caller may run profile (EX9).
func AllowsCallerProfile(profile Profile, caller Caller) bool {
	switch profile {
	case ProfileRecord:
		return caller == CallerUser || caller == CallerCLI || caller == CallerE2E
	case ProfileTerminology:
		return caller == CallerUser || caller == CallerCLI
	default:
		return false
	}
}

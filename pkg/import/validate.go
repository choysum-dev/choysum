// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg

import "strings"

// ValidateSpec checks caller×profile×policy and required fields (§12.7, §5.2).
func ValidateSpec(spec Spec) error {
	if !spec.Profile.Valid() {
		return Errorf(CodeModelNotFound, "profile is required")
	}
	if !spec.Caller.Valid() {
		return Errorf(CodeCallerProfileDenied, "caller is required")
	}
	if !allowsCallerProfile(spec.Profile, spec.Caller) {
		return ErrCallerProfileDenied
	}

	policy := spec.Policy
	if policy == PolicyUnspecified {
		policy = PolicyAtomic
	}
	if !policy.Valid() {
		return ErrPolicyDenied
	}
	if !AllowsPolicy(spec.Profile, policy) {
		return ErrPolicyDenied
	}
	if spec.DryRun && policy != PolicyAtomic {
		return ErrDryRunRequiresAtomic
	}

	if spec.Async && spec.Profile != ProfileRecord {
		return Errorf(CodePolicyDenied, "async is only allowed for record profile")
	}

	switch spec.Profile {
	case ProfileRecord:
		if strings.TrimSpace(spec.Model) == "" {
			return Errorf(CodeModelNotFound, "model is required for record profile")
		}
	case ProfileInitdata, ProfileTerminology:
		if strings.TrimSpace(spec.Module) == "" {
			return Errorf(CodeInvalidFormat, "module is required for "+string(spec.Profile)+" profile")
		}
	}

	if strings.TrimSpace(spec.Source.Format) == "" {
		return Errorf(CodeInvalidFormat, "source format is required")
	}

	return nil
}

func allowsCallerProfile(profile Profile, caller Caller) bool {
	switch profile {
	case ProfileInitdata:
		return caller == CallerLifecycle || caller == CallerE2E
	case ProfileTerminology:
		return caller == CallerLifecycle || caller == CallerCLI
	case ProfileRecord:
		return caller == CallerUser || caller == CallerCLI || caller == CallerLifecycle || caller == CallerE2E
	default:
		return false
	}
}

// EffectivePolicy returns spec policy or atomic default.
func EffectivePolicy(spec Spec) Policy {
	if spec.Policy == PolicyUnspecified {
		return PolicyAtomic
	}
	return spec.Policy
}

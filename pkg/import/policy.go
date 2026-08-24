// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg

// Policy controls transaction boundaries for a run.
type Policy string

const (
	PolicyAtomic      Policy = "atomic"
	PolicyStopKeep    Policy = "stop_keep"
	PolicyBestEffort  Policy = "best_effort"
	PolicyUnspecified Policy = ""
)

// Valid reports whether p is a known policy constant.
func (p Policy) Valid() bool {
	switch p {
	case PolicyAtomic, PolicyStopKeep, PolicyBestEffort:
		return true
	default:
		return false
	}
}

// AllowedPolicies returns policies permitted for a profile (§5.2).
func AllowedPolicies(profile Profile) []Policy {
	switch profile {
	case ProfileInitdata, ProfileTerminology:
		return []Policy{PolicyAtomic}
	case ProfileRecord:
		return []Policy{PolicyAtomic, PolicyStopKeep, PolicyBestEffort}
	default:
		return nil
	}
}

// AllowsPolicy reports whether profile may use policy.
func AllowsPolicy(profile Profile, policy Policy) bool {
	for _, allowed := range AllowedPolicies(profile) {
		if allowed == policy {
			return true
		}
	}
	return false
}

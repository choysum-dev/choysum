// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

// Deprecated: SpikeMountOptions was a PR-unit-vue-spike probe. Use @choysum/test-utils
// (pkg/jsengine/scripts/choysummount) via BuildFrontendVueHostBundle instead.
type SpikeMountOptions struct {
	Stubs map[string]bool
}

// Deprecated: SpikeWrapper was a PR-unit-vue-spike probe.
type SpikeWrapper struct {
	SetupRan      bool
	StubsAccepted map[string]bool
}

// Deprecated: use FrozenVTUSubsetAPIs.
var PlannedVTUSubsetAPIs = FrozenVTUSubsetAPIs

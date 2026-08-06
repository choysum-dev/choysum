// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

// Plan is the Decide output for C2 app-model inject.
type Plan struct {
	NeedInject      bool
	SupersedeInject bool
	// NeedEnsureServiceEntry is set when Spec.EnsureServiceEntry and the module
	// has no ServiceEntryPoint but Decide still wants NeedInject. Materialize of
	// the virtual service entry is PR-P2; P1 only reserves the flag.
	NeedEnsureServiceEntry bool
	// ScheduledApp is set when this session claimed the process-wide NeedInject slot.
	// Cleared via Session.ReleaseSchedules on failure or after Persist/Bundle.
	ScheduledApp string
}

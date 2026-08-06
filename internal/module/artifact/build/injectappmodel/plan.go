// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

// Plan is the Decide output for C2 app-model inject.
type Plan struct {
	NeedInject      bool
	SupersedeInject bool
	// ScheduledApp is set when this session claimed the process-wide NeedInject slot.
	// Cleared via Session.ReleaseSchedules on failure or after Persist/Bundle.
	ScheduledApp string
}

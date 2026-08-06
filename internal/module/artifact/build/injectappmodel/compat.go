// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import "strings"

// ReleaseSchedule clears one NeedInject claim for modelName/app on DefaultRegistry.
func ReleaseSchedule(modelName, app string) {
	app = strings.TrimSpace(app)
	if app == "" {
		return
	}
	DefaultRegistry().ReleaseClaim(modelName, app)
}

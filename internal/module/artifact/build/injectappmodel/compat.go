// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import "strings"

// ReleaseSchedule clears one process-wide NeedInject claim for modelName/app.
func ReleaseSchedule(modelName, app string) {
	app = strings.TrimSpace(app)
	if app == "" {
		return
	}
	if spec, ok := specByName(modelName); ok && spec.scheduled != nil {
		spec.scheduled.Delete(app)
	}
}

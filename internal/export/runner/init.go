// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import exportpkg "github.com/choysum-dev/choysum/pkg/export"

func init() {
	exportpkg.SetRun(Run)
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package inprocess

import "github.com/choysum-dev/choysum/pkg/bus"

func init() {
	bus.Register("inprocess", NewInProcessBus)
}

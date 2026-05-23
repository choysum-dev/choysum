// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package local

import (
	"github.com/choysum-dev/choysum/pkg/registry"
)

func init() {
	registry.Register("local", NewLocalRegistry)
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	stubreader "github.com/choysum-dev/choysum/internal/export/reader/stub"
	"github.com/choysum-dev/choysum/internal/export/registry"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

func init() {
	// Skeleton default: stub readers until PR-export-2 / PR-export-2t register real ones.
	registry.Register(exportpkg.ProfileRecord, stubreader.Reader{})
	registry.Register(exportpkg.ProfileTerminology, stubreader.Reader{})
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	termreader "github.com/choysum-dev/choysum/internal/export/reader/term"
	"github.com/choysum-dev/choysum/internal/export/registry"
	posink "github.com/choysum-dev/choysum/internal/export/sink/po"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

func init() {
	registry.Register(exportpkg.ProfileTerminology, termreader.Reader{})
	registry.RegisterSink("po", posink.Writer{})
}

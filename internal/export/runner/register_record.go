// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	recordreader "github.com/choysum-dev/choysum/internal/export/reader/record"
	"github.com/choysum-dev/choysum/internal/export/registry"
	csvsink "github.com/choysum-dev/choysum/internal/export/sink/csv"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

func init() {
	registry.Register(exportpkg.ProfileRecord, recordreader.Reader{})
	registry.RegisterSink("csv", csvsink.Writer{})
}

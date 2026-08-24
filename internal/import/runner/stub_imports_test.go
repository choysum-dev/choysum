// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner_test

import (
	_ "github.com/choysum-dev/choysum/internal/import/adapter/stub"
	"github.com/choysum-dev/choysum/internal/import/registry"
	stubwriter "github.com/choysum-dev/choysum/internal/import/writer/stub"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func init() {
	registry.RegisterWriter(importpkg.ProfileRecord, stubwriter.Writer{})
}

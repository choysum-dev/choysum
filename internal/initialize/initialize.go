// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initialize

import (
	_ "github.com/choysum-dev/choysum/internal/bus/inprocess"
	_ "github.com/choysum-dev/choysum/internal/defaultengine"
	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	_ "github.com/choysum-dev/choysum/internal/defaultscope"
	_ "github.com/choysum-dev/choysum/internal/document/storage/db"
	_ "github.com/choysum-dev/choysum/internal/document/storage/s3"
	_ "github.com/choysum-dev/choysum/internal/import/runner"
	_ "github.com/choysum-dev/choysum/internal/jwtauth"
	_ "github.com/choysum-dev/choysum/internal/registry/local"
	_ "github.com/choysum-dev/choysum/internal/registry/mdns"
)

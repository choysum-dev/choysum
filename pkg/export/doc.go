// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package exportpkg is the public Export platform contract (profile exportpkg).
//
// Import path: github.com/choysum-dev/choysum/pkg/export
//
// Typical use:
//
//	import exportpkg "github.com/choysum-dev/choysum/pkg/export"
//	report, err := exportpkg.Run(ctx, scope, spec)
//
// Report reuses pkg/import.Report (snake_case JSON).
package exportpkg

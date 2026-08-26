// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

import exportpkg "github.com/choysum-dev/choysum/pkg/export"

// Plan is the resolved export work description produced from Spec.
type Plan struct {
	Profile     exportpkg.Profile
	Caller      exportpkg.Caller
	Mode        exportpkg.Mode
	Format      string
	Model       string
	Fields      []string
	Domain      string
	Ids         []string
	Limit       int
	Offset      int
	Application string
	Module      string
	Lang        string
	CompanyID   string

	// StubUnitCount / StubFailUnitIndex are skeleton/test hooks (1-based fail index).
	StubUnitCount     int
	StubFailUnitIndex int
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportpkg

// Spec is the input to a single export run.
type Spec struct {
	Profile Profile `json:"profile"`
	Caller  Caller  `json:"caller"`
	Mode    Mode    `json:"mode,omitempty"`
	Format  string  `json:"format,omitempty"`

	Model  string   `json:"model,omitempty"`
	Fields []string `json:"fields,omitempty"`
	Domain string   `json:"domain,omitempty"`
	Ids    []string `json:"ids,omitempty"`
	Limit  int      `json:"limit,omitempty"`
	Offset int      `json:"offset,omitempty"`

	Application string `json:"application,omitempty"`
	Module      string `json:"module,omitempty"`
	Lang        string `json:"lang,omitempty"`

	Options Options `json:"options,omitempty"`
	Async   bool    `json:"async,omitempty"`
}

// Options carries profile-specific extensions.
type Options struct {
	CompanyID         string `json:"company_id,omitempty"`
	StubUnitCount     int    `json:"stub_unit_count,omitempty"`      // test hook
	StubFailUnitIndex int    `json:"stub_fail_unit_index,omitempty"` // test hook; 1-based
}

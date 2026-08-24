// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg

// Spec is the input to a single import run (§3.2).
type Spec struct {
	Profile     Profile `json:"profile"`
	Caller      Caller  `json:"caller"`
	Policy      Policy  `json:"policy"`
	DryRun      bool    `json:"dry_run"`
	Source      Source  `json:"source"`
	Module      string  `json:"module,omitempty"`
	Application string  `json:"application,omitempty"`
	Model       string  `json:"model,omitempty"`
	Options     Options `json:"options,omitempty"`
	Async       bool    `json:"async,omitempty"`
}

// Source describes import input bytes or references.
type Source struct {
	Format      string `json:"format,omitempty"`
	DocumentRef string `json:"document_ref,omitempty"`
	Path        string `json:"path,omitempty"`
}

// Options carries profile-specific extensions.
type Options struct {
	WithDemo          bool              `json:"with_demo,omitempty"`
	ColumnMapping     map[string]string `json:"column_mapping,omitempty"`
	CompanyID         string            `json:"company_id,omitempty"`
	Limit             int               `json:"limit,omitempty"`
	NextOffset        int               `json:"next_offset,omitempty"`
	StubUnitCount     int               `json:"stub_unit_count,omitempty"`      // skeleton/tests only
	StubFailUnitIndex int               `json:"stub_fail_unit_index,omitempty"` // skeleton/tests only; 1-based
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

// TermOccurrence is one extractable `_t` / `_lt` literal call site.
type TermOccurrence struct {
	Module     string
	Scope      string
	Src        string
	Kind       string
	SourcePath string
	Line       int
	Col        int
}

// ExtractIssue records a non-fatal extract problem (e.g. non-literal msgid).
type ExtractIssue struct {
	Severity   string
	Code       string
	Message    string
	SourcePath string
	Line       int
	Col        int
}

const (
	KindLiteral         = "literal"
	KindFieldLabel      = "field_label"
	KindSelectionLabel  = "selection_label"
	KindMenu            = "menu"
	KindRoute           = "route"
	KindAction          = "action"

	IssueNonLiteralMsgid    = "non_literal_msgid"
	IssueModuleMismatch     = "create_translate_module_mismatch"
	IssueSeverityWarn       = "warn"
	TemplateLocation        = "template"
)

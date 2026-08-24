// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg

// MessageType classifies an import report line.
type MessageType string

const (
	MessageError   MessageType = "error"
	MessageWarning MessageType = "warning"
	MessageSkip    MessageType = "skip"
)

// Message is one diagnostic row in a report.
type Message struct {
	Type      MessageType `json:"type"`
	Row       int         `json:"row,omitempty"`
	Field     string      `json:"field,omitempty"`
	Code      string      `json:"code,omitempty"`
	Text      string      `json:"text"`
	RecordRef string      `json:"record_ref,omitempty"`
}

// Stats aggregates unit outcomes.
type Stats struct {
	Total   int `json:"total"`
	Ok      int `json:"ok"`
	Error   int `json:"error"`
	Skip    int `json:"skip"`
	Warning int `json:"warning,omitempty"`
}

// ReportMeta optional metadata (terminology lang, etc.).
type ReportMeta struct {
	Lang        string `json:"lang,omitempty"`
	SourceRef   string `json:"source_ref,omitempty"`
	TargetModel string `json:"target_model,omitempty"`
}

// Report is the unified import result shape (ImportReport).
type Report struct {
	Profile     Profile     `json:"profile"`
	Policy      Policy      `json:"policy"`
	DryRun      bool        `json:"dry_run"`
	Stats       Stats       `json:"stats"`
	Messages    []Message   `json:"messages"`
	ArtifactRef string      `json:"artifact_ref,omitempty"`
	Meta        *ReportMeta `json:"meta,omitempty"`
}

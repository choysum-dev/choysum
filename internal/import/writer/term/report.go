// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package term

import (
	"fmt"
	"strings"

	i18nimport "github.com/choysum-dev/choysum/internal/i18n/import"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

const (
	codeOverrideSkip   = "override_skip"
	codeRejectedNoCtxt = "rejected_no_msgctxt"
	codeObsoleteSkip   = "obsolete_skip"
	codePurgedRetired  = "purged_retired"
)

// StatsToReport maps PO ImportStats to ImportReport fields (appendix E.3).
func StatsToReport(stats *i18nimport.ImportStats) importpkg.Report {
	if stats == nil {
		return importpkg.Report{Profile: importpkg.ProfileTerminology}
	}
	messages := make([]importpkg.Message, 0, 4)
	skip := 0
	warning := 0

	if stats.SkippedOverride > 0 {
		skip += stats.SkippedOverride
		messages = append(messages, importpkg.Message{
			Type: importpkg.MessageSkip,
			Code: codeOverrideSkip,
			Text: fmt.Sprintf("skipped %d packaged term(s) with Source=override", stats.SkippedOverride),
		})
	}
	if stats.RejectedNoCtxt > 0 {
		skip += stats.RejectedNoCtxt
		messages = append(messages, importpkg.Message{
			Type: importpkg.MessageSkip,
			Code: codeRejectedNoCtxt,
			Text: fmt.Sprintf("rejected %d PO entr(y/ies) without msgctxt", stats.RejectedNoCtxt),
		})
	}
	if stats.SkippedObsolete > 0 {
		skip += stats.SkippedObsolete
		messages = append(messages, importpkg.Message{
			Type: importpkg.MessageSkip,
			Code: codeObsoleteSkip,
			Text: fmt.Sprintf("skipped %d obsolete (#~) PO entr(y/ies)", stats.SkippedObsolete),
		})
	}
	if stats.PurgedRetired > 0 {
		warning++
		messages = append(messages, importpkg.Message{
			Type: importpkg.MessageWarning,
			Code: codePurgedRetired,
			Text: fmt.Sprintf("purged %d retired S7 terminology row(s)", stats.PurgedRetired),
		})
	}

	var meta *importpkg.ReportMeta
	if lang := strings.TrimSpace(stats.Lang); lang != "" {
		meta = &importpkg.ReportMeta{Lang: lang}
	}

	total := stats.Upserted + skip + warning
	return importpkg.Report{
		Profile: importpkg.ProfileTerminology,
		Stats: importpkg.Stats{
			Total:   total,
			Ok:      stats.Upserted,
			Skip:    skip,
			Warning: warning,
		},
		Messages: messages,
		Meta:     meta,
	}
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	"context"

	"github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/registry"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Run is the default export runner implementation.
func Run(ctx context.Context, runtimeScope scope.Scope, spec exportpkg.Spec) (importpkg.Report, error) {
	p, err := PlanFromSpec(spec)
	if err != nil {
		return importpkg.Report{}, err
	}

	reader, err := registry.ReaderFor(spec.Profile)
	if err != nil {
		return importpkg.Report{}, err
	}

	result, readErr := reader.Read(ctx, runtimeScope, p)
	if readErr == nil && len(result.Headers) > 0 {
		if sink, sinkLookupErr := registry.SinkFor(p.Format); sinkLookupErr == nil {
			if sinkErr := sink.Write(ctx, runtimeScope, p, &result); sinkErr != nil {
				return importpkg.Report{}, sinkErr
			}
		}
	}
	report := importpkg.Report{
		Profile:  importpkg.Profile(spec.Profile),
		Policy:   importpkg.PolicyUnspecified,
		DryRun:   false,
		Messages: toImportMessages(result.Messages),
		Meta:     reportMeta(p),
	}

	syntheticErr := false
	if readErr != nil && !hasErrorClassMessage(report.Messages) {
		syntheticErr = true
		if expErr, ok := exportpkg.AsError(readErr); ok {
			report.Messages = append(report.Messages, importpkg.Message{
				Type:      importpkg.MessageError,
				Row:       expErr.Row,
				Field:     expErr.Field,
				Code:      expErr.Code,
				Text:      expErr.Text,
				RecordRef: expErr.RecordRef,
			})
		} else {
			report.Messages = append(report.Messages, importpkg.Message{
				Type: importpkg.MessageError,
				Text: readErr.Error(),
			})
		}
	}

	report.Stats = buildStats(result, report.Messages, syntheticErr)
	if readErr != nil {
		return report, readErr
	}
	return report, nil
}

func buildStats(result registry.Result, messages []importpkg.Message, syntheticErr bool) importpkg.Stats {
	if result.HasOutcomes() {
		stats := importpkg.Stats{
			Total:   result.Outcomes.Total,
			Ok:      result.Outcomes.Ok,
			Error:   result.Outcomes.Error,
			Skip:    result.Outcomes.Skip,
			Warning: result.Outcomes.Warning,
		}
		if syntheticErr && stats.Error == 0 {
			if stats.Ok > 0 {
				stats.Ok--
			} else if stats.Skip > 0 {
				stats.Skip--
			}
			stats.Error++
		}
		return stats
	}

	total := result.UnitCount
	if total == 0 {
		total = len(messages)
	}
	return statsFromMessages(total, messages)
}

func statsFromMessages(total int, messages []importpkg.Message) importpkg.Stats {
	rowOutcome := make(map[int]importpkg.MessageType)
	var warnCount, errNoRow, skipNoRow int

	for _, msg := range messages {
		typ := msg.Type
		if typ == "" {
			typ = importpkg.MessageError
		}
		if typ == importpkg.MessageWarning {
			warnCount++
			continue
		}
		if msg.Row <= 0 {
			if typ == importpkg.MessageSkip {
				skipNoRow++
			} else {
				errNoRow++
			}
			continue
		}
		if cur, ok := rowOutcome[msg.Row]; !ok || outcomeRank(typ) > outcomeRank(cur) {
			rowOutcome[msg.Row] = typ
		}
	}

	var errCount, skipCount int
	for _, typ := range rowOutcome {
		if typ == importpkg.MessageSkip {
			skipCount++
		} else {
			errCount++
		}
	}
	errCount += errNoRow
	skipCount += skipNoRow

	ok := total - errCount - skipCount
	if ok < 0 {
		ok = 0
	}
	return importpkg.Stats{
		Total:   total,
		Ok:      ok,
		Error:   errCount,
		Skip:    skipCount,
		Warning: warnCount,
	}
}

func outcomeRank(typ importpkg.MessageType) int {
	switch typ {
	case importpkg.MessageError:
		return 3
	case importpkg.MessageSkip:
		return 2
	case importpkg.MessageWarning:
		return 1
	default:
		return 3
	}
}

func hasErrorClassMessage(messages []importpkg.Message) bool {
	for _, msg := range messages {
		if msg.Type != importpkg.MessageSkip && msg.Type != importpkg.MessageWarning {
			return true
		}
	}
	return false
}

func reportMeta(p plan.Plan) *importpkg.ReportMeta {
	if p.Lang == "" && p.Model == "" {
		return nil
	}
	return &importpkg.ReportMeta{
		Lang:        p.Lang,
		TargetModel: p.Model,
	}
}

func toImportMessages(msgs []registry.Message) []importpkg.Message {
	if len(msgs) == 0 {
		return nil
	}
	out := make([]importpkg.Message, 0, len(msgs))
	for _, msg := range msgs {
		typ := importpkg.MessageType(msg.Type)
		if typ == "" {
			typ = importpkg.MessageError
		}
		out = append(out, importpkg.Message{
			Type:      typ,
			Row:       msg.Row,
			Field:     msg.Field,
			Code:      msg.Code,
			Text:      msg.Text,
			RecordRef: msg.RecordRef,
		})
	}
	return out
}

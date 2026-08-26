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
	report := importpkg.Report{
		Profile:  importpkg.Profile(spec.Profile),
		Policy:   importpkg.PolicyUnspecified,
		DryRun:   false,
		Stats:    importpkg.Stats{Total: result.UnitCount},
		Messages: toImportMessages(result.Messages),
		Meta:     reportMeta(p),
	}

	if readErr != nil && len(report.Messages) == 0 {
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

	report.Stats = statsFromMessages(result.UnitCount, report.Messages)
	if readErr != nil {
		return report, readErr
	}
	return report, nil
}

func statsFromMessages(total int, messages []importpkg.Message) importpkg.Stats {
	var errCount, skipCount, warnCount int
	for _, msg := range messages {
		switch msg.Type {
		case importpkg.MessageSkip:
			skipCount++
		case importpkg.MessageWarning:
			warnCount++
		default:
			errCount++
		}
	}
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

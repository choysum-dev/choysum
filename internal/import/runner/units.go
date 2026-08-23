// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	"github.com/choysum-dev/choysum/internal/import/plan"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

type messageCollector struct {
	messages []importpkg.Message
	ok       int
	errors   int
	skip     int
	warning  int
	firstErr error
}

func newMessageCollector(total int) *messageCollector {
	return &messageCollector{}
}

func (c *messageCollector) addOK(n int) {
	if n <= 0 {
		return
	}
	c.ok += n
}

func (c *messageCollector) addError(err error, unit plan.Unit) {
	c.errors++
	if c.firstErr == nil {
		c.firstErr = err
	}
	if impErr, ok := importpkg.AsError(err); ok {
		msg := impErr.Message()
		if msg.Row == 0 && unit != nil {
			msg.Row = unit.UnitIndex()
		}
		c.messages = append(c.messages, msg)
		return
	}
	row := 0
	if unit != nil {
		row = unit.UnitIndex()
	}
	c.messages = append(c.messages, importpkg.Message{
		Type: importpkg.MessageError,
		Row:  row,
		Code: importpkg.CodeInvalidFormat,
		Text: err.Error(),
	})
}

func (c *messageCollector) hasHardError() bool {
	return c.errors > 0
}

func (c *messageCollector) stats(total int) importpkg.Stats {
	return importpkg.Stats{
		Total:   total,
		Ok:      c.ok,
		Error:   c.errors,
		Skip:    c.skip,
		Warning: c.warning,
	}
}

func (c *messageCollector) buildReport(spec importpkg.Spec, total int) importpkg.Report {
	return importpkg.Report{
		Profile:  spec.Profile,
		Policy:   importpkg.EffectivePolicy(spec),
		DryRun:   spec.DryRun,
		Stats:    c.stats(total),
		Messages: append([]importpkg.Message(nil), c.messages...),
		Meta:     reportMeta(spec),
	}
}

func reportMeta(spec importpkg.Spec) *importpkg.ReportMeta {
	if spec.Source.DocumentRef == "" && spec.Model == "" {
		return nil
	}
	return &importpkg.ReportMeta{
		SourceRef:   spec.Source.DocumentRef,
		TargetModel: spec.Model,
	}
}

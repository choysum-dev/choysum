// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package term

import (
	"testing"

	i18nimport "github.com/choysum-dev/choysum/internal/i18n/import"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestStatsToReport_OverrideSkip(t *testing.T) {
	report := StatsToReport(&i18nimport.ImportStats{
		Upserted:        2,
		SkippedOverride: 3,
		Lang:            "zh_CN",
	})
	if report.Stats.Ok != 2 || report.Stats.Skip != 3 {
		t.Fatalf("stats = %+v", report.Stats)
	}
	if report.Meta == nil || report.Meta.Lang != "zh_CN" {
		t.Fatalf("meta = %+v", report.Meta)
	}
	found := false
	for _, msg := range report.Messages {
		if msg.Code == codeOverrideSkip && msg.Type == importpkg.MessageSkip {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("messages = %+v, want override_skip", report.Messages)
	}
}

func TestStatsToReport_RejectedNoCtxt(t *testing.T) {
	report := StatsToReport(&i18nimport.ImportStats{
		RejectedNoCtxt:  4,
		SkippedObsolete: 1,
		PurgedRetired:   2,
		Lang:            "en_US",
	})
	if report.Stats.Skip != 5 || report.Stats.Warning != 1 {
		t.Fatalf("stats = %+v", report.Stats)
	}
	codes := map[string]bool{}
	for _, msg := range report.Messages {
		codes[msg.Code] = true
		if msg.Code == codeRejectedNoCtxt && msg.Type != importpkg.MessageSkip {
			t.Fatalf("rejected_no_msgctxt type = %q", msg.Type)
		}
		if msg.Code == codePurgedRetired && msg.Type != importpkg.MessageWarning {
			t.Fatalf("purged_retired type = %q", msg.Type)
		}
	}
	for _, want := range []string{codeRejectedNoCtxt, codeObsoleteSkip, codePurgedRetired} {
		if !codes[want] {
			t.Fatalf("missing code %q in %+v", want, report.Messages)
		}
	}
}

func TestStatsToReport_nil(t *testing.T) {
	report := StatsToReport(nil)
	if report.Profile != importpkg.ProfileTerminology || report.Stats.Total != 0 {
		t.Fatalf("nil stats report = %+v", report)
	}
}

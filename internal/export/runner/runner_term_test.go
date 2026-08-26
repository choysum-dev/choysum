// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/plan"
	stubreader "github.com/choysum-dev/choysum/internal/export/reader/stub"
	"github.com/choysum-dev/choysum/internal/export/registry"
	posink "github.com/choysum-dev/choysum/internal/export/sink/po"
	"github.com/choysum-dev/choysum/internal/i18n/po"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

func TestRunWithResult_terminologyPO(t *testing.T) {
	registry.ResetForTest()
	registry.ResetSinksForTest()
	t.Cleanup(func() {
		registry.Register(exportpkg.ProfileRecord, stubreader.Reader{})
		registry.Register(exportpkg.ProfileTerminology, stubreader.Reader{})
	})

	registry.Register(exportpkg.ProfileTerminology, fakeReader{
		result: registry.Result{
			POEntries: []po.Entry{{Msgid: "Hello", Msgstr: "你好"}},
			Outcomes:  registry.Outcomes{Total: 1, Ok: 1},
		},
	})
	registry.RegisterSink("po", posink.Writer{})

	report, result, err := RunWithResult(context.Background(), nil, exportpkg.Spec{
		Profile:     exportpkg.ProfileTerminology,
		Caller:      exportpkg.CallerUser,
		Application: "auth",
		Module:      "auth",
		Lang:        "zh_CN",
		Format:      "po",
	})
	if err != nil {
		t.Fatalf("RunWithResult: %v", err)
	}
	if len(result.POBytes) == 0 || report.Stats.Total != 1 {
		t.Fatalf("poBytes=%d stats=%+v", len(result.POBytes), report.Stats)
	}
}

func TestNeedsSink_emptyFormat(t *testing.T) {
	if needsSink(plan.Plan{Format: ""}, registry.Result{Headers: []string{"A"}}) {
		t.Fatal("empty format should skip sink")
	}
}

func TestNeedsSink_terminologyWithEntries(t *testing.T) {
	if !needsSink(planFromProfile(exportpkg.ProfileTerminology), registry.Result{POEntries: []po.Entry{{Msgid: "x"}}}) {
		t.Fatal("terminology with PO entries should need sink")
	}
}

func TestNeedsSink_recordRequiresHeaders(t *testing.T) {
	if !needsSink(plan.Plan{Profile: exportpkg.ProfileRecord, Format: "csv"}, registry.Result{Headers: []string{"A"}}) {
		t.Fatal("record export with headers should need sink")
	}
}

func TestNeedsSink_terminologyWithoutEntries(t *testing.T) {
	if needsSink(planFromProfile(exportpkg.ProfileTerminology), registry.Result{}) {
		t.Fatal("empty PO entries should skip sink")
	}
}

func planFromProfile(profile exportpkg.Profile) plan.Plan {
	return plan.Plan{Profile: profile, Format: "po"}
}

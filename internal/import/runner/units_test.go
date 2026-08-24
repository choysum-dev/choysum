// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/import/plan"
	planstub "github.com/choysum-dev/choysum/internal/import/plan/stub"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestExecutePlan_nilScope(t *testing.T) {
	_, err := executePlan(context.Background(), nil, importpkg.Spec{}, plan.Plan{}, nil)
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("executePlan(nil scope) error = %v", err)
	}
}

func TestExecutePlan_unknownPolicy(t *testing.T) {
	runtimeScope := testScope(t)
	_, err := executePlan(context.Background(), runtimeScope, importpkg.Spec{Policy: importpkg.Policy("bogus")}, plan.Plan{}, nil)
	if !errors.Is(err, importpkg.ErrPolicyDenied) {
		t.Fatalf("executePlan(unknown policy) error = %v", err)
	}
}

func TestMessageCollector(t *testing.T) {
	c := newMessageCollector(3)
	c.addOK(0)
	c.addSkip(0)

	c.addOK(1)
	c.addSkip(2)

	impErr := &importpkg.Error{Code: "constraint", Text: "bad"}
	c.addError(impErr, planstub.Unit{Index: 9})
	c.addError(errors.New("plain"), planstub.Unit{Index: 4})

	report := c.buildReport(importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Policy:  importpkg.PolicyBestEffort,
		DryRun:  true,
		Source:  importpkg.Source{DocumentRef: "doc-1"},
		Model:   "base.Country",
	}, 5)

	if report.Stats.Ok != 1 || report.Stats.Skip != 2 || report.Stats.Error != 2 {
		t.Fatalf("stats = %+v", report.Stats)
	}
	if report.Meta == nil || report.Meta.SourceRef != "doc-1" || report.Meta.TargetModel != "base.Country" {
		t.Fatalf("meta = %+v", report.Meta)
	}
	if len(report.Messages) != 2 {
		t.Fatalf("messages = %+v", report.Messages)
	}
	if report.Messages[0].Row != 9 {
		t.Fatalf("structured error row = %d, want unit index 9", report.Messages[0].Row)
	}
	if report.Messages[1].Row != 4 || report.Messages[1].Code != importpkg.CodeInvalidFormat {
		t.Fatalf("plain error message = %+v", report.Messages[1])
	}
}

func TestReportMeta_nilWhenEmpty(t *testing.T) {
	if got := reportMeta(importpkg.Spec{}); got != nil {
		t.Fatalf("reportMeta(empty) = %+v, want nil", got)
	}
}

func testScope(t *testing.T) scope.Scope {
	t.Helper()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "units.db"),
		},
	}
	return defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
}

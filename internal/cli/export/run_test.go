// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportcli

import (
	"context"
	"encoding/csv"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	importcli "github.com/choysum-dev/choysum/internal/cli/import"
	"github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/export/registry"
	"github.com/choysum-dev/choysum/internal/export/runner"
	_ "github.com/choysum-dev/choysum/internal/export/runner"
	importcsv "github.com/choysum-dev/choysum/internal/import/adapter/csv"
	importcaller "github.com/choysum-dev/choysum/internal/import/caller"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

func TestRecordSpecFromOptions(t *testing.T) {
	spec, err := recordSpecFromOptions(RecordOptions{
		Model:      "base.Country",
		OutputPath: filepath.Join(t.TempDir(), "base.Country.csv"),
	})
	if err != nil {
		t.Fatalf("recordSpecFromOptions: %v", err)
	}
	if spec.Profile != exportpkg.ProfileRecord || spec.Caller != exportpkg.CallerCLI {
		t.Fatalf("spec = %+v", spec)
	}
	if spec.Format != "csv" || spec.Mode != exportpkg.ModeData {
		t.Fatalf("format/mode = %q/%q", spec.Format, spec.Mode)
	}
}

func TestRecordSpecFromOptionsValidation(t *testing.T) {
	t.Run("requires model", func(t *testing.T) {
		_, err := recordSpecFromOptions(RecordOptions{OutputPath: "base.Country.csv"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("requires output path", func(t *testing.T) {
		_, err := recordSpecFromOptions(RecordOptions{Model: "base.Country"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("requires csv extension", func(t *testing.T) {
		_, err := recordSpecFromOptions(RecordOptions{Model: "base.Country", OutputPath: "base.Country.po"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("template mode", func(t *testing.T) {
		spec, err := recordSpecFromOptions(RecordOptions{
			Model:      "base.Country",
			OutputPath: "base.Country.csv",
			Mode:       exportpkg.ModeTemplate,
		})
		if err != nil {
			t.Fatalf("recordSpecFromOptions: %v", err)
		}
		if spec.Mode != exportpkg.ModeTemplate {
			t.Fatalf("mode = %q", spec.Mode)
		}
	})
}

func TestModelFromFilenameForExportOutput(t *testing.T) {
	model, err := importcli.ModelFromFilename(filepath.Join(t.TempDir(), "base_Country.csv"))
	if err != nil {
		t.Fatalf("ModelFromFilename: %v", err)
	}
	if model != "base.Country" {
		t.Fatalf("model = %q", model)
	}
}

func TestRunRecord_CountryCSV(t *testing.T) {
	runtimeScope := newRecordExportTestScope(t)
	seedRecordExportCountryRows(t, runtimeScope)

	prev := runRecordExport
	runRecordExport = func(ctx context.Context, rs scope.Scope, spec exportpkg.Spec) (importpkg.Report, registry.Result, error) {
		ctx = importcaller.ContextWithCaller(ctx, &recordExportCountryCaller{scope: rs})
		return runner.RunWithResult(ctx, rs, spec)
	}
	t.Cleanup(func() { runRecordExport = prev })

	prevExec := newCLIExportExecutor
	newCLIExportExecutor = func(scope.Scope) (jsexecutor.JsExecutor, error) {
		return stubCLIExportExecutor{}, nil
	}
	t.Cleanup(func() { newCLIExportExecutor = prevExec })

	outPath := filepath.Join(t.TempDir(), "base.Country.csv")
	report, csvBytes, err := RunRecord(context.Background(), runtimeScope, RecordOptions{
		Model:      "base.Country",
		OutputPath: outPath,
	})
	if err != nil {
		t.Fatalf("RunRecord: %v", err)
	}
	if report.Stats.Ok != 2 || len(csvBytes) == 0 {
		t.Fatalf("report=%+v csvLen=%d", report.Stats, len(csvBytes))
	}

	table, err := csv.NewReader(strings.NewReader(string(importcsv.StripUTF8BOM(csvBytes)))).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(table) < 2 {
		t.Fatalf("rows = %d, want at least header + data", len(table))
	}
}

type stubCLIExportExecutor struct{}

func (stubCLIExportExecutor) AppendJsScripts(...*jsengine.JsScript) {}
func (stubCLIExportExecutor) Start() error                          { return nil }
func (stubCLIExportExecutor) Stop() error                           { return nil }
func (stubCLIExportExecutor) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return nil, nil
}
func (stubCLIExportExecutor) GetJsScripts() []*jsengine.JsScript { return nil }
func (stubCLIExportExecutor) SetJsScripts([]*jsengine.JsScript)  {}
func (stubCLIExportExecutor) Reload(...*jsengine.JsScript) error { return nil }

type recordExportCountryCaller struct {
	scope scope.Scope
}

func (c *recordExportCountryCaller) Call(ctx context.Context, req importcaller.CallRequest) (any, error) {
	db := c.dbFromContext(ctx)
	switch req.Model + "." + req.Method {
	case "base.Country.Search":
		var rows []recordExportCountryRow
		q := db.Table("base_country c").
			Select("c.id, c.name, c.code, c.zip_required, c.state_required, c.is_active, cur.code AS currency_code").
			Joins("LEFT JOIN base_currency cur ON cur.id = c.default_currency_id")
		if opts, ok := req.Args[1].(map[string]any); ok {
			if limit, ok := opts["limit"].(int); ok && limit > 0 {
				q = q.Limit(limit)
			}
		}
		if err := q.Order("c.code asc").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]any, 0, len(rows))
		for _, row := range rows {
			rec := map[string]any{
				"Id":            row.ID,
				"Name":          row.Name,
				"Code":          row.Code,
				"ZipRequired":   row.ZipRequired,
				"StateRequired": row.StateRequired,
				"IsActive":      row.IsActive,
			}
			if row.CurrencyCode != "" {
				rec["DefaultCurrencyId/Code"] = row.CurrencyCode
			}
			out = append(out, rec)
		}
		return out, nil
	default:
		return nil, nil
	}
}

func (c *recordExportCountryCaller) dbFromContext(ctx context.Context) *gorm.DB {
	if tx, ok := scope.TransactionFromContext(ctx); ok && tx != nil && tx.Session() != nil {
		return tx.Session().DB
	}
	if rs, ok := scope.ScopeFromContext(ctx); ok && rs != nil && rs.Session() != nil {
		return rs.Session().DB
	}
	return c.scope.Session().DB
}

type recordExportCountryRow struct {
	ID            string `gorm:"column:id"`
	Name          string `gorm:"column:name"`
	Code          string `gorm:"column:code"`
	ZipRequired   bool   `gorm:"column:zip_required"`
	StateRequired bool   `gorm:"column:state_required"`
	IsActive      bool   `gorm:"column:is_active"`
	CurrencyCode  string `gorm:"column:currency_code"`
}

type recordExportCountryTable struct {
	ID                string    `gorm:"column:id;primaryKey"`
	Name              string    `gorm:"column:name"`
	Code              string    `gorm:"column:code;uniqueIndex"`
	ZipRequired       bool      `gorm:"column:zip_required"`
	StateRequired     bool      `gorm:"column:state_required"`
	IsActive          bool      `gorm:"column:is_active"`
	DefaultCurrencyID string    `gorm:"column:default_currency_id"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

func (recordExportCountryTable) TableName() string { return "base_country" }

type recordExportCurrencyTable struct {
	ID   string `gorm:"column:id;primaryKey"`
	Code string `gorm:"column:code;uniqueIndex"`
	Name string `gorm:"column:name"`
}

func (recordExportCurrencyTable) TableName() string { return "base_currency" }

func newRecordExportTestScope(t *testing.T) scope.Scope {
	t.Helper()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "record-export-cli.db"),
		},
	}
	return defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
}

func seedRecordExportCountryRows(t *testing.T, runtimeScope scope.Scope) {
	t.Helper()
	db := runtimeScope.Session().DB
	if err := db.AutoMigrate(&recordExportCountryTable{}, &recordExportCurrencyTable{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := db.Create(&recordExportCurrencyTable{ID: "cur-cny", Code: "CNY", Name: "CNY"}).Error; err != nil {
		t.Fatalf("seed currency: %v", err)
	}
	rows := []recordExportCountryTable{
		{ID: xid.New().String(), Name: "Export Alpha", Code: "EXA001", ZipRequired: true, StateRequired: false, IsActive: true, DefaultCurrencyID: "cur-cny", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: xid.New().String(), Name: "Export Beta", Code: "EXB002", ZipRequired: false, StateRequired: true, IsActive: true, DefaultCurrencyID: "cur-cny", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed country: %v", err)
		}
	}
}

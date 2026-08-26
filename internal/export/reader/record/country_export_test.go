// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record_test

import (
	"context"
	"encoding/csv"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	exportplan "github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/reader/record"
	_ "github.com/choysum-dev/choysum/internal/export/runner"
	csvsink "github.com/choysum-dev/choysum/internal/export/sink/csv"
	importcsv "github.com/choysum-dev/choysum/internal/import/adapter/csv"
	importcaller "github.com/choysum-dev/choysum/internal/import/caller"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

func TestExport_Country_CSVRoundTripShape(t *testing.T) {
	runtimeScope := newCountryExportScope(t)
	seedCountryExportRows(t, runtimeScope)

	ctx := importcaller.ContextWithCaller(context.Background(), &countryExportCaller{scope: runtimeScope})
	spec := exportplan.Plan{
		Profile: exportpkg.ProfileRecord,
		Model:   "base.Country",
		Mode:    exportpkg.ModeData,
		Format:  "csv",
	}

	result, err := record.Reader{}.Read(ctx, runtimeScope, spec)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if result.Outcomes.Total != 2 {
		t.Fatalf("total = %d, want 2", result.Outcomes.Total)
	}

	w := csvsink.Writer{}
	if err := w.Write(ctx, runtimeScope, spec, &result); err != nil {
		t.Fatalf("Write: %v", err)
	}

	table, err := csv.NewReader(strings.NewReader(string(importcsv.StripUTF8BOM(result.CSVBytes)))).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	wantHeader := []string{"Name", "Code", "DefaultCurrencyId/Code", "ZipRequired", "StateRequired", "IsActive"}
	if len(table) != 3 {
		t.Fatalf("rows = %d, want 3", len(table))
	}
	for i, col := range wantHeader {
		if table[0][i] != col {
			t.Fatalf("header[%d] = %q, want %q", i, table[0][i], col)
		}
	}
	if table[1][0] != "Export Alpha" || table[1][2] != "CNY" {
		t.Fatalf("row1 = %#v", table[1])
	}
}

func TestExport_Country_RunIntegration(t *testing.T) {
	runtimeScope := newCountryExportScope(t)
	seedCountryExportRows(t, runtimeScope)
	ctx := importcaller.ContextWithCaller(context.Background(), &countryExportCaller{scope: runtimeScope})

	report, err := exportpkg.Run(ctx, runtimeScope, exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerUser,
		Model:   "base.Country",
		Format:  "csv",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Stats.Total != 2 || report.Stats.Ok != 2 {
		t.Fatalf("stats = %+v", report.Stats)
	}
}

type countryExportCaller struct {
	scope scope.Scope
}

func (c *countryExportCaller) Call(ctx context.Context, req importcaller.CallRequest) (any, error) {
	db := c.dbFromContext(ctx)
	switch req.Model + "." + req.Method {
	case "base.Country.Search":
		var rows []exportCountryRow
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

func (c *countryExportCaller) dbFromContext(ctx context.Context) *gorm.DB {
	if tx, ok := scope.TransactionFromContext(ctx); ok && tx != nil && tx.Session() != nil {
		return tx.Session().DB
	}
	if rs, ok := scope.ScopeFromContext(ctx); ok && rs != nil && rs.Session() != nil {
		return rs.Session().DB
	}
	return c.scope.Session().DB
}

type exportCountryRow struct {
	ID            string `gorm:"column:id"`
	Name          string `gorm:"column:name"`
	Code          string `gorm:"column:code"`
	ZipRequired   bool   `gorm:"column:zip_required"`
	StateRequired bool   `gorm:"column:state_required"`
	IsActive      bool   `gorm:"column:is_active"`
	CurrencyCode  string `gorm:"column:currency_code"`
}

func newCountryExportScope(t *testing.T) scope.Scope {
	t.Helper()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "country-export.db"),
		},
	}
	runtimeScope := defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err := runtimeScope.Session().AutoMigrate(&exportCountryTable{}, &exportCurrencyTable{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return runtimeScope
}

type exportCountryTable struct {
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

func (exportCountryTable) TableName() string { return "base_country" }

type exportCurrencyTable struct {
	ID   string `gorm:"column:id;primaryKey"`
	Code string `gorm:"column:code;uniqueIndex"`
	Name string `gorm:"column:name"`
}

func (exportCurrencyTable) TableName() string { return "base_currency" }

func seedCountryExportRows(t *testing.T, runtimeScope scope.Scope) {
	t.Helper()
	db := runtimeScope.Session().DB
	if err := db.Create(&exportCurrencyTable{ID: "cur-cny", Code: "CNY", Name: "CNY"}).Error; err != nil {
		t.Fatalf("seed currency: %v", err)
	}
	rows := []exportCountryTable{
		{ID: xid.New().String(), Name: "Export Alpha", Code: "EXA001", ZipRequired: true, StateRequired: false, IsActive: true, DefaultCurrencyID: "cur-cny", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: xid.New().String(), Name: "Export Beta", Code: "EXB002", ZipRequired: false, StateRequired: true, IsActive: true, DefaultCurrencyID: "cur-cny", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed country: %v", err)
		}
	}
}

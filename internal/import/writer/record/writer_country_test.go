// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/import/runner"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

func TestCountryImport_AtomicRollback(t *testing.T) {
	runtimeScope := newCountryImportScope(t)
	path := writeCountryCSV(t, ""+
		"Name,Code,IsActive,ZipRequired,StateRequired\n"+
		"Good,GC001,true,true,false\n"+
		"Bad,,true,true,false\n")
	spec := countryImportSpec(path)

	_, err := runner.Run(context.Background(), runtimeScope, spec)
	if err == nil {
		t.Fatal("expected import error")
	}
	if count := countCountries(t, runtimeScope); count != 0 {
		t.Fatalf("country count = %d, want 0 after atomic rollback", count)
	}
}

func TestCountryImport_DryRun(t *testing.T) {
	runtimeScope := newCountryImportScope(t)
	path := writeCountryCSV(t, "Name,Code,IsActive,ZipRequired,StateRequired\nDry,DR001,true,true,false\n")
	spec := countryImportSpec(path)
	spec.DryRun = true

	report, err := runner.Run(context.Background(), runtimeScope, spec)
	if err != nil {
		t.Fatalf("Run dry-run: %v", err)
	}
	if !report.DryRun || report.Stats.Ok != 1 {
		t.Fatalf("report = %#v", report)
	}
	if count := countCountries(t, runtimeScope); count != 0 {
		t.Fatalf("country count = %d, want 0 after dry-run", count)
	}
}

func TestCountryImport_CodeUpsert(t *testing.T) {
	runtimeScope := newCountryImportScope(t)
	path := writeCountryCSV(t, "Name,Code,IsActive,ZipRequired,StateRequired\nFirst,UP001,true,true,false\n")
	spec := countryImportSpec(path)
	if _, err := runner.Run(context.Background(), runtimeScope, spec); err != nil {
		t.Fatalf("first import: %v", err)
	}

	path = writeCountryCSV(t, "Name,Code,IsActive,ZipRequired,StateRequired\nSecond,UP001,true,true,false\n")
	spec = countryImportSpec(path)
	if _, err := runner.Run(context.Background(), runtimeScope, spec); err != nil {
		t.Fatalf("second import: %v", err)
	}
	if count := countCountries(t, runtimeScope); count != 1 {
		t.Fatalf("country count = %d, want 1", count)
	}
	var name string
	if err := runtimeScope.Session().DB.Table("base_country").Select("name").Where("code = ?", "UP001").Scan(&name).Error; err != nil {
		t.Fatalf("scan name: %v", err)
	}
	if name != "Second" {
		t.Fatalf("name = %q, want Second", name)
	}
}

func countryImportSpec(path string) importpkg.Spec {
	return importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Model:   "base.Country",
		Source: importpkg.Source{
			Format: "csv",
			Path:   path,
		},
	}
}

func newCountryImportScope(t *testing.T) scope.Scope {
	t.Helper()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "country-import.db"),
		},
	}
	runtimeScope := defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	seedCountryImportSchema(t, runtimeScope.Session().DB)
	return runtimeScope
}

func seedCountryImportSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&meta.Module{}, &meta.Model{}, &meta.Field{}, &testCountryRow{}, &testCurrencyRow{}, &modmeta.ModelData{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	countryModel := &meta.Model{Name: "Country", Application: "base", Path: "/tmp", ModelTable: "base_country"}
	currencyModel := &meta.Model{Name: "Currency", Application: "base", Path: "/tmp", ModelTable: "base_currency"}
	if err := db.Create(countryModel).Error; err != nil {
		t.Fatalf("seed country model: %v", err)
	}
	if err := db.Create(currencyModel).Error; err != nil {
		t.Fatalf("seed currency model: %v", err)
	}
	if err := db.Create(&meta.Field{Name: "DefaultCurrencyId", FieldType: "ManyToOne", ModelId: countryModel.Id}).Error; err != nil {
		t.Fatalf("seed country field: %v", err)
	}
	if err := db.Create(&testCurrencyRow{ID: "cur-cny", Code: "CNY", Name: "CNY", IsActive: true}).Error; err != nil {
		t.Fatalf("seed currency row: %v", err)
	}
}

type testCountryRow struct {
	ID            string    `gorm:"column:id;primaryKey"`
	Name          string    `gorm:"column:name"`
	Code          string    `gorm:"column:code;uniqueIndex"`
	IsActive      bool      `gorm:"column:is_active"`
	ZipRequired   bool      `gorm:"column:zip_required"`
	StateRequired bool      `gorm:"column:state_required"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (testCountryRow) TableName() string { return "base_country" }

type testCurrencyRow struct {
	ID       string `gorm:"column:id;primaryKey"`
	Code     string `gorm:"column:code;uniqueIndex"`
	Name     string `gorm:"column:name"`
	IsActive bool   `gorm:"column:is_active"`
}

func (testCurrencyRow) TableName() string { return "base_currency" }

func countCountries(t *testing.T, runtimeScope scope.Scope) int64 {
	t.Helper()
	var count int64
	if err := runtimeScope.Session().DB.Table("base_country").Count(&count).Error; err != nil {
		t.Fatalf("Count: %v", err)
	}
	return count
}

func writeCountryCSV(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "countries.csv")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}
	return path
}

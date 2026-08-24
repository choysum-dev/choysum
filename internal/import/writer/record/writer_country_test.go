// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	importcaller "github.com/choysum-dev/choysum/internal/import/caller"
	"github.com/choysum-dev/choysum/internal/import/runner"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

func TestCountryImport_AtomicRollback(t *testing.T) {
	runtimeScope := newCountryImportScope(t)
	path := writeCountryCSV(t, ""+
		"Name,Code,IsActive,ZipRequired,StateRequired\n"+
		"Good,GC001,true,true,false\n"+
		"Bad,,true,true,false\n")
	spec := countryImportSpec(path)

	_, err := runner.Run(withTestORMCaller(runtimeScope), runtimeScope, spec)
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

	report, err := runner.Run(withTestORMCaller(runtimeScope), runtimeScope, spec)
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

func TestCountryImport_CurrencyM2O(t *testing.T) {
	runtimeScope := newCountryImportScope(t)
	path := writeCountryCSV(t, "Name,Code,DefaultCurrencyId/Code,IsActive,ZipRequired,StateRequired\nM2O,MO001,CNY,true,true,false\n")
	spec := countryImportSpec(path)
	if _, err := runner.Run(withTestORMCaller(runtimeScope), runtimeScope, spec); err != nil {
		t.Fatalf("import: %v", err)
	}
	var currencyID string
	if err := runtimeScope.Session().DB.Table("base_country").Select("default_currency_id").Where("code = ?", "MO001").Scan(&currencyID).Error; err != nil {
		t.Fatalf("scan: %v", err)
	}
	if currencyID != "cur-cny" {
		t.Fatalf("default_currency_id = %q, want cur-cny", currencyID)
	}
}

func TestCountryImport_ExternalIDCreateAndUpdate(t *testing.T) {
	runtimeScope := newCountryImportScope(t)
	path := writeCountryCSV(t, "id,Name,Code,IsActive,ZipRequired,StateRequired\nimport.ext1,First,EX001,true,true,false\n")
	if _, err := runner.Run(withTestORMCaller(runtimeScope), runtimeScope, countryImportSpec(path)); err != nil {
		t.Fatalf("create: %v", err)
	}
	path = writeCountryCSV(t, "id,Name,Code,IsActive,ZipRequired,StateRequired\nimport.ext1,Second,EX001,true,true,false\n")
	if _, err := runner.Run(withTestORMCaller(runtimeScope), runtimeScope, countryImportSpec(path)); err != nil {
		t.Fatalf("update: %v", err)
	}
	if count := countCountries(t, runtimeScope); count != 1 {
		t.Fatalf("country count = %d, want 1", count)
	}
	var name string
	if err := runtimeScope.Session().DB.Table("base_country").Select("name").Where("code = ?", "EX001").Scan(&name).Error; err != nil {
		t.Fatalf("scan: %v", err)
	}
	if name != "Second" {
		t.Fatalf("name = %q", name)
	}
}

func TestCountryImport_ExternalIDMissingMappedRowRecreates(t *testing.T) {
	runtimeScope := newCountryImportScope(t)
	path := writeCountryCSV(t, "id,Name,Code,IsActive,ZipRequired,StateRequired\nimport.gone,Gone,GN001,true,true,false\n")
	if _, err := runner.Run(withTestORMCaller(runtimeScope), runtimeScope, countryImportSpec(path)); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := runtimeScope.Session().DB.Exec("DELETE FROM base_country").Error; err != nil {
		t.Fatalf("delete countries: %v", err)
	}
	path = writeCountryCSV(t, "id,Name,Code,IsActive,ZipRequired,StateRequired\nimport.gone,Back,GN001,true,true,false\n")
	if _, err := runner.Run(withTestORMCaller(runtimeScope), runtimeScope, countryImportSpec(path)); err != nil {
		t.Fatalf("recreate: %v", err)
	}
	if count := countCountries(t, runtimeScope); count != 1 {
		t.Fatalf("country count = %d, want 1", count)
	}
}

func TestCountryImport_ProtectedExternalID(t *testing.T) {
	runtimeScope := newCountryImportScope(t)
	db := runtimeScope.Session().DB
	if err := db.Create(&meta.Module{Name: "base", ApplicationStr: "base", Path: "/tmp"}).Error; err != nil {
		t.Fatalf("seed module: %v", err)
	}
	path := writeCountryCSV(t, "id,Name,Code,IsActive,ZipRequired,StateRequired\nbase.blocked,X,BK001,true,true,false\n")
	if _, err := runner.Run(withTestORMCaller(runtimeScope), runtimeScope, countryImportSpec(path)); err == nil {
		t.Fatal("expected protected external id error")
	}
}

func TestCountryImport_M2ONotFound(t *testing.T) {
	runtimeScope := newCountryImportScope(t)
	path := writeCountryCSV(t, "Name,Code,DefaultCurrencyId/Code,IsActive,ZipRequired,StateRequired\nBad,BD001,ZZZ,true,true,false\n")
	if _, err := runner.Run(withTestORMCaller(runtimeScope), runtimeScope, countryImportSpec(path)); err == nil {
		t.Fatal("expected M2O not found error")
	}
}

func TestCountryImport_UnsupportedModel(t *testing.T) {
	runtimeScope := newCountryImportScope(t)
	path := writeCountryCSV(t, "Name,Code\nX,U1\n")
	spec := countryImportSpec(path)
	spec.Model = "base.Partner"
	if _, err := runner.Run(withTestORMCaller(runtimeScope), runtimeScope, spec); err == nil {
		t.Fatal("expected unsupported model error")
	}
}

func TestCountryImport_CodeUpsert(t *testing.T) {
	runtimeScope := newCountryImportScope(t)
	path := writeCountryCSV(t, "Name,Code,IsActive,ZipRequired,StateRequired\nFirst,UP001,true,true,false\n")
	spec := countryImportSpec(path)
	if _, err := runner.Run(withTestORMCaller(runtimeScope), runtimeScope, spec); err != nil {
		t.Fatalf("first import: %v", err)
	}

	path = writeCountryCSV(t, "Name,Code,IsActive,ZipRequired,StateRequired\nSecond,UP001,true,true,false\n")
	spec = countryImportSpec(path)
	if _, err := runner.Run(withTestORMCaller(runtimeScope), runtimeScope, spec); err != nil {
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

func withTestORMCaller(runtimeScope scope.Scope) context.Context {
	return importcaller.ContextWithCaller(context.Background(), &gormORMCaller{scope: runtimeScope})
}

// gormORMCaller is a Go-unit stand-in for TS ORM Create/UpdateById/Search.
type gormORMCaller struct {
	scope scope.Scope
}

func (c *gormORMCaller) Call(ctx context.Context, req importcaller.CallRequest) (any, error) {
	db := c.dbFromContext(ctx)
	switch req.Model + "." + req.Method {
	case "base.Country.Create":
		vals, _ := req.Args[0].(map[string]any)
		if code, _ := vals["Code"].(string); strings.TrimSpace(code) == "" {
			return nil, fmt.Errorf("Code is required")
		}
		id := xid.New().String()
		row := map[string]any{
			"id":             id,
			"name":           vals["Name"],
			"code":           vals["Code"],
			"is_active":      vals["IsActive"],
			"zip_required":   vals["ZipRequired"],
			"state_required": vals["StateRequired"],
			"created_at":     time.Now().UTC(),
			"updated_at":     time.Now().UTC(),
		}
		if cur, ok := vals["DefaultCurrencyId"].(string); ok && cur != "" {
			row["default_currency_id"] = cur
		}
		if err := db.Table("base_country").Create(row).Error; err != nil {
			return nil, err
		}
		return map[string]any{"Id": id, "Code": vals["Code"]}, nil
	case "base.Country.UpdateById":
		id, _ := req.Args[0].(string)
		vals, _ := req.Args[1].(map[string]any)
		updates := map[string]any{"updated_at": time.Now().UTC()}
		for k, v := range vals {
			switch k {
			case "Name":
				updates["name"] = v
			case "Code":
				updates["code"] = v
			case "IsActive":
				updates["is_active"] = v
			case "ZipRequired":
				updates["zip_required"] = v
			case "StateRequired":
				updates["state_required"] = v
			case "DefaultCurrencyId":
				updates["default_currency_id"] = v
			}
		}
		if err := db.Table("base_country").Where("id = ?", id).Updates(updates).Error; err != nil {
			return nil, err
		}
		return map[string]any{"Id": id}, nil
	case "base.Country.Search", "base.Currency.Search":
		table := "base_country"
		if req.Model == "base.Currency" {
			table = "base_currency"
		}
		cond, _ := req.Args[0].(map[string]any)
		and, _ := cond["And"].([]any)
		if len(and) == 0 {
			return []any{}, nil
		}
		tuple, _ := and[0].([]any)
		if len(tuple) < 3 {
			return []any{}, nil
		}
		field, _ := tuple[0].(string)
		value := tuple[2]
		col := strings.ToLower(field)
		if field == "Code" {
			col = "code"
		}
		if field == "Id" {
			col = "id"
		}
		var id string
		if err := db.Table(table).Select("id").Where(col+" = ?", value).Limit(1).Scan(&id).Error; err != nil {
			return nil, err
		}
		if id == "" {
			return []any{}, nil
		}
		return []any{map[string]any{"Id": id}}, nil
	default:
		return nil, fmt.Errorf("unsupported orm call %s.%s", req.Model, req.Method)
	}
}

func (c *gormORMCaller) dbFromContext(ctx context.Context) *gorm.DB {
	if tx, ok := scope.TransactionFromContext(ctx); ok && tx != nil && tx.Session() != nil {
		return tx.Session().DB
	}
	if rs, ok := scope.ScopeFromContext(ctx); ok && rs != nil && rs.Session() != nil {
		return rs.Session().DB
	}
	return c.scope.Session().DB
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

	unique := true
	codeField := &meta.Field{Name: "Code", FieldType: "varchar", NotNull: true, ModelId: countryModel.Id}
	if err := codeField.SetResolvedSpec(&meta.FieldResolvedSpec{
		Structural: meta.FieldStructuralSpec{
			Name:      "Code",
			FieldType: "varchar",
			StorageHints: &meta.FieldStructuralStorageHints{
				Unique: &unique,
			},
		},
	}); err != nil {
		t.Fatalf("code unique spec: %v", err)
	}
	fields := []*meta.Field{
		{Name: "Name", FieldType: "varchar", NotNull: true, ModelId: countryModel.Id},
		codeField,
		{Name: "IsActive", FieldType: "boolean", NotNull: true, ModelId: countryModel.Id},
		{Name: "ZipRequired", FieldType: "boolean", NotNull: true, ModelId: countryModel.Id},
		{Name: "StateRequired", FieldType: "boolean", NotNull: true, ModelId: countryModel.Id},
		{Name: "DefaultCurrencyId", FieldType: "ManyToOne", Relation: "ManyToOne", RelationModel: "base.Currency", ModelId: countryModel.Id},
		{Name: "Code", FieldType: "varchar", NotNull: true, ModelId: currencyModel.Id},
		{Name: "Name", FieldType: "varchar", ModelId: currencyModel.Id},
	}
	currencyCode := fields[6]
	if err := currencyCode.SetResolvedSpec(&meta.FieldResolvedSpec{
		Structural: meta.FieldStructuralSpec{
			Name:         "Code",
			FieldType:    "varchar",
			StorageHints: &meta.FieldStructuralStorageHints{Unique: &unique},
		},
	}); err != nil {
		t.Fatalf("currency code unique: %v", err)
	}
	for _, f := range fields {
		if err := db.Create(f).Error; err != nil {
			t.Fatalf("seed field %s: %v", f.Name, err)
		}
	}
	if err := db.Create(&testCurrencyRow{ID: "cur-cny", Code: "CNY", Name: "CNY", IsActive: true}).Error; err != nil {
		t.Fatalf("seed currency row: %v", err)
	}
}

type testCountryRow struct {
	ID                string    `gorm:"column:id;primaryKey"`
	Name              string    `gorm:"column:name"`
	Code              string    `gorm:"column:code;uniqueIndex"`
	IsActive          bool      `gorm:"column:is_active"`
	ZipRequired       bool      `gorm:"column:zip_required"`
	StateRequired     bool      `gorm:"column:state_required"`
	DefaultCurrencyID string    `gorm:"column:default_currency_id"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
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

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
	"gorm.io/gorm"
)

const partnerImportCompanyID = "cmp-partner-import"

func TestPartnerImport_CompanyScopedUpsert(t *testing.T) {
	runtimeScope := newPartnerImportScope(t)
	path := writePartnerCSV(t, ""+
		"Name,Code,IsActive,CustomerRank,SupplierRank\n"+
		"Import Partner One,IMP-P001,true,1,0\n"+
		"Import Partner Two,IMP-P002,true,0,2\n")
	spec := partnerImportSpec(path, partnerImportCompanyID)

	report, err := runner.Run(withPartnerORMCaller(runtimeScope, partnerImportCompanyID), runtimeScope, spec)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Stats.Ok != 2 {
		t.Fatalf("report stats = %#v", report.Stats)
	}
	if count := countPartners(t, runtimeScope, partnerImportCompanyID); count != 2 {
		t.Fatalf("partner count = %d, want 2", count)
	}
}

func TestPartnerImport_DryRun(t *testing.T) {
	runtimeScope := newPartnerImportScope(t)
	path := writePartnerCSV(t, "Name,Code,IsActive,CustomerRank,SupplierRank\nDry Partner,DRY-P001,true,1,0\n")
	spec := partnerImportSpec(path, partnerImportCompanyID)
	spec.DryRun = true

	report, err := runner.Run(withPartnerORMCaller(runtimeScope, partnerImportCompanyID), runtimeScope, spec)
	if err != nil {
		t.Fatalf("Run dry-run: %v", err)
	}
	if !report.DryRun || report.Stats.Ok != 1 {
		t.Fatalf("report = %#v", report)
	}
	if count := countPartners(t, runtimeScope, partnerImportCompanyID); count != 0 {
		t.Fatalf("partner count = %d, want 0 after dry-run", count)
	}
}

func partnerImportSpec(path, companyID string) importpkg.Spec {
	return importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Model:   "partner.Partner",
		Source: importpkg.Source{
			Format: "csv",
			Path:   path,
		},
		Options: importpkg.Options{
			CompanyID: companyID,
		},
	}
}

func writePartnerCSV(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "partners.csv")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}
	return path
}

func newPartnerImportScope(t *testing.T) scope.Scope {
	t.Helper()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "partner-import.db"),
		},
	}
	runtimeScope := defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	seedPartnerImportSchema(t, runtimeScope.Session().DB)
	return runtimeScope
}

func seedPartnerImportSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&meta.Module{}, &meta.Model{}, &meta.Field{}, &testPartnerRow{}, &modmeta.ModelData{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	partnerModel := &meta.Model{
		Name:         "Partner",
		Application:  "partner",
		Path:         "/tmp",
		ModelTable:   "partner_partner",
		CompanyField: strPtr("CompanyId"),
	}
	if err := db.Create(partnerModel).Error; err != nil {
		t.Fatalf("seed partner model: %v", err)
	}

	codeField := &meta.Field{Name: "Code", FieldType: "varchar", NotNull: true, ModelId: partnerModel.Id}
	if err := codeField.SetResolvedSpec(&meta.FieldResolvedSpec{
		Structural: meta.FieldStructuralSpec{
			Name:      "Code",
			FieldType: "varchar",
			StorageHints: &meta.FieldStructuralStorageHints{
				UniqueIndex: strPtr("uidx_partner_company_code"),
			},
		},
	}); err != nil {
		t.Fatalf("code unique spec: %v", err)
	}
	companyField := &meta.Field{Name: "CompanyId", FieldType: "ManyToOneRef", NotNull: true, ModelId: partnerModel.Id}
	if err := companyField.SetResolvedSpec(&meta.FieldResolvedSpec{
		Structural: meta.FieldStructuralSpec{
			Name:      "CompanyId",
			FieldType: "ManyToOneRef",
			StorageHints: &meta.FieldStructuralStorageHints{
				UniqueIndex: strPtr("uidx_partner_company_code"),
			},
		},
	}); err != nil {
		t.Fatalf("company unique spec: %v", err)
	}
	fields := []*meta.Field{
		{Name: "Name", FieldType: "varchar", NotNull: true, ModelId: partnerModel.Id},
		codeField,
		companyField,
		{Name: "IsActive", FieldType: "boolean", NotNull: true, ModelId: partnerModel.Id},
		{Name: "CustomerRank", FieldType: "int", NotNull: true, ModelId: partnerModel.Id},
		{Name: "SupplierRank", FieldType: "int", NotNull: true, ModelId: partnerModel.Id},
	}
	for _, f := range fields {
		if err := db.Create(f).Error; err != nil {
			t.Fatalf("seed field %s: %v", f.Name, err)
		}
	}
}

func strPtr(s string) *string { return &s }

type testPartnerRow struct {
	ID           string    `gorm:"column:id;primaryKey"`
	Name         string    `gorm:"column:name"`
	Code         string    `gorm:"column:code"`
	CompanyID    string    `gorm:"column:company_id"`
	IsActive     bool      `gorm:"column:is_active"`
	CustomerRank int       `gorm:"column:customer_rank"`
	SupplierRank int       `gorm:"column:supplier_rank"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (testPartnerRow) TableName() string { return "partner_partner" }

func countPartners(t *testing.T, runtimeScope scope.Scope, companyID string) int {
	t.Helper()
	var count int64
	if err := runtimeScope.Session().DB.Table("partner_partner").Where("company_id = ?", companyID).Count(&count).Error; err != nil {
		t.Fatalf("count partners: %v", err)
	}
	return int(count)
}

type partnerORMCaller struct {
	scope     scope.Scope
	companyID string
}

func withPartnerORMCaller(runtimeScope scope.Scope, companyID string) context.Context {
	return importcaller.ContextWithCaller(context.Background(), partnerORMCaller{scope: runtimeScope, companyID: companyID})
}

func (c partnerORMCaller) Call(ctx context.Context, req importcaller.CallRequest) (any, error) {
	db := c.dbFromContext(ctx)
	switch req.Model + "." + req.Method {
	case "partner.Partner.Create":
		vals, _ := req.Args[0].(map[string]any)
		id := "pt-" + strings.ToLower(strings.TrimSpace(vals["Code"].(string)))
		row := map[string]any{
			"id":            id,
			"name":          vals["Name"],
			"code":          strings.ToUpper(strings.TrimSpace(vals["Code"].(string))),
			"company_id":    vals["CompanyId"],
			"is_active":     vals["IsActive"],
			"customer_rank": vals["CustomerRank"],
			"supplier_rank": vals["SupplierRank"],
			"created_at":    time.Now().UTC(),
			"updated_at":    time.Now().UTC(),
		}
		if err := db.Table("partner_partner").Create(row).Error; err != nil {
			return nil, err
		}
		return map[string]any{"Id": id}, nil
	case "partner.Partner.UpdateById":
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
			case "CustomerRank":
				updates["customer_rank"] = v
			case "SupplierRank":
				updates["supplier_rank"] = v
			default:
				col := strings.ToLower(k[:1]) + k[1:]
				updates[col] = v
			}
		}
		if err := db.Table("partner_partner").Where("id = ?", id).Updates(updates).Error; err != nil {
			return nil, err
		}
		return map[string]any{"Id": id}, nil
	case "partner.Partner.Search":
		cond, _ := req.Args[0].(map[string]any)
		and, _ := cond["And"].([]any)
		query := db.Table("partner_partner")
		for _, clause := range and {
			tuple, _ := clause.([]any)
			if len(tuple) < 3 {
				continue
			}
			field, _ := tuple[0].(string)
			col := strings.ToLower(field[:1]) + field[1:]
			if field == "CompanyId" {
				col = "company_id"
			}
			if field == "Code" && tuple[1] == "in" {
				query = query.Where("code IN ?", tuple[2])
				continue
			}
			query = query.Where(col+" = ?", tuple[2])
		}
		var id string
		if err := query.Select("id").Limit(1).Scan(&id).Error; err != nil {
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

func (c partnerORMCaller) dbFromContext(ctx context.Context) *gorm.DB {
	if tx, ok := scope.TransactionFromContext(ctx); ok && tx != nil && tx.Session() != nil {
		return tx.Session().DB
	}
	if rs, ok := scope.ScopeFromContext(ctx); ok && rs != nil && rs.Session() != nil {
		return rs.Session().DB
	}
	return c.scope.Session().DB
}

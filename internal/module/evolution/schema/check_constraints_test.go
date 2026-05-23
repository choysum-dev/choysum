// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"testing"

	"gorm.io/gorm"
)

func TestNormalizeCheckExprAndEscapeSQLServerLiteral(t *testing.T) {
	if got := normalizeCheckExpr("  ` status IN ('draft', 'done') `  "); got != "(status IN ('draft', 'done'))" {
		t.Fatalf("normalizeCheckExpr() = %q", got)
	}
	if got := normalizeCheckExpr("  ' amount > 0 '  "); got != "(amount > 0)" {
		t.Fatalf("normalizeCheckExpr() stripped quotes = %q", got)
	}
	if got := normalizeCheckExpr("\namount > 0\nAND amount < 10\n"); got != "(amount > 0 AND amount < 10)" {
		t.Fatalf("normalizeCheckExpr() collapsed = %q", got)
	}
	if got := normalizeCheckExpr("(score >= 0)"); got != "(score >= 0)" {
		t.Fatalf("normalizeCheckExpr() wrapped existing parens = %q", got)
	}
	if got := normalizeCheckExpr("   "); got != "" {
		t.Fatalf("normalizeCheckExpr(blank) = %q", got)
	}
	if got := escapeSQLServerStringLiteral("a'b'c"); got != "a''b''c" {
		t.Fatalf("escapeSQLServerStringLiteral() = %q", got)
	}
}

func TestCheckConstraintRuntimeHelpers(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB

	if err := ensureCheckConstraint(db, "sqlite", "sales_order", "chk_sales_order_status", "status <> ''"); err != nil {
		t.Fatalf("ensureCheckConstraint(sqlite) error = %v", err)
	}
	if err := ensureCheckConstraint(db, "unknown", "sales_order", "chk_sales_order_status", "status <> ''"); err != nil {
		t.Fatalf("ensureCheckConstraint(unknown) error = %v", err)
	}
	if err := dropCheckConstraintBestEffort(db, "sqlite", "sales_order", "chk_sales_order_status"); err != nil {
		t.Fatalf("dropCheckConstraintBestEffort(sqlite) error = %v", err)
	}
	if err := dropCheckConstraintBestEffort(db, "unknown", "sales_order", "chk_sales_order_status"); err != nil {
		t.Fatalf("dropCheckConstraintBestEffort(unknown) error = %v", err)
	}

	if err := ensurePostgresCheckConstraint(db, "sales_order", "chk_sales_order_status", "status <> ''"); err == nil {
		t.Fatal("expected postgres DDL to fail on sqlite")
	}
	if err := ensureMySQLCheckConstraint(db, "sales_order", "chk_sales_order_status", "status <> ''"); err == nil {
		t.Fatal("expected mysql DDL to fail on sqlite")
	}
	if err := ensureSQLServerCheckConstraint(db, "sales_order", "chk_sales_order_status", "status <> ''"); err == nil {
		t.Fatal("expected sqlserver DDL to fail on sqlite")
	}
	if err := dropCheckConstraintBestEffort(db, "postgres", "sales_order", "chk_sales_order_status"); err == nil {
		t.Fatal("expected postgres drop constraint to fail on sqlite")
	}
	if err := dropCheckConstraintBestEffort(db, "mysql", "sales_order", "chk_sales_order_status"); err == nil {
		t.Fatal("expected mysql drop constraint to fail on sqlite")
	}
	if err := dropCheckConstraintBestEffort(db, "sqlserver", "sales_order", "chk_sales_order_status"); err == nil {
		t.Fatal("expected sqlserver drop constraint to fail on sqlite")
	}
}

func TestCheckConstraintHelpersIgnoreBlankExpressions(t *testing.T) {
	runtimeScope := newSchemaTestScope(t)
	db := runtimeScope.Session().DB

	for _, tc := range []struct {
		name    string
		dialect string
		fn      func(*gorm.DB, string, string, string) error
	}{
		{name: "postgres blank", dialect: "postgres", fn: ensurePostgresCheckConstraint},
		{name: "mysql blank", dialect: "mysql", fn: ensureMySQLCheckConstraint},
		{name: "sqlserver blank", dialect: "sqlserver", fn: ensureSQLServerCheckConstraint},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(db, "sales_order", "chk_sales_order_status", "   "); err != nil {
				t.Fatalf("blank expr error = %v", err)
			}
		})
	}

	for _, dialect := range []string{"postgres", "mysql", "sqlserver"} {
		t.Run("dispatch "+dialect+" blank", func(t *testing.T) {
			if err := ensureCheckConstraint(db, dialect, "sales_order", "chk_sales_order_status", "   "); err != nil {
				t.Fatalf("ensureCheckConstraint(%s, blank) error = %v", dialect, err)
			}
		})
	}
}

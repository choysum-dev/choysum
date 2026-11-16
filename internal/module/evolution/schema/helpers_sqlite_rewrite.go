// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// sqliteTableRewrite describes a private whole-table rewrite used when single-column
// ALTER is insufficient (ReplaceColumn / SetNotNull fallbacks). Not a public API.
type sqliteTableRewrite struct {
	Rename       map[string]string   // old physical name → new
	Drop         map[string]struct{} // columns to omit
	ForceNotNull map[string]struct{} // columns that must be NOT NULL in the new table
}

func isSQLiteDialect(dialect string) bool {
	return strings.EqualFold(strings.TrimSpace(dialect), "sqlite")
}

// sqliteRewriteTable rebuilds table under rewrite rules. Caller must use sqlite dialect.
func sqliteRewriteTable(db *gorm.DB, table string, rewrite sqliteTableRewrite) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	table = strings.TrimSpace(table)
	if table == "" {
		return fmt.Errorf("table is required")
	}
	types, err := getColumnTypes(db, table)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", table, err)
	}
	if len(types) == 0 {
		return fmt.Errorf("table %s has no columns", table)
	}

	type colDef struct {
		srcName string
		dstName string
		decl    string
	}
	var cols []colDef
	var selectParts []string
	for _, ct := range types {
		if ct == nil {
			continue
		}
		src := strings.TrimSpace(ct.Name())
		if src == "" {
			continue
		}
		if rewrite.Drop != nil {
			if _, ok := rewrite.Drop[strings.ToLower(src)]; ok {
				continue
			}
		}
		dst := src
		if rewrite.Rename != nil {
			if to, ok := rewrite.Rename[strings.ToLower(src)]; ok && strings.TrimSpace(to) != "" {
				dst = strings.TrimSpace(to)
			}
		}
		typeName := strings.TrimSpace(ct.DatabaseTypeName())
		if typeName == "" {
			typeName = "TEXT"
		}
		notNull := false
		if n, ok := ct.Nullable(); ok && !n {
			notNull = true
		}
		if pk, ok := ct.PrimaryKey(); ok && pk {
			notNull = true
		}
		if rewrite.ForceNotNull != nil {
			if _, ok := rewrite.ForceNotNull[strings.ToLower(src)]; ok {
				notNull = true
			}
			if _, ok := rewrite.ForceNotNull[strings.ToLower(dst)]; ok {
				notNull = true
			}
		}
		decl := quoteIdent("sqlite", dst) + " " + typeName
		if notNull {
			decl += " NOT NULL"
		}
		cols = append(cols, colDef{srcName: src, dstName: dst, decl: decl})
		selectParts = append(selectParts, quoteIdent("sqlite", src))
	}
	if len(cols) == 0 {
		return fmt.Errorf("rewrite of %s would leave no columns", table)
	}

	tmp := "choysum_rw_" + strings.ReplaceAll(table, `"`, "")
	// Keep temp name short / safe.
	if len(tmp) > 60 {
		tmp = tmp[:60]
	}
	qTable := quoteIdent("sqlite", table)
	qTmp := quoteIdent("sqlite", tmp)

	var decls []string
	for _, c := range cols {
		decls = append(decls, c.decl)
	}
	createSQL := fmt.Sprintf("CREATE TABLE %s (%s)", qTmp, strings.Join(decls, ", "))
	if err := helperExec(db, createSQL); err != nil {
		return fmt.Errorf("create rewrite table %s: %w", tmp, err)
	}
	insertSQL := fmt.Sprintf("INSERT INTO %s SELECT %s FROM %s", qTmp, strings.Join(selectParts, ", "), qTable)
	if err := helperExec(db, insertSQL); err != nil {
		_ = helperExec(db, "DROP TABLE IF EXISTS "+qTmp)
		return fmt.Errorf("copy into rewrite table %s: %w", tmp, err)
	}
	if err := helperExec(db, "DROP TABLE "+qTable); err != nil {
		_ = helperExec(db, "DROP TABLE IF EXISTS "+qTmp)
		return fmt.Errorf("drop original %s: %w", table, err)
	}
	if err := helperExec(db, fmt.Sprintf("ALTER TABLE %s RENAME TO %s", qTmp, qTable)); err != nil {
		return fmt.Errorf("rename rewrite table to %s: %w", table, err)
	}
	return nil
}

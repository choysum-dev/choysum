// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"crypto/sha1"
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
// Runs create/copy/drop/rename in one transaction. Temporarily disables foreign_keys when
// needed so DROP TABLE cannot cascade-delete child rows; restores indexes/triggers that
// still apply after rename/drop.
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

	indexSQLs, err := sqliteListMasterSQLs(db, "index", table)
	if err != nil {
		return err
	}
	triggerSQLs, err := sqliteListMasterSQLs(db, "trigger", table)
	if err != nil {
		return err
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
		decl := sqliteColumnDecl(ct, dst, rewrite)
		cols = append(cols, colDef{srcName: src, dstName: dst, decl: decl})
		selectParts = append(selectParts, quoteIdent("sqlite", src))
	}
	if len(cols) == 0 {
		return fmt.Errorf("rewrite of %s would leave no columns", table)
	}

	tmp := sqliteRewriteTempName(table)
	qTable := quoteIdent("sqlite", table)
	qTmp := quoteIdent("sqlite", tmp)

	var decls []string
	for _, c := range cols {
		decls = append(decls, c.decl)
	}
	createSQL := fmt.Sprintf("CREATE TABLE %s (%s)", qTmp, strings.Join(decls, ", "))
	insertSQL := fmt.Sprintf("INSERT INTO %s SELECT %s FROM %s", qTmp, strings.Join(selectParts, ", "), qTable)

	// One connection for FK pragma + DDL. Reuse an open tx connection (upgrade
	// scripts often run inside Transactor.Required); otherwise pin via Connection.
	return sqliteRunOnOneConn(db, func(connDB *gorm.DB) (err error) {
		fkWasOn, fkErr := sqliteReadForeignKeys(connDB)
		if fkErr != nil {
			return fkErr
		}
		fkDisabled := false
		if fkWasOn {
			if offErr := helperExec(connDB, "PRAGMA foreign_keys=OFF"); offErr != nil {
				return fmt.Errorf("disable foreign_keys for rewrite: %w", offErr)
			}
			stillOn, stillErr := sqliteReadForeignKeys(connDB)
			if stillErr != nil {
				return stillErr
			}
			if stillOn {
				// PRAGMA foreign_keys is a no-op inside an open transaction; refuse when
				// DROP could cascade to referencing tables.
				referenced, refErr := sqliteTableIsReferenced(connDB, table)
				if refErr != nil {
					return refErr
				}
				if referenced {
					return fmt.Errorf("sqlite rewrite: foreign_keys still ON (pragma ignored inside a transaction) while other tables reference %s", table)
				}
			} else {
				fkDisabled = true
			}
		}
		// Restore before this callback returns so the pooled conn is never released with FK off.
		defer func() {
			if !fkDisabled {
				return
			}
			if restoreErr := helperExec(connDB, "PRAGMA foreign_keys=ON"); restoreErr != nil && err == nil {
				err = fmt.Errorf("restore foreign_keys after rewrite: %w", restoreErr)
			}
		}()

		txErr := connDB.Transaction(func(tx *gorm.DB) error {
			_ = helperExec(tx, "DROP TABLE IF EXISTS "+qTmp)
			if err := helperExec(tx, createSQL); err != nil {
				return fmt.Errorf("create rewrite table %s: %w", tmp, err)
			}
			if err := helperExec(tx, insertSQL); err != nil {
				_ = helperExec(tx, "DROP TABLE IF EXISTS "+qTmp)
				return fmt.Errorf("copy into rewrite table %s: %w", tmp, err)
			}
			if err := helperExec(tx, "DROP TABLE "+qTable); err != nil {
				_ = helperExec(tx, "DROP TABLE IF EXISTS "+qTmp)
				return fmt.Errorf("drop original %s: %w", table, err)
			}
			if err := helperExec(tx, fmt.Sprintf("ALTER TABLE %s RENAME TO %s", qTmp, qTable)); err != nil {
				return fmt.Errorf("rename rewrite table to %s: %w", table, err)
			}
			for _, sql := range indexSQLs {
				rewritten, ok := sqliteRewriteObjectSQL(sql, rewrite)
				if !ok {
					continue
				}
				if err := helperExec(tx, rewritten); err != nil {
					return fmt.Errorf("restore index after rewrite of %s: %w", table, err)
				}
			}
			for _, sql := range triggerSQLs {
				rewritten, ok := sqliteRewriteObjectSQL(sql, rewrite)
				if !ok {
					continue
				}
				if err := helperExec(tx, rewritten); err != nil {
					return fmt.Errorf("restore trigger after rewrite of %s: %w", table, err)
				}
			}
			if err := sqliteCheckForeignKeys(tx); err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			_ = helperExec(connDB, "DROP TABLE IF EXISTS "+qTmp)
			return txErr
		}
		return nil
	})
}

// sqliteRunOnOneConn runs fn on a single SQLite connection. If db is already bound
// to an open transaction, reuses it (avoids "database is locked" from a second pool
// conn). Otherwise pins a pool connection for the duration of fn.
func sqliteRunOnOneConn(db *gorm.DB, fn func(*gorm.DB) error) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if db.Statement != nil {
		if tx, ok := db.Statement.ConnPool.(gorm.TxCommitter); ok && tx != nil {
			return fn(db)
		}
	}
	return db.Connection(fn)
}

func sqliteColumnDecl(ct gorm.ColumnType, dst string, rewrite sqliteTableRewrite) string {
	typeName := sqliteBareTypeName(ct.DatabaseTypeName())
	isPK := false
	if pk, ok := ct.PrimaryKey(); ok && pk {
		isPK = true
	}
	notNull := isPK
	if n, ok := ct.Nullable(); ok && !n {
		notNull = true
	}
	src := strings.TrimSpace(ct.Name())
	if rewrite.ForceNotNull != nil {
		if _, ok := rewrite.ForceNotNull[strings.ToLower(src)]; ok {
			notNull = true
		}
		if _, ok := rewrite.ForceNotNull[strings.ToLower(dst)]; ok {
			notNull = true
		}
	}
	decl := quoteIdent("sqlite", dst) + " " + typeName
	if isPK {
		decl += " PRIMARY KEY"
		// SQLite only allows AUTOINCREMENT on INTEGER PRIMARY KEY.
		if ai, ok := ct.AutoIncrement(); ok && ai && strings.EqualFold(typeName, "INTEGER") {
			decl += " AUTOINCREMENT"
		}
	} else if notNull {
		decl += " NOT NULL"
	}
	if def, ok := ct.DefaultValue(); ok {
		def = strings.TrimSpace(def)
		if def != "" {
			decl += " DEFAULT " + def
		}
	}
	return decl
}

// sqliteBareTypeName keeps only the type token so we do not duplicate PRIMARY KEY /
// NOT NULL / DEFAULT when a driver embeds constraints in DatabaseTypeName().
func sqliteBareTypeName(raw string) string {
	typeName := strings.TrimSpace(raw)
	if typeName == "" {
		return "TEXT"
	}
	upper := strings.ToUpper(typeName)
	cutAt := len(typeName)
	for _, kw := range []string{"PRIMARY KEY", "AUTOINCREMENT", "NOT NULL", "DEFAULT"} {
		i := strings.Index(upper, kw)
		if i < 0 {
			continue
		}
		if i > 0 && upper[i-1] != ' ' && upper[i-1] != '\t' {
			continue
		}
		if i < cutAt {
			cutAt = i
		}
	}
	typeName = strings.TrimSpace(typeName[:cutAt])
	if typeName == "" {
		return "TEXT"
	}
	return typeName
}

// Overridable seams for error-path tests.
var (
	sqliteListMasterSQLs   = sqliteMasterSQLs
	sqliteReadForeignKeys  = sqliteForeignKeysEnabled
	sqliteCheckForeignKeys = sqliteForeignKeyCheck
)

func sqliteRewriteTempName(table string) string {
	// Deterministic per table so DROP IF EXISTS reclaims leftovers from a crashed run.
	sum := sha1.Sum([]byte(strings.TrimSpace(table)))
	return fmt.Sprintf("choysum_rw_%x", sum[:10])
}

// sqliteTableIsReferenced reports whether any other table has a foreign key to table.
func sqliteTableIsReferenced(db *gorm.DB, table string) (bool, error) {
	var names []string
	if err := db.Raw(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).Scan(&names).Error; err != nil {
		return false, fmt.Errorf("list tables for fk refs: %w", err)
	}
	want := strings.ToLower(strings.TrimSpace(table))
	for _, name := range names {
		if strings.EqualFold(strings.TrimSpace(name), table) {
			continue
		}
		var rows []struct {
			Table string
		}
		q := fmt.Sprintf("PRAGMA foreign_key_list(%s)", quoteIdent("sqlite", name))
		if err := db.Raw(q).Scan(&rows).Error; err != nil {
			return false, fmt.Errorf("foreign_key_list %s: %w", name, err)
		}
		for _, row := range rows {
			if strings.ToLower(strings.TrimSpace(row.Table)) == want {
				return true, nil
			}
		}
	}
	return false, nil
}

func sqliteMasterSQLs(db *gorm.DB, kind, table string) ([]string, error) {
	var rows []string
	if err := db.Raw(
		`SELECT sql FROM sqlite_master WHERE type = ? AND tbl_name = ? AND sql IS NOT NULL`,
		kind, table,
	).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list %s for %s: %w", kind, table, err)
	}
	return rows, nil
}

func sqliteForeignKeysEnabled(db *gorm.DB) (bool, error) {
	var on int
	if err := db.Raw("PRAGMA foreign_keys").Scan(&on).Error; err != nil {
		return false, fmt.Errorf("read foreign_keys pragma: %w", err)
	}
	return on != 0, nil
}

func sqliteForeignKeyCheck(db *gorm.DB) error {
	var violations []struct {
		Table  string
		Rowid  int64
		Parent string
		Fkid   int64
	}
	if err := db.Raw("PRAGMA foreign_key_check").Scan(&violations).Error; err != nil {
		return fmt.Errorf("foreign_key_check: %w", err)
	}
	if len(violations) > 0 {
		return fmt.Errorf("foreign_key_check failed after rewrite (%d violation(s))", len(violations))
	}
	return nil
}

// sqliteRewriteObjectSQL adjusts CREATE INDEX/TRIGGER SQL for dropped/renamed columns.
// Returns ok=false when the object references a dropped column.
func sqliteRewriteObjectSQL(sql string, rewrite sqliteTableRewrite) (string, bool) {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return "", false
	}
	lower := strings.ToLower(sql)
	if rewrite.Drop != nil {
		// Match columns in the definition body (after first '('), not the object name
		// (e.g. idx_gone_created must not be skipped when dropping column "gone").
		body := lower
		if i := strings.Index(lower, "("); i >= 0 {
			body = lower[i:]
		} else if i := strings.Index(lower, " of "); i >= 0 {
			body = lower[i:]
		}
		for col := range rewrite.Drop {
			if sqliteSQLMentionsColumn(body, strings.ToLower(col)) {
				return "", false
			}
		}
	}
	out := sql
	if rewrite.Rename != nil {
		for from, to := range rewrite.Rename {
			out = sqliteReplaceIdent(out, from, to)
		}
	}
	return out, true
}

func sqliteSQLMentionsColumn(lowerSQL, colLower string) bool {
	if colLower == "" {
		return false
	}
	// Match quoted or bare identifiers without treating substrings as hits.
	patterns := []string{`"` + colLower + `"`, "`" + colLower + "`", "[" + colLower + "]"}
	for _, p := range patterns {
		if strings.Contains(lowerSQL, p) {
			return true
		}
	}
	for i := 0; i < len(lowerSQL); {
		idx := strings.Index(lowerSQL[i:], colLower)
		if idx < 0 {
			return false
		}
		idx += i
		beforeOK := idx == 0 || !isIdentByte(lowerSQL[idx-1])
		after := idx + len(colLower)
		afterOK := after >= len(lowerSQL) || !isIdentByte(lowerSQL[after])
		if beforeOK && afterOK {
			return true
		}
		i = idx + len(colLower)
	}
	return false
}

func sqliteReplaceIdent(sql, from, to string) string {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" || strings.EqualFold(from, to) {
		return sql
	}
	replacer := strings.NewReplacer(
		`"`+from+`"`, `"`+to+`"`,
		`"`+strings.ToLower(from)+`"`, `"`+to+`"`,
		`"`+strings.ToUpper(from)+`"`, `"`+to+`"`,
	)
	out := replacer.Replace(sql)
	// Bare identifier tokens (case-insensitive).
	var b strings.Builder
	lower := strings.ToLower(out)
	fromLower := strings.ToLower(from)
	i := 0
	for i < len(out) {
		idx := strings.Index(lower[i:], fromLower)
		if idx < 0 {
			b.WriteString(out[i:])
			break
		}
		idx += i
		beforeOK := idx == 0 || !isIdentByte(out[idx-1])
		after := idx + len(from)
		afterOK := after >= len(out) || !isIdentByte(out[after])
		b.WriteString(out[i:idx])
		if beforeOK && afterOK {
			b.WriteString(to)
		} else {
			b.WriteString(out[idx:after])
		}
		i = after
	}
	return b.String()
}

func isIdentByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

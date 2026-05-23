// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import "fmt"

type PostgresForeignKeyBuilder struct{}

func (b *PostgresForeignKeyBuilder) BuildForeignKeySQL(fk ForeignKeyInfo) string {
	// Quote identifiers with double quotes to avoid reserved keyword conflicts.
	sql := fmt.Sprintf(
		`ALTER TABLE "%s" ADD CONSTRAINT "fk_%s_%s" FOREIGN KEY ("%s") REFERENCES "%s"("%s")`,
		fk.TableName,
		fk.TableName,
		fk.ColumnName,
		fk.ColumnName,
		fk.ReferTableName,
		fk.ReferColumnName,
	)

	if fk.OnDelete != "" {
		sql += " ON DELETE " + fk.OnDelete
	}
	if fk.OnUpdate != "" {
		sql += " ON UPDATE " + fk.OnUpdate
	}

	return sql
}

func (b *PostgresForeignKeyBuilder) BuildDropForeignKeySQL(fk ForeignKeyInfo) string {
	// Quote identifiers with double quotes.
	return fmt.Sprintf(
		`ALTER TABLE "%s" DROP CONSTRAINT IF EXISTS "fk_%s_%s"`,
		fk.TableName,
		fk.TableName,
		fk.ColumnName,
	)
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import "fmt"

type MySQLForeignKeyBuilder struct{}

func (b *MySQLForeignKeyBuilder) BuildForeignKeySQL(fk ForeignKeyInfo) string {
	// Quote identifiers with backticks to avoid reserved keyword conflicts.
	sql := fmt.Sprintf(
		"ALTER TABLE `%s` ADD CONSTRAINT `fk_%s_%s` FOREIGN KEY (`%s`) REFERENCES `%s`(`%s`)",
		fk.TableName,
		fk.TableName,
		fk.ColumnName,
		fk.ColumnName,
		fk.ReferTableName,
		fk.ReferColumnName,
	)

	if fk.OnDelete != "" {
		sql += fmt.Sprintf(" ON DELETE %s", fk.OnDelete)
	}
	if fk.OnUpdate != "" {
		sql += fmt.Sprintf(" ON UPDATE %s", fk.OnUpdate)
	}

	return sql
}

func (b *MySQLForeignKeyBuilder) BuildDropForeignKeySQL(fk ForeignKeyInfo) string {
	// Quote identifiers with backticks.
	return fmt.Sprintf(
		"ALTER TABLE `%s` DROP FOREIGN KEY IF EXISTS `fk_%s_%s`",
		fk.TableName,
		fk.TableName,
		fk.ColumnName,
	)
}

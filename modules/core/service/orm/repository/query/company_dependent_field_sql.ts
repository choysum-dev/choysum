// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { sql } from 'kysely';
import type { DialectName } from '../repository_dialect';
import { resolveCompanyDependentCompanyId } from '../projection/company_dependent_field_codec';
import { repositoryPredicateRef, type RepositoryPredicateBuilder } from './predicate_builder_adapter';

function quoteSqlString(value: string): string {
  return `'${String(value).replace(/'/g, "''")}'`;
}

/**
 * Build a SQL expression that unwraps a companyDependent JSON/JSONB map
 * for the active company (no fallback — F0).
 *
 * When no active company is available, returns a NULL SQL expression.
 */
export function buildCompanyDependentFieldUnwrapExpr(
  dialect: DialectName | string,
  eb: RepositoryPredicateBuilder,
  tableQualifiedColumn: string,
  companyId?: string
): unknown {
  const resolved = String(companyId ?? resolveCompanyDependentCompanyId() ?? '').trim();
  const col = repositoryPredicateRef(eb, tableQualifiedColumn);
  if (!resolved) {
    return sql`NULL`;
  }
  const d = String(dialect || 'postgres').toLowerCase() as DialectName | string;
  const keyLit = sql.raw(quoteSqlString(resolved));

  switch (d) {
    case 'postgres':
    case 'postgresql':
      return sql`(${col}->>${keyLit})`;
    case 'mysql':
    case 'mariadb': {
      const path = sql.raw(quoteSqlString(`$.${resolved}`));
      return sql`JSON_UNQUOTE(JSON_EXTRACT(${col}, ${path}))`;
    }
    case 'sqlite': {
      const path = sql.raw(quoteSqlString(`$.${resolved}`));
      return sql`json_extract(${col}, ${path})`;
    }
    case 'mssql':
    case 'sqlserver': {
      const path = sql.raw(quoteSqlString(`$.${resolved}`));
      return sql`JSON_VALUE(${col}, ${path})`;
    }
    default:
      return sql`(${col}->>${keyLit})`;
  }
}

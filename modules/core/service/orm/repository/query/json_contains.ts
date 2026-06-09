// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { sql } from 'kysely';
import type { DialectName } from '../repository_dialect';

type RepositoryContainsExpressionBuilder = {
  (lhs: unknown, operator: unknown, rhs: unknown): unknown;
  fn(name: string, args: unknown[]): unknown;
};

function asContainsExpressionBuilder(input: unknown): RepositoryContainsExpressionBuilder {
  if (typeof input !== 'function') {
    throw new Error('contains expression builder is invalid');
  }

  const record = input as { fn?: unknown };
  if (typeof record.fn !== 'function') {
    throw new Error('contains expression builder fn is invalid');
  }

  return input as RepositoryContainsExpressionBuilder;
}

export function buildContainsExpression(dialect: DialectName, eb: unknown, lhsExpr: unknown, rhs: unknown, selfTable?: string, fieldName?: string): unknown {
  const builder = asContainsExpressionBuilder(eb);
  const rhsJson = typeof rhs === 'string' ? rhs : JSON.stringify(rhs);
  const escapedJson = rhsJson.replace(/'/g, "''");

  switch (dialect) {
    case 'postgres':
      return builder(lhsExpr, '@>', sql.raw(`'${escapedJson}'::jsonb`));

    case 'mysql':
      return builder.fn('JSON_CONTAINS', [lhsExpr, sql.raw(`'${escapedJson}'`)]);

    case 'sqlite': {
      if (!selfTable || !fieldName) {
        throw new Error('SQLite contains requires selfTable and fieldName arguments');
      }

      const targetValue = typeof rhs === 'string' ? `"${rhs.replace(/"/g, '""')}"` : JSON.stringify(rhs);
      const escapedTarget = targetValue.replace(/'/g, "''");

      return sql.raw(`EXISTS(SELECT 1 FROM json_each(${selfTable}.${fieldName}) WHERE value = '${escapedTarget}')`);
    }

    case 'mssql':
      throw new Error('contains is not supported for the MSSQL dialect yet');

    default:
      throw new Error(`contains does not support database dialect: ${dialect}`);
  }
}

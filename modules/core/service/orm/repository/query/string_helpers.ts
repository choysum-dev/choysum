// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { sql, Expression, ExpressionWrapper } from 'kysely';
import type { DialectName } from '../repository_dialect';
import type { ObjectRecord } from '../../../../utils/types';

type StringExpression = Expression<unknown> | ExpressionWrapper<ObjectRecord, string, unknown>;

type StringFnApi = ((name: string, args: ReadonlyArray<unknown>) => StringExpression) & {
  coalesce: (expression: unknown, fallback: unknown) => StringExpression;
};

type StringQueryBuilder = {
  fn: StringFnApi;
};

function quoteSqlString(value: string) {
  return value.replace(/'/g, "''");
}

export function getStringHelpers(dialect: DialectName, qb: unknown) {
  const queryBuilder = qb as StringQueryBuilder;
  const fn = queryBuilder.fn;
  const isPostgres = dialect === 'postgres';
  const isMssql = dialect === 'mssql';

  const emptyLit = isPostgres ? sql.raw(`''::text`) : isMssql ? sql.raw(`N''`) : sql.raw(`''`);

  const strLit = (value: string) => {
    const quoted = quoteSqlString(value);
    if (isPostgres) return sql.raw(`'${quoted}'::text`);
    if (isMssql) return sql.raw(`N'${quoted}'`);
    return sql.raw(`'${quoted}'`);
  };

  const coalesceText = (expression: StringExpression) => fn.coalesce(expression, emptyLit);

  const concat = (...parts: Array<StringExpression | string>): StringExpression => {
    if (dialect === 'sqlite') {
      const expressions = parts.map(part => (typeof part === 'string' ? strLit(part) : sql`coalesce(${part}, ${emptyLit})`));
      return sql.join(expressions, sql` || `);
    }

    if (isMssql) {
      const args = parts.map(part => (typeof part === 'string' ? strLit(part) : coalesceText(part)));
      return fn('concat', args);
    }

    const args = parts.map(part => (typeof part === 'string' ? strLit(part) : coalesceText(part)));
    return fn('concat_ws', [isPostgres ? sql.raw(`''::text`) : sql.raw(`''`), ...args]);
  };

  const concatWs = (separator: string, ...parts: Array<StringExpression | string>): StringExpression => {
    if (dialect === 'sqlite') {
      const separatorLiteral = strLit(separator);
      const expressions = parts.map(part => (typeof part === 'string' ? strLit(part) : sql`coalesce(${part}, ${emptyLit})`));
      return sql.join(expressions, sql` || ${separatorLiteral} || `);
    }

    const separatorLiteral = strLit(separator);
    const args = parts.map(part => (typeof part === 'string' ? strLit(part) : coalesceText(part)));

    if (isMssql) {
      return fn('concat_ws', [separatorLiteral, ...args]);
    }

    return fn('concat_ws', [separatorLiteral, ...args]);
  };

  const lower = (expression: StringExpression) => fn('lower', [expression]);

  return { concat, concatWs, lower };
}

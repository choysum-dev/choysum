// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { sql } from 'kysely';
import { buildTimeBucketExpr } from '../time_bucket_sql';

function renderOperationNode(expr: any): string {
  return JSON.stringify((expr as any).toOperationNode());
}

test('repository time bucket sql builds postgres truncation with timezone application', () => {
  const expr = buildTimeBucketExpr('postgres', sql.ref('demo.CreatedAt'), 'month', 'Asia/Shanghai');
  const rendered = renderOperationNode(expr);

  expect(rendered.includes('DATE_TRUNC')).toBe(true);
  expect(rendered.includes('AT TIME ZONE')).toBe(true);
  expect(rendered.includes('Asia/Shanghai')).toBe(true);
});

test('repository time bucket sql builds mysql and sqlite week expressions with dialect-specific helpers', () => {
  const mysqlExpr = buildTimeBucketExpr('mysql', sql.ref('demo.CreatedAt'), 'week', 'UTC');
  const sqliteExpr = buildTimeBucketExpr('sqlite', sql.ref('demo.CreatedAt'), 'week');

  expect(renderOperationNode(mysqlExpr).includes('WEEKDAY')).toBe(true);
  expect(renderOperationNode(mysqlExpr).includes('CONVERT_TZ')).toBe(true);
  expect(renderOperationNode(sqliteExpr).includes('weekday 1')).toBe(true);
});

test('repository time bucket sql covers all dialect-granularity branches', () => {
  const col = sql.ref('demo.CreatedAt');

  const pgYear = renderOperationNode(buildTimeBucketExpr('postgres', col, 'year', 'UTC'));
  const pgQuarter = renderOperationNode(buildTimeBucketExpr('postgres', col, 'quarter'));
  const pgWeek = renderOperationNode(buildTimeBucketExpr('postgres', col, 'week'));
  const pgDay = renderOperationNode(buildTimeBucketExpr('postgres', col, 'day'));
  expect(pgYear.includes("DATE_TRUNC('year'")).toBe(true);
  expect(pgYear.includes('AT TIME ZONE')).toBe(true);
  expect(pgQuarter.includes("DATE_TRUNC('quarter'")).toBe(true);
  expect(pgWeek.includes("DATE_TRUNC('week'")).toBe(true);
  expect(pgDay.includes("DATE_TRUNC('day'")).toBe(true);

  const mysqlYear = renderOperationNode(buildTimeBucketExpr('mysql', col, 'year', 'UTC'));
  const mysqlQuarter = renderOperationNode(buildTimeBucketExpr('mysql', col, 'quarter', 'UTC'));
  const mysqlMonth = renderOperationNode(buildTimeBucketExpr('mysql', col, 'month', 'UTC'));
  const mysqlDay = renderOperationNode(buildTimeBucketExpr('mysql', col, 'day', 'UTC'));
  expect(mysqlYear.includes('DATE_FORMAT')).toBe(true);
  expect(mysqlQuarter.includes('QUARTER')).toBe(true);
  expect(mysqlMonth.includes('%Y-%m-01')).toBe(true);
  expect(mysqlDay.includes('%Y-%m-%d 00:00:00')).toBe(true);

  const sqliteYear = renderOperationNode(buildTimeBucketExpr('sqlite', col, 'year', 'UTC'));
  const sqliteQuarter = renderOperationNode(buildTimeBucketExpr('sqlite', col, 'quarter'));
  const sqliteMonth = renderOperationNode(buildTimeBucketExpr('sqlite', col, 'month'));
  const sqliteDay = renderOperationNode(buildTimeBucketExpr('sqlite', col, 'day'));
  expect(sqliteYear.includes('start of year')).toBe(true);
  expect(sqliteQuarter.includes('STRFTIME')).toBe(true);
  expect(sqliteMonth.includes('start of month')).toBe(true);
  expect(sqliteDay.includes('start of day')).toBe(true);

  const mssqlYear = renderOperationNode(buildTimeBucketExpr('mssql', col, 'year', 'UTC'));
  const mssqlQuarter = renderOperationNode(buildTimeBucketExpr('mssql', col, 'quarter', 'UTC'));
  const mssqlMonth = renderOperationNode(buildTimeBucketExpr('mssql', col, 'month', 'UTC'));
  const mssqlWeek = renderOperationNode(buildTimeBucketExpr('mssql', col, 'week', 'UTC'));
  const mssqlDay = renderOperationNode(buildTimeBucketExpr('mssql', col, 'day', 'UTC'));
  expect(mssqlYear.includes('DATEFROMPARTS')).toBe(true);
  expect(mssqlQuarter.includes('DATEPART')).toBe(true);
  expect(mssqlMonth.includes('MONTH')).toBe(true);
  expect(mssqlWeek.includes('DATEDIFF')).toBe(true);
  expect(mssqlDay.includes('CAST')).toBe(true);
});

test('repository time bucket sql throws on unsupported granularity and dialect', () => {
  const col = sql.ref('demo.CreatedAt');

  expect(() => buildTimeBucketExpr('postgres', col, 'hour' as any)).toThrow('Unsupported granularity for postgres: hour');
  expect(() => buildTimeBucketExpr('mysql', col, 'hour' as any)).toThrow('Unsupported granularity for mysql: hour');
  expect(() => buildTimeBucketExpr('sqlite', col, 'hour' as any)).toThrow('Unsupported granularity for sqlite: hour');
  expect(() => buildTimeBucketExpr('mssql', col, 'hour' as any)).toThrow('Unsupported granularity for mssql: hour');
  expect(() => buildTimeBucketExpr('oracle' as any, col, 'day')).toThrow('Unsupported dialect: oracle');
});

test('repository time bucket sql keeps sqlite and mssql expressions unchanged when timezone is provided', () => {
  const col = sql.ref('demo.CreatedAt');

  const sqliteExpr = renderOperationNode(buildTimeBucketExpr('sqlite', col, 'day', "Asia/Shanghai'; DROP"));
  const mssqlExpr = renderOperationNode(buildTimeBucketExpr('mssql', col, 'day', 'Asia/Shanghai'));

  expect(sqliteExpr.includes('CONVERT_TZ')).toBe(false);
  expect(sqliteExpr.includes('AT TIME ZONE')).toBe(false);
  expect(mssqlExpr.includes('CONVERT_TZ')).toBe(false);
  expect(mssqlExpr.includes('AT TIME ZONE')).toBe(false);
});

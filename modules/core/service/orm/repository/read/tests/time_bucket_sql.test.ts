// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { sql } from 'kysely';
import { buildTimeBucketExpr, resolveFixedUtcOffsetMinutes } from '../time_bucket_sql';

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

test('repository time bucket sql applies fixed offset on sqlite/mssql for non-DST zones', () => {
  const col = sql.ref('demo.CreatedAt');

  // Asia/Dubai is fixed +04 across the CASE window (unlike Asia/Shanghai which had DST in 1990–91).
  const sqliteExpr = renderOperationNode(buildTimeBucketExpr('sqlite', col, 'day', 'Asia/Dubai'));
  const mssqlExpr = renderOperationNode(buildTimeBucketExpr('mssql', col, 'day', 'Asia/Dubai'));

  expect(sqliteExpr.includes('datetime')).toBe(true);
  expect(sqliteExpr.includes('+04:00')).toBe(true);
  expect(sqliteExpr.includes('AT TIME ZONE')).toBe(false);
  expect(mssqlExpr.includes('DATEADD')).toBe(true);
  expect(mssqlExpr.includes('240')).toBe(true);
});

test('repository time bucket sql rejects invalid IANA on mysql', () => {
  const col = sql.ref('demo.CreatedAt');
  expect(() => buildTimeBucketExpr('mysql', col, 'day', "Asia/Shanghai'; DROP")).toThrow(/Invalid IANA/);
});

test('repository time bucket sql rejects invalid IANA on sqlite', () => {
  const col = sql.ref('demo.CreatedAt');
  expect(() => buildTimeBucketExpr('sqlite', col, 'day', "Asia/Shanghai'; DROP")).toThrow(/Invalid IANA/);
});

test('repository time bucket sql sqlite UTC timezone leaves column without datetime offset', () => {
  const col = sql.ref('demo.CreatedAt');
  const withUtc = renderOperationNode(buildTimeBucketExpr('sqlite', col, 'day', 'UTC'));
  const without = renderOperationNode(buildTimeBucketExpr('sqlite', col, 'day'));
  expect(withUtc.includes('+08:00')).toBe(false);
  expect(withUtc.includes('start of day')).toBe(true);
  expect(without.includes('start of day')).toBe(true);
});

test('repository time bucket sql resolveFixedUtcOffsetMinutes uses in-window segments not a single year snapshot', () => {
  expect(resolveFixedUtcOffsetMinutes('Asia/Dubai')).toBe(240);
  expect(resolveFixedUtcOffsetMinutes('UTC')).toBe(0);
  expect(resolveFixedUtcOffsetMinutes('America/New_York')).toBe(null);
  // Shanghai observed DST inside 1990–2040 → must not take the fixed +08 fast path.
  expect(resolveFixedUtcOffsetMinutes('Asia/Shanghai')).toBe(null);
});

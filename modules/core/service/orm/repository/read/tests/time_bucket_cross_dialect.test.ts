// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { sql } from 'kysely';
import { coerceToBucketStart } from '../time_bucket_runtime';
import { buildTimeBucketExpr, listZoneOffsetSegments, resolveFixedUtcOffsetMinutes } from '../time_bucket_sql';
import { simulateDialectBucketStart, TIME_BUCKET_CROSS_DIALECT_FIXTURES } from '../time_bucket_fixtures';

function renderOperationNode(expr: any): string {
  return JSON.stringify((expr as any).toOperationNode());
}

test('cross-dialect time bucket fixtures: runtime matches expected keys', () => {
  for (const fixture of TIME_BUCKET_CROSS_DIALECT_FIXTURES) {
    const got = coerceToBucketStart(fixture.instant, fixture.granularity, fixture.timezone);
    expect(got.toISOString()).toBe(fixture.expected);
  }
});

test('cross-dialect time bucket fixtures: sqlite/mssql/postgres/mysql agree on every fixture', () => {
  const dialects = ['postgres', 'mysql', 'sqlite', 'mssql'] as const;
  for (const fixture of TIME_BUCKET_CROSS_DIALECT_FIXTURES) {
    const keys = dialects.map(dialect =>
      simulateDialectBucketStart(dialect, fixture.instant, fixture.granularity, fixture.timezone).toISOString()
    );
    for (const key of keys) {
      expect(key).toBe(fixture.expected);
    }
  }
});

test('cross-dialect time bucket sqlite DST SQL emits CASE offset transitions for America/New_York', () => {
  const col = sql.ref('demo.CreatedAt');
  const rendered = renderOperationNode(buildTimeBucketExpr('sqlite', col, 'day', 'America/New_York'));
  expect(rendered.includes('CASE')).toBe(true);
  expect(rendered.includes('-05:00') || rendered.includes('-04:00')).toBe(true);
  expect(listZoneOffsetSegments('America/New_York').length).toBeGreaterThan(10);
});

test('cross-dialect time bucket sqlite Shanghai SQL still uses fixed +08:00 without CASE', () => {
  const col = sql.ref('demo.CreatedAt');
  const rendered = renderOperationNode(buildTimeBucketExpr('sqlite', col, 'day', 'Asia/Shanghai'));
  expect(rendered.includes('+08:00')).toBe(true);
  expect(rendered.includes('CASE WHEN')).toBe(false);
  expect(resolveFixedUtcOffsetMinutes('Asia/Shanghai')).toBe(480);
});

test('cross-dialect time bucket mssql DST SQL emits CASE DATEADD transitions', () => {
  const col = sql.ref('demo.CreatedAt');
  const rendered = renderOperationNode(buildTimeBucketExpr('mssql', col, 'day', 'America/New_York'));
  expect(rendered.includes('CASE')).toBe(true);
  expect(rendered.includes('DATEADD')).toBe(true);
});

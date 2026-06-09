// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getStringHelpers } from '..';

function createQb() {
  const calls: Array<Record<string, any>> = [];
  const fn: any = (name: string, args: any[]) => {
    calls.push({ name, args });
    return { kind: 'fn', name, args };
  };
  fn.coalesce = (expr: any, fallback: any) => {
    calls.push({ name: 'coalesce', args: [expr, fallback] });
    return { kind: 'coalesce', expr, fallback };
  };
  return { qb: { fn }, calls };
}

test('repository string helpers route concat/lower to postgres and mssql function branches', () => {
  const pg = createQb();
  const pgHelpers = getStringHelpers('postgres', pg.qb as any);

  const pgConcat = pgHelpers.concat({ kind: 'expr' } as any, 'x');
  expect((pgConcat as any).kind).toBe('fn');
  expect((pgConcat as any).name).toBe('concat_ws');
  expect(Array.isArray((pgConcat as any).args)).toBe(true);
  expect(pg.calls.some(call => call.name === 'coalesce')).toBe(true);

  const lowered = pgHelpers.lower({ kind: 'expr-lower' } as any);
  expect(lowered).toEqual({ kind: 'fn', name: 'lower', args: [{ kind: 'expr-lower' }] });

  const ms = createQb();
  const msHelpers = getStringHelpers('mssql', ms.qb as any);
  const msConcat = msHelpers.concat({ kind: 'expr' } as any, "o'hara");
  expect((msConcat as any).kind).toBe('fn');
  expect((msConcat as any).name).toBe('concat');
  expect(Array.isArray((msConcat as any).args)).toBe(true);

  const msConcatWs = msHelpers.concatWs('-', { kind: 'expr' } as any, 'tail');
  expect((msConcatWs as any).kind).toBe('fn');
  expect((msConcatWs as any).name).toBe('concat_ws');
  expect(Array.isArray((msConcatWs as any).args)).toBe(true);
});

test('repository string helpers use sqlite join branch for concat and concatWs', () => {
  const sqlite = createQb();
  const helpers = getStringHelpers('sqlite', sqlite.qb as any);

  const concatExpr = helpers.concat({ kind: 'expr' } as any, 'tail');
  expect(concatExpr).toBeTruthy();
  expect(typeof (concatExpr as any).toOperationNode).toBe('function');

  const concatWsExpr = helpers.concatWs('::', { kind: 'expr' } as any, 'tail');
  expect(concatWsExpr).toBeTruthy();
  expect(typeof (concatWsExpr as any).toOperationNode).toBe('function');

  expect(sqlite.calls).toEqual([]);
});

test('repository string helpers use mysql concat_ws branch with non-postgres separator literal', () => {
  const mysql = createQb();
  const helpers = getStringHelpers('mysql' as any, mysql.qb as any);

  const result = helpers.concat({ kind: 'expr' } as any, 'tail');
  expect((result as any).kind).toBe('fn');
  expect((result as any).name).toBe('concat_ws');
  expect(Array.isArray((result as any).args)).toBe(true);
  expect(((result as any).args || []).length >= 2).toBe(true);
});

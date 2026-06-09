// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { buildContainsExpression } from '..';

function createExpressionBuilder() {
  const eb: any = (lhs: any, op: any, rhs: any) => ({ lhs, op, rhs });
  eb.fn = (name: string, args: any[]) => ({ kind: 'fn', name, args });
  return eb;
}

test('repository json contains builds postgres/mysql expressions and sqlite raw expression', () => {
  const eb = createExpressionBuilder();

  const pg = buildContainsExpression('postgres', eb, 'col_json', 'plain-string');
  expect((pg as any).lhs).toBe('col_json');
  expect((pg as any).op).toBe('@>');
  expect(Boolean((pg as any).rhs)).toBe(true);

  const mysql = buildContainsExpression('mysql', eb, 'col_json', { k: 'v' });
  expect((mysql as any).kind).toBe('fn');
  expect((mysql as any).name).toBe('JSON_CONTAINS');
  expect(Array.isArray((mysql as any).args)).toBe(true);

  const sqlite = buildContainsExpression('sqlite', eb, 'col_json', 'line"quoted"value', 'demo_table', 'Payload');
  expect(sqlite).toBeTruthy();
  expect(typeof (sqlite as any).toOperationNode).toBe('function');
});

test('repository json contains throws for unsupported or incomplete dialect inputs', () => {
  const eb = createExpressionBuilder();

  expect(() => buildContainsExpression('sqlite', eb, 'col_json', { k: 'v' })).toThrow('SQLite contains requires selfTable and fieldName arguments');
  expect(() => buildContainsExpression('mssql', eb, 'col_json', { k: 'v' })).toThrow('contains is not supported for the MSSQL dialect yet');
  expect(() => buildContainsExpression('oracle' as any, eb, 'col_json', { k: 'v' })).toThrow('contains does not support database dialect: oracle');
});

test('repository json contains sqlite branch supports non-string rhs payload', () => {
  const eb = createExpressionBuilder();
  const sqlite = buildContainsExpression('sqlite', eb, 'col_json', { nested: { k: 'v' } }, 'demo_table', 'Payload');
  expect(sqlite).toBeTruthy();
  expect(typeof (sqlite as any).toOperationNode).toBe('function');
});

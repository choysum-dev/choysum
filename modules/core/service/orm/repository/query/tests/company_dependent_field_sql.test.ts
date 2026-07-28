// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext } from '../../../../runtime/context';
import { buildCompanyDependentFieldUnwrapExpr } from '../company_dependent_field_sql';
import { quoteJsonObjectPath, quoteSqlStringLiteral } from '../json_path';

function createExpressionBuilder() {
  const eb: any = (lhs: any, op: any, rhs: any) => ({ lhs, op, rhs });
  eb.ref = (path: string) => ({ kind: 'ref', path });
  return eb;
}

test('quoteJsonObjectPath quotes special keys for MySQL/SQLite/MSSQL extractors', () => {
  expect(quoteJsonObjectPath('comp_main')).toBe('$."comp_main"');
  expect(quoteJsonObjectPath('comp-eu')).toBe('$."comp-eu"');
  expect(quoteJsonObjectPath('a.b')).toBe('$."a.b"');
  expect(quoteJsonObjectPath('say "hi"')).toBe('$."say \\"hi\\""');
  expect(quoteSqlStringLiteral("O'Brien")).toBe("'O''Brien'");
});

test('buildCompanyDependentFieldUnwrapExpr quotes JSON path keys', () => {
  const eb = createExpressionBuilder();

  for (const [dialect, key] of [
    ['mysql', 'comp-eu'],
    ['mariadb', 'a.b'],
    ['sqlite', 'comp eu'],
    ['mssql', 'comp-eu'],
    ['postgres', "O'Brien"],
  ] as const) {
    const expr = buildCompanyDependentFieldUnwrapExpr(dialect, eb, 't.Cost', key) as any;
    expect(typeof expr.toOperationNode).toBe('function');
  }

  const empty = buildCompanyDependentFieldUnwrapExpr('postgres', eb, 't.Cost', '   ') as any;
  expect(typeof empty.toOperationNode).toBe('function');

  const fromCtx = withContext({ activeCompanyId: 'comp-eu' }, () =>
    buildCompanyDependentFieldUnwrapExpr('mysql', eb, 'demo.Cost')
  ) as any;
  expect(typeof fromCtx.toOperationNode).toBe('function');

  // Spot-check that MySQL path embeds the quoted-member form for hyphenated ids.
  // JSON.stringify escapes quotes, so look for $.\"comp-eu\" rather than $."comp-eu".
  const mysql = buildCompanyDependentFieldUnwrapExpr('mysql', eb, 't.Cost', 'comp-eu') as any;
  const raw = JSON.stringify(mysql.toOperationNode());
  expect(raw.includes('JSON_EXTRACT')).toBe(true);
  expect(raw.includes('$.\\"comp-eu\\"')).toBe(true);
  expect(raw.includes('$.comp-eu')).toBe(false);
});

test('buildCompanyDependentFieldUnwrapExpr covers all dialect aliases', () => {
  const eb = createExpressionBuilder();
  for (const dialect of ['postgres', 'postgresql', 'mysql', 'mariadb', 'sqlite', 'mssql', 'sqlserver', 'unknown'] as const) {
    const expr = buildCompanyDependentFieldUnwrapExpr(dialect, eb, 't.Cost', 'comp_main') as any;
    expect(typeof expr.toOperationNode).toBe('function');
    const raw = JSON.stringify(expr.toOperationNode());
    if (dialect === 'postgres' || dialect === 'postgresql' || dialect === 'unknown') {
      expect(raw.includes('->>')).toBe(true);
    } else if (dialect === 'mysql' || dialect === 'mariadb') {
      expect(raw.includes('JSON_EXTRACT')).toBe(true);
    } else if (dialect === 'sqlite') {
      expect(raw.includes('json_extract')).toBe(true);
    } else {
      expect(raw.includes('JSON_VALUE')).toBe(true);
    }
  }
  expect(quoteJsonObjectPath('a\\b"c')).toBe('$."a\\\\b\\"c"');
});

test('buildCompanyDependentFieldUnwrapExpr defaults empty dialect to postgres path', () => {
  const eb = createExpressionBuilder();
  const expr = buildCompanyDependentFieldUnwrapExpr('', eb, 't.Cost', 'comp_main') as any;
  const raw = JSON.stringify(expr.toOperationNode());
  expect(raw.includes('->>')).toBe(true);
  const expr2 = buildCompanyDependentFieldUnwrapExpr(undefined as any, eb, 't.Cost', 'comp_main') as any;
  expect(typeof expr2.toOperationNode).toBe('function');
});

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext } from '../../../../runtime/context';
import {
  buildTranslatedFieldUnwrapExpr,
  buildTranslatedTrigramPrefilterLhs,
  resolveTranslatedTrigramPrefilterPattern,
} from '../translated_field_sql';

function createExpressionBuilder() {
  const eb: any = (lhs: any, op: any, rhs: any) => ({ lhs, op, rhs });
  eb.ref = (path: string) => ({ kind: 'ref', path });
  eb.fn = (name: string, args: any[]) => ({ kind: 'fn', name, args });
  return eb;
}

test('buildTranslatedFieldUnwrapExpr builds dialect-specific unwrap expressions', () => {
  const eb = createExpressionBuilder();

  const pgBase = buildTranslatedFieldUnwrapExpr('postgres', eb, 't.Name', 'en_US') as any;
  expect(typeof pgBase.toOperationNode).toBe('function');

  const pgOther = buildTranslatedFieldUnwrapExpr('postgresql', eb, 't.Name', 'zh_CN') as any;
  expect(typeof pgOther.toOperationNode).toBe('function');

  const mysql = buildTranslatedFieldUnwrapExpr('mysql', eb, 't.Name', 'zh_CN') as any;
  expect(typeof mysql.toOperationNode).toBe('function');

  const mariadb = buildTranslatedFieldUnwrapExpr('mariadb', eb, 't.Name', 'en_US') as any;
  expect(typeof mariadb.toOperationNode).toBe('function');

  const sqlite = buildTranslatedFieldUnwrapExpr('sqlite', eb, 't.Name', 'zh_CN') as any;
  expect(typeof sqlite.toOperationNode).toBe('function');

  const mssql = buildTranslatedFieldUnwrapExpr('mssql', eb, 't.Name', 'zh_CN') as any;
  expect(typeof mssql.toOperationNode).toBe('function');

  const sqlserver = buildTranslatedFieldUnwrapExpr('sqlserver', eb, 't.Name', 'en_US') as any;
  expect(typeof sqlserver.toOperationNode).toBe('function');

  const fallback = buildTranslatedFieldUnwrapExpr('oracle' as any, eb, 't.Name', 'zh_CN') as any;
  expect(typeof fallback.toOperationNode).toBe('function');

  const escaped = buildTranslatedFieldUnwrapExpr('postgres', eb, 't.Name', "O'Brien") as any;
  expect(typeof escaped.toOperationNode).toBe('function');
});

test('buildTranslatedFieldUnwrapExpr resolves lang from request context', () => {
  const eb = createExpressionBuilder();
  const expr = withContext({ lang: 'zh_CN' }, () => buildTranslatedFieldUnwrapExpr('sqlite', eb, 'demo.Name')) as any;
  expect(typeof expr.toOperationNode).toBe('function');
});

test('buildTranslatedTrigramPrefilterLhs and resolveTranslatedTrigramPrefilterPattern cover ops', () => {
  const eb = createExpressionBuilder();
  const lhs = buildTranslatedTrigramPrefilterLhs(eb, 't.Name') as any;
  expect(typeof lhs.toOperationNode).toBe('function');

  expect(resolveTranslatedTrigramPrefilterPattern('=', 'abc')).toBe('%abc%');
  expect(resolveTranslatedTrigramPrefilterPattern('==', 'ab')).toBeNull();
  expect(resolveTranslatedTrigramPrefilterPattern('like', 'hello')).toBe('hello');
  expect(resolveTranslatedTrigramPrefilterPattern('=like', '%abc%')).toBe('%abc%');
  expect(resolveTranslatedTrigramPrefilterPattern('in', [123 as any])).toBeNull();
  expect(resolveTranslatedTrigramPrefilterPattern('contains', 'abc')).toBeNull();
});

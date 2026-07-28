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
  const pgBaseNode = JSON.stringify(pgBase.toOperationNode());
  expect(pgBaseNode.includes('COALESCE')).toBe(false);

  const pgOther = buildTranslatedFieldUnwrapExpr('postgresql', eb, 't.Name', 'zh_CN') as any;
  expect(typeof pgOther.toOperationNode).toBe('function');
  expect(JSON.stringify(pgOther.toOperationNode()).includes('COALESCE')).toBe(true);

  const mysql = buildTranslatedFieldUnwrapExpr('mysql', eb, 't.Name', 'zh_CN') as any;
  expect(typeof mysql.toOperationNode).toBe('function');
  expect(JSON.stringify(mysql.toOperationNode()).includes('COALESCE')).toBe(true);

  const mariadb = buildTranslatedFieldUnwrapExpr('mariadb', eb, 't.Name', 'en_US') as any;
  expect(typeof mariadb.toOperationNode).toBe('function');
  expect(JSON.stringify(mariadb.toOperationNode()).includes('COALESCE')).toBe(false);

  const sqlite = buildTranslatedFieldUnwrapExpr('sqlite', eb, 't.Name', 'zh_CN') as any;
  expect(typeof sqlite.toOperationNode).toBe('function');
  expect(JSON.stringify(sqlite.toOperationNode()).includes('COALESCE')).toBe(true);

  const sqliteBase = buildTranslatedFieldUnwrapExpr('sqlite', eb, 't.Name', 'en_US') as any;
  expect(JSON.stringify(sqliteBase.toOperationNode()).includes('COALESCE')).toBe(false);

  const mssql = buildTranslatedFieldUnwrapExpr('mssql', eb, 't.Name', 'zh_CN') as any;
  expect(typeof mssql.toOperationNode).toBe('function');
  expect(JSON.stringify(mssql.toOperationNode()).includes('COALESCE')).toBe(true);

  const sqlserver = buildTranslatedFieldUnwrapExpr('sqlserver', eb, 't.Name', 'en_US') as any;
  expect(typeof sqlserver.toOperationNode).toBe('function');
  expect(JSON.stringify(sqlserver.toOperationNode()).includes('COALESCE')).toBe(false);

  const fallback = buildTranslatedFieldUnwrapExpr('oracle' as any, eb, 't.Name', 'zh_CN') as any;
  expect(typeof fallback.toOperationNode).toBe('function');
  expect(JSON.stringify(fallback.toOperationNode()).includes('COALESCE')).toBe(true);

  const emptyDialect = buildTranslatedFieldUnwrapExpr('', eb, 't.Name', 'zh_CN') as any;
  expect(typeof emptyDialect.toOperationNode).toBe('function');

  const emptyLang = buildTranslatedFieldUnwrapExpr('postgres', eb, 't.Name', '   ') as any;
  expect(JSON.stringify(emptyLang.toOperationNode()).includes('COALESCE')).toBe(false);

  const escaped = buildTranslatedFieldUnwrapExpr('postgres', eb, 't.Name', "O'Brien") as any;
  expect(typeof escaped.toOperationNode).toBe('function');

  const mysqlSpecial = buildTranslatedFieldUnwrapExpr('mysql', eb, 't.Name', 'zh-CN') as any;
  const mysqlSpecialNode = JSON.stringify(mysqlSpecial.toOperationNode());
  expect(mysqlSpecialNode.includes('$."zh-CN"') || mysqlSpecialNode.includes('zh-CN')).toBe(true);
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

test('buildTranslatedFieldUnwrapExpr defaults to en_US without lang context', () => {
  const eb = createExpressionBuilder();
  const expr = buildTranslatedFieldUnwrapExpr('sqlite', eb, 'demo.Name') as any;
  expect(typeof expr.toOperationNode).toBe('function');
  expect(JSON.stringify(expr.toOperationNode()).includes('COALESCE')).toBe(false);
});

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isInertQueryWithReturning, isResultSetQuery } from './is-result-set-query';

function compiled(query: unknown, sql: string) {
  return {
    query,
    sql,
    parameters: [],
  } as any;
}

test('isResultSetQuery returns true for select and raw select/explain', () => {
  expect(isResultSetQuery(compiled({ kind: 'SelectQueryNode' }, 'select 1'))).toBe(true);
  expect(isResultSetQuery(compiled({ kind: 'RawNode' }, 'select 1'))).toBe(true);
  expect(isResultSetQuery(compiled({ kind: 'RawNode' }, '  explain select 1'))).toBe(true);
  expect(isResultSetQuery(compiled({ kind: 'RawNode' }, '(select 1)'))).toBe(true);
});

test('isResultSetQuery handles with-sql branches and write statements', () => {
  expect(isResultSetQuery(compiled({ kind: 'RawNode' }, 'with t as (select 1) select * from t'))).toBe(true);
  expect(isResultSetQuery(compiled({ kind: 'RawNode' }, 'with t as (select 1) delete from t'))).toBe(false);
  expect(isResultSetQuery(compiled({ kind: 'RawNode' }, 'with t as (select 1) insert into a values (1)'))).toBe(false);
  expect(isResultSetQuery(compiled({ kind: 'RawNode' }, 'with t as (select 1) replace into a values (1)'))).toBe(false);
  expect(isResultSetQuery(compiled({ kind: 'RawNode' }, 'with t as (select 1) update a set b = 1'))).toBe(false);
});

test('isInertQueryWithReturning and explain-on-dml behavior', () => {
  const insertReturning = compiled({ kind: 'InsertQueryNode', returning: {} }, 'insert into t values (1) returning id');
  const insertNoReturning = compiled({ kind: 'InsertQueryNode' }, 'insert into t values (1)');
  const updateExplain = compiled({ kind: 'UpdateQueryNode', explain: { kind: 'ExplainNode' } }, 'update t set a = 1');
  const deleteNoExplain = compiled({ kind: 'DeleteQueryNode' }, 'delete from t');
  const unrelated = compiled({ kind: 'CreateTableNode' }, 'create table t(id int)');

  expect(isInertQueryWithReturning(insertReturning)).toBe(true);
  expect(isInertQueryWithReturning(insertNoReturning)).toBe(false);

  expect(isResultSetQuery(updateExplain)).toBe(true);
  expect(isResultSetQuery(deleteNoExplain)).toBe(false);
  expect(isResultSetQuery(unrelated)).toBe(false);
});

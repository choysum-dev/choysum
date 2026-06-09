// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { throwRepositorySqlWriteError } from '..';

function expectWrapped(meta: any, error: unknown, mode: 'create' | 'update' | 'preview') {
  try {
    throwRepositorySqlWriteError(meta, error, mode);
    throw new Error('expected throw');
  } catch (e) {
    return e;
  }
}

test('validation sql helper wraps sqlite unique violation with normalized columns', () => {
  const e = expectWrapped(
    { fullModelName: 'demo.Model', modelName: '', name: '' },
    new Error('UNIQUE constraint failed: demo_table.Name, demo_table.Code'),
    'create'
  );

  expect(e instanceof ChoysumError).toBe(true);
  const wrapped = e as ChoysumError;
  expect(wrapped.code).toBe('validation_failed');
  expect(wrapped.metadata.issueCode).toBe('sql_unique_violation');
  expect(wrapped.metadata.sqlField).toBe('Name');
  expect(wrapped.metadata.sqlColumns).toBe('Name,Code');
});

test('validation sql helper wraps postgres unique and check constraint violations', () => {
  const unique = expectWrapped(
    { fullModelName: 'demo.Model', modelName: '', name: '' },
    new Error('duplicate key value violates unique constraint "uq_demo_name"'),
    'update'
  ) as ChoysumError;
  expect(unique.metadata.issueCode).toBe('sql_unique_violation');
  expect(unique.metadata.sqlConstraint).toBe('uq_demo_name');

  const pgCheck = expectWrapped(
    { fullModelName: 'demo.Model', modelName: '', name: '' },
    new Error('new row violates check constraint "ck_demo_status"'),
    'update'
  ) as ChoysumError;
  expect(pgCheck.metadata.issueCode).toBe('sql_check_violation');
  expect(pgCheck.metadata.sqlConstraint).toBe('ck_demo_status');
});

test('validation sql helper wraps sqlite check and foreign-key violations', () => {
  const sqliteCheck = expectWrapped(
    { fullModelName: 'demo.Model', modelName: '', name: '' },
    new Error('CHECK constraint failed: ck_demo_age'),
    'update'
  ) as ChoysumError;
  expect(sqliteCheck.metadata.issueCode).toBe('sql_check_violation');
  expect(sqliteCheck.metadata.sqlConstraint).toBe('ck_demo_age');

  const sqliteFk = expectWrapped({ fullModelName: 'demo.Model', modelName: '', name: '' }, new Error('FOREIGN KEY constraint failed'), 'update') as ChoysumError;
  expect(sqliteFk.metadata.issueCode).toBe('sql_fk_violation');
});

test('validation sql helper wraps postgres fk and supports metadata/cause text collection', () => {
  const cause = new Error('violates foreign key constraint "fk_demo_partner"');
  const err: any = new Error('outer sql error');
  err.cause = cause;
  err.metadata = { extra: 'ignored but collectable' };

  const wrapped = expectWrapped({ fullModelName: 'demo.Model', modelName: '', name: '' }, err, 'preview') as ChoysumError;

  expect(wrapped.metadata.issueCode).toBe('sql_fk_violation');
  expect(wrapped.metadata.sqlConstraint).toBe('fk_demo_partner');
});

test('validation sql helper rethrows original error when no sql signature is matched', () => {
  const raw = new Error('plain runtime failure');
  const thrown = expectWrapped({ fullModelName: 'demo.Model' }, raw, 'create');
  expect(thrown).toBe(raw);
});

test('validation sql helper handles sqlite check without constraint name and empty candidates', () => {
  const sqliteCheckNoName = expectWrapped(
    { fullModelName: 'demo.Model', modelName: '', name: '' },
    new Error('CHECK constraint failed'),
    'update'
  ) as ChoysumError;
  expect(sqliteCheckNoName.metadata.issueCode).toBe('sql_check_violation');
  expect(sqliteCheckNoName.metadata.sqlConstraint).toBeUndefined();

  const blankRaw = expectWrapped({ fullModelName: 'demo.Model' }, '', 'create');
  expect(blankRaw).toBe('');
});

test('validation sql helper collectTexts covers non-error inputs, empty message/toString, metadata empties and depth cutoff', () => {
  const objRaw = expectWrapped({ fullModelName: 'demo.Model' }, { metadata: { a: '  ', b: '' } } as any, 'create');
  expect(objRaw).toEqual({ metadata: { a: '  ', b: '' } });

  const blankObject: any = { toString: () => '   ' };
  const blankObjectRaw = expectWrapped({ fullModelName: 'demo.Model' }, blankObject, 'create');
  expect(blankObjectRaw).toBe(blankObject);

  const emptyErr: any = new Error('base');
  emptyErr.message = '';
  emptyErr.toString = () => '';
  emptyErr.metadata = { keep: '', trim: '   ' };
  const out = expectWrapped({ fullModelName: 'demo.Model' }, emptyErr, 'create');
  expect(out).toBe(emptyErr);

  const e1: any = new Error('l1');
  const e2: any = new Error('l2');
  const e3: any = new Error('l3');
  const e4: any = new Error('l4');
  const e5: any = new Error('l5');
  const e6: any = new Error('duplicate key value violates unique constraint "uq_from_deep"');
  e5.cause = e6;
  e4.cause = e5;
  e3.cause = e4;
  e2.cause = e3;
  e1.cause = e2;

  const wrappedDeep = expectWrapped({ fullModelName: 'demo.Model' }, e1, 'update');
  expect(wrappedDeep).toBe(e1);
});

test('validation sql helper normalizeSqlColumns fallback keeps original segment when split parts are empty', () => {
  const wrapped = expectWrapped(
    { fullModelName: 'demo.Model', modelName: '', name: '' },
    new Error('UNIQUE constraint failed: ..., demo_table.Name'),
    'create'
  ) as ChoysumError;

  expect(wrapped.metadata.issueCode).toBe('sql_unique_violation');
  expect(wrapped.metadata.sqlColumns).toBe('...,Name');
});

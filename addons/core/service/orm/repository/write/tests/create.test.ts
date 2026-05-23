// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { applyRepositoryCreatePostWrite, executeRepositoryCreate, insertRepositoryCreateEntities } from '../create';

test('repository create runtime inserts prepared entities and returns created ids', async () => {
  const calls: Array<Record<string, any>> = [];
  const insertQuery = { kind: 'insert-query' };
  const valuesQuery = {
    returning(field: unknown) {
      calls.push({ method: 'returning', field });
      return insertQuery;
    },
  };
  const db = {
    insertInto(table: string) {
      calls.push({ method: 'insertInto', table });
      return {
        values(input: unknown) {
          calls.push({ method: 'values', input });
          return valuesQuery;
        },
      };
    },
  };

  const ids = await insertRepositoryCreateEntities(
    {
      db,
      table: 'demo_table',
      async execute(query) {
        calls.push({ method: 'execute', query });
        return [{ Id: 'row_1' }, { Id: 'row_2' }] as any;
      },
      wrapSqlWriteError(error, mode) {
        calls.push({ method: 'wrap', error, mode });
        throw error;
      },
    },
    [{ Id: 'row_1' } as any, { Id: 'row_2' } as any]
  );

  expect(ids).toEqual(['row_1', 'row_2']);
  expect(calls).toEqual([
    { method: 'insertInto', table: 'demo_table' },
    { method: 'values', input: [{ Id: 'row_1' }, { Id: 'row_2' }] },
    { method: 'returning', field: 'Id' },
    { method: 'execute', query: insertQuery },
  ]);
});

test('repository create runtime delegates insert errors to sql write wrapper', async () => {
  const db = {
    insertInto() {
      return {
        values() {
          return {
            returning() {
              return { kind: 'insert-query' };
            },
          };
        },
      };
    },
  };
  const sqlError = new Error('boom');
  const wrappedError = new Error('wrapped');
  const calls: Array<Record<string, any>> = [];

  let actualError: unknown;
  try {
    await insertRepositoryCreateEntities(
      {
        db,
        table: 'demo_table',
        async execute() {
          throw sqlError;
        },
        wrapSqlWriteError(error, mode) {
          calls.push({ method: 'wrap', error, mode });
          throw wrappedError;
        },
      },
      [{ Id: 'row_1' } as any]
    );
  } catch (error) {
    actualError = error;
  }

  expect(actualError).toBe(wrappedError);
  expect(calls).toEqual([{ method: 'wrap', error: sqlError, mode: 'create' }]);
});

test('repository create post-write skips record-rule assertion when no ids are created', async () => {
  const calls: Array<Record<string, any>> = [];
  const ids = await applyRepositoryCreatePostWrite(
    {
      async assertRecordRuleAllCreatedAllowed(createdIds, env) {
        calls.push({ method: 'assert', createdIds, env });
      },
    },
    [],
    { kind: 'condition', condition: ['Id', '!=', null] } as any
  );

  expect(ids).toEqual([]);
  expect(calls).toEqual([]);
});

test('repository create post-write asserts record-rule for created ids', async () => {
  const calls: Array<Record<string, any>> = [];
  const ids = await applyRepositoryCreatePostWrite(
    {
      async assertRecordRuleAllCreatedAllowed(createdIds, env) {
        calls.push({ method: 'assert', createdIds, env });
      },
    },
    ['row_1', 'row_2'],
    { kind: 'condition', condition: ['Id', '!=', null] } as any
  );

  expect(ids).toEqual(['row_1', 'row_2']);
  expect(calls).toEqual([
    {
      method: 'assert',
      createdIds: ['row_1', 'row_2'],
      env: { kind: 'condition', condition: ['Id', '!=', null] },
    },
  ]);
});

test('repository create executor composes authz prepare runtime and post-write helpers', async () => {
  const calls: Array<Record<string, any>> = [];
  const ids = await executeRepositoryCreate(
    {
      meta: { fullModelName: 'demo.Model' } as any,
      async getRecordRuleEnvelope(op) {
        calls.push({ method: 'recordRuleEnvelope', op });
        return { kind: 'condition', condition: ['Id', '!=', null] } as any;
      },
      permissionDenied(code, message, metadata) {
        calls.push({ method: 'permissionDenied', code, message, metadata });
        return new Error('permission');
      },
      async assertFieldRuleWriteAllowed(payload) {
        calls.push({ method: 'fieldRule', payload });
      },
      generateId() {
        calls.push({ method: 'generateId' });
        return 'generated_id';
      },
      applyDefaultCompanyIdOnCreate(entity) {
        calls.push({ method: 'defaultCompany', entity });
        return { ...entity, CompanyId: 'company_a' } as any;
      },
      async validateFields(input, mode) {
        calls.push({ method: 'validate', input, mode });
      },
      encodeForDb(input) {
        calls.push({ method: 'encode', input });
        return { ...input, Encoded: true } as any;
      },
      db: {
        insertInto(table: string) {
          calls.push({ method: 'insertInto', table });
          return {
            values(input: unknown) {
              calls.push({ method: 'values', input });
              return {
                returning(field: unknown) {
                  calls.push({ method: 'returning', field });
                  return { kind: 'insert-query' };
                },
              };
            },
          };
        },
      },
      table: 'demo_table',
      async execute(query) {
        calls.push({ method: 'execute', query });
        return [{ Id: 'generated_id' }] as any;
      },
      wrapSqlWriteError(error, mode) {
        calls.push({ method: 'wrap', error, mode });
        throw error;
      },
      async assertRecordRuleAllCreatedAllowed(createdIds, env) {
        calls.push({ method: 'recordRuleCreated', createdIds, env });
      },
    },
    [{ Name: 'first' } as any]
  );

  expect(ids).toEqual(['generated_id']);
  expect(calls).toEqual([
    { method: 'recordRuleEnvelope', op: 'create' },
    { method: 'fieldRule', payload: { Name: 'first' } },
    { method: 'generateId' },
    { method: 'defaultCompany', entity: { Id: 'generated_id', Name: 'first' } },
    { method: 'validate', input: { Id: 'generated_id', Name: 'first', CompanyId: 'company_a' }, mode: 'create' },
    { method: 'encode', input: { Id: 'generated_id', Name: 'first', CompanyId: 'company_a' } },
    { method: 'insertInto', table: 'demo_table' },
    { method: 'values', input: [{ Id: 'generated_id', Name: 'first', CompanyId: 'company_a', Encoded: true }] },
    { method: 'returning', field: 'Id' },
    { method: 'execute', query: { kind: 'insert-query' } },
    {
      method: 'recordRuleCreated',
      createdIds: ['generated_id'],
      env: { kind: 'condition', condition: ['Id', '!=', null] },
    },
  ]);
});

test('repository create executor invokes persist recompute hook after post-write checks', async () => {
  const calls: Array<Record<string, any>> = [];

  const ids = await executeRepositoryCreate(
    {
      meta: { fullModelName: 'demo.Model' } as any,
      async getRecordRuleEnvelope() {
        return { kind: 'condition', condition: ['Id', '!=', null] } as any;
      },
      permissionDenied() {
        return new Error('permission');
      },
      async assertFieldRuleWriteAllowed() {},
      generateId() {
        return 'generated_id';
      },
      applyDefaultCompanyIdOnCreate(entity) {
        return entity;
      },
      async validateFields() {},
      encodeForDb(input) {
        return input;
      },
      db: {
        insertInto() {
          return {
            values() {
              return {
                returning() {
                  return { kind: 'insert-query' };
                },
              };
            },
          };
        },
      },
      table: 'demo_table',
      async execute() {
        return [{ Id: 'generated_id' }] as any;
      },
      wrapSqlWriteError(error) {
        throw error as Error;
      },
      async assertRecordRuleAllCreatedAllowed(createdIds) {
        calls.push({ method: 'postWrite', createdIds });
      },
      async recomputePersistForCreate(createdIds, sanitizedEntities) {
        calls.push({ method: 'recompute', createdIds, sanitizedEntities });
      },
    },
    [{ Name: 'first' } as any]
  );

  expect(ids).toEqual(['generated_id']);
  expect(calls).toEqual([
    { method: 'postWrite', createdIds: ['generated_id'] },
    {
      method: 'recompute',
      createdIds: ['generated_id'],
      sanitizedEntities: [{ Id: 'generated_id', Name: 'first' }],
    },
  ]);
});

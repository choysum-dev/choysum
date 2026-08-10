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
      meta: { fullModelName: 'demo.Model', fields: new Map() } as any,
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
  expect(calls.filter(c => c.method === 'recordRuleEnvelope')).toEqual([{ method: 'recordRuleEnvelope', op: 'create' }]);
  expect(calls.filter(c => c.method === 'fieldRule')).toEqual([{ method: 'fieldRule', payload: { Name: 'first' } }]);
  expect(calls.filter(c => c.method === 'generateId')).toEqual([{ method: 'generateId' }]);
  expect(calls.filter(c => c.method === 'defaultCompany')).toEqual([
    { method: 'defaultCompany', entity: { Id: 'generated_id', Name: 'first' } },
  ]);
  const validate = calls.find(c => c.method === 'validate');
  expect(validate?.mode).toBe('create');
  expect(validate?.input.Id).toBe('generated_id');
  expect(validate?.input.Name).toBe('first');
  expect(validate?.input.CompanyId).toBe('company_a');
  expect(validate?.input.CreatedAt instanceof Date).toBe(true);
  expect(validate?.input.UpdatedAt instanceof Date).toBe(true);
  const encode = calls.find(c => c.method === 'encode');
  expect(encode?.input.Id).toBe('generated_id');
  expect(encode?.input.CreatedAt instanceof Date).toBe(true);
  expect(calls.filter(c => c.method === 'insertInto')).toEqual([{ method: 'insertInto', table: 'demo_table' }]);
  const values = calls.find(c => c.method === 'values');
  expect(values?.input).toHaveLength(1);
  expect(values?.input[0].Id).toBe('generated_id');
  expect(values?.input[0].Name).toBe('first');
  expect(values?.input[0].CompanyId).toBe('company_a');
  expect(values?.input[0].Encoded).toBe(true);
  expect(values?.input[0].CreatedAt instanceof Date).toBe(true);
  expect(calls.filter(c => c.method === 'returning')).toEqual([{ method: 'returning', field: 'Id' }]);
  expect(calls.filter(c => c.method === 'execute')).toEqual([{ method: 'execute', query: { kind: 'insert-query' } }]);
  expect(calls.filter(c => c.method === 'recordRuleCreated')).toEqual([
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
      meta: { fullModelName: 'demo.Model', fields: new Map() } as any,
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
  expect(calls).toHaveLength(2);
  expect(calls[0]).toEqual({ method: 'postWrite', createdIds: ['generated_id'] });
  expect(calls[1].method).toBe('recompute');
  expect(calls[1].createdIds).toEqual(['generated_id']);
  expect(calls[1].sanitizedEntities).toHaveLength(1);
  expect(calls[1].sanitizedEntities[0].Id).toBe('generated_id');
  expect(calls[1].sanitizedEntities[0].Name).toBe('first');
  expect(calls[1].sanitizedEntities[0].CreatedAt instanceof Date).toBe(true);
  expect(calls[1].sanitizedEntities[0].UpdatedAt instanceof Date).toBe(true);
});

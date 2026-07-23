// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ensureRepositoryCreateAllowed, prepareRepositoryCreateEntities } from '../create_helpers';
import { withContext } from '../../../../runtime/context';

test('repository create write helpers return record-rule envelope when create is allowed', async () => {
  const calls: Array<Record<string, any>> = [];
  const envelope = await ensureRepositoryCreateAllowed({
    meta: { fullModelName: 'auth.Role' } as any,
    async getRecordRuleEnvelope(op) {
      calls.push({ method: 'recordRuleEnvelope', op });
      return { kind: 'condition', condition: ['Id', '!=', null] } as any;
    },
    permissionDenied(code, message, metadata) {
      calls.push({ method: 'permissionDenied', code, message, metadata });
      return new Error('denied');
    },
  });

  expect(envelope).toEqual({ kind: 'condition', condition: ['Id', '!=', null] });
  expect(calls).toEqual([{ method: 'recordRuleEnvelope', op: 'create' }]);
});

test('repository create write helpers throw permission denied when create is blocked by record rule', async () => {
  const calls: Array<Record<string, any>> = [];
  const deniedError = new Error('denied');

  let actualError: unknown;
  try {
    await ensureRepositoryCreateAllowed({
      meta: { fullModelName: 'auth.Role' } as any,
      async getRecordRuleEnvelope(op) {
        calls.push({ method: 'recordRuleEnvelope', op });
        return { kind: 'false', reason: 'rr_denied' } as any;
      },
      permissionDenied(code, message, metadata) {
        calls.push({ method: 'permissionDenied', code, message, metadata });
        return deniedError;
      },
    });
  } catch (error) {
    actualError = error;
  }

  expect(actualError).toBe(deniedError);
  expect(calls).toEqual([
    { method: 'recordRuleEnvelope', op: 'create' },
    {
      method: 'permissionDenied',
      code: 'record_rule_denied',
      message: 'record rule denied',
      metadata: { model: 'auth.Role', op: 'create', reason: 'rr_denied' },
    },
  ]);
});

test('repository create write helpers permission denied metadata falls back to modelName and default reason', async () => {
  const deniedError = new Error('denied-modelName');
  let actualError: unknown;
  const calls: Array<Record<string, any>> = [];

  try {
    await ensureRepositoryCreateAllowed({
      meta: { fullModelName: '', modelName: 'ModelNameFallback', name: '' } as any,
      async getRecordRuleEnvelope() {
        return { kind: 'false' } as any;
      },
      permissionDenied(code, message, metadata) {
        calls.push({ code, message, metadata });
        return deniedError;
      },
    });
  } catch (error) {
    actualError = error;
  }

  expect(actualError).toBe(deniedError);
  expect(calls).toEqual([
    {
      code: 'record_rule_denied',
      message: 'record rule denied',
      metadata: { model: 'ModelNameFallback', op: 'create', reason: 'denied' },
    },
  ]);
});

test('repository create write helpers permission denied metadata falls back to name', async () => {
  const deniedError = new Error('denied-name');
  let actualError: unknown;
  const calls: Array<Record<string, any>> = [];

  try {
    await ensureRepositoryCreateAllowed({
      meta: { fullModelName: '', modelName: '', name: 'NameFallback' } as any,
      async getRecordRuleEnvelope() {
        return { kind: 'false', reason: 'explicit_reason' } as any;
      },
      permissionDenied(code, message, metadata) {
        calls.push({ code, message, metadata });
        return deniedError;
      },
    });
  } catch (error) {
    actualError = error;
  }

  expect(actualError).toBe(deniedError);
  expect(calls).toEqual([
    {
      code: 'record_rule_denied',
      message: 'record rule denied',
      metadata: { model: 'NameFallback', op: 'create', reason: 'explicit_reason' },
    },
  ]);
});

test('repository create write helpers prepare entities with generated ids, defaults, validation and encoding', async () => {
  const calls: Array<Record<string, any>> = [];
  const entities = await prepareRepositoryCreateEntities(
    {
      meta: { fields: new Map() } as any,
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
    },
    [{ Name: 'first' } as any, { Id: 'existing_id', Name: 'second' } as any]
  );

  expect(entities).toEqual([
    { Id: 'generated_id', Name: 'first', CompanyId: 'company_a', Encoded: true },
    { Id: 'existing_id', Name: 'second', CompanyId: 'company_a', Encoded: true },
  ]);
  expect(calls).toEqual([
    { method: 'fieldRule', payload: { Name: 'first' } },
    { method: 'fieldRule', payload: { Id: 'existing_id', Name: 'second' } },
    { method: 'generateId' },
    { method: 'defaultCompany', entity: { Id: 'generated_id', Name: 'first' } },
    { method: 'defaultCompany', entity: { Id: 'existing_id', Name: 'second' } },
    { method: 'validate', input: { Id: 'generated_id', Name: 'first', CompanyId: 'company_a' }, mode: 'create' },
    { method: 'validate', input: { Id: 'existing_id', Name: 'second', CompanyId: 'company_a' }, mode: 'create' },
    { method: 'encode', input: { Id: 'generated_id', Name: 'first', CompanyId: 'company_a' } },
    { method: 'encode', input: { Id: 'existing_id', Name: 'second', CompanyId: 'company_a' } },
  ]);
});

test('repository create write helpers prepare translated fields before encode', async () => {
  const encodedInputs: any[] = [];
  const entities = await withContext({ lang: 'zh_CN' }, () =>
    prepareRepositoryCreateEntities(
      {
        meta: {
          fields: new Map([['Name', { translate: true, column: { name: 'Name' }, storageHints: { size: 100 } }]]),
        } as any,
        async assertFieldRuleWriteAllowed() {},
        generateId() {
          return 'id_1';
        },
        applyDefaultCompanyIdOnCreate(entity) {
          return entity;
        },
        async validateFields() {},
        encodeForDb(input) {
          encodedInputs.push(input);
          return { ...input, Encoded: true } as any;
        },
      },
      [{ Name: '你好' } as any]
    )
  );

  expect(encodedInputs[0].Name).toEqual({ zh_CN: '你好', en_US: '你好' });
  expect(entities[0]).toEqual({ Id: 'id_1', Name: { zh_CN: '你好', en_US: '你好' }, Encoded: true });
});

test('repository create write helpers return empty prepared entities for empty input', async () => {
  const calls: Array<Record<string, any>> = [];
  const entities = await prepareRepositoryCreateEntities(
    {
      meta: { fields: new Map() } as any,
      async assertFieldRuleWriteAllowed(payload) {
        calls.push({ method: 'fieldRule', payload });
      },
      generateId() {
        calls.push({ method: 'generateId' });
        return 'generated_id';
      },
      applyDefaultCompanyIdOnCreate(entity) {
        calls.push({ method: 'defaultCompany', entity });
        return entity;
      },
      async validateFields(input, mode) {
        calls.push({ method: 'validate', input, mode });
      },
      encodeForDb(input) {
        calls.push({ method: 'encode', input });
        return input;
      },
    },
    []
  );

  expect(entities).toEqual([]);
  expect(calls).toEqual([]);
});

test('repository create write helpers treat undefined input as empty payload list', async () => {
  const calls: Array<Record<string, any>> = [];
  const entities = await prepareRepositoryCreateEntities(
    {
      meta: { fields: new Map() } as any,
      async assertFieldRuleWriteAllowed(payload) {
        calls.push({ method: 'fieldRule', payload });
      },
      generateId() {
        calls.push({ method: 'generateId' });
        return 'generated_id';
      },
      applyDefaultCompanyIdOnCreate(entity) {
        calls.push({ method: 'defaultCompany', entity });
        return entity;
      },
      async validateFields(input, mode) {
        calls.push({ method: 'validate', input, mode });
      },
      encodeForDb(input) {
        calls.push({ method: 'encode', input });
        return input;
      },
    },
    undefined as any
  );

  expect(entities).toEqual([]);
  expect(calls).toEqual([]);
});

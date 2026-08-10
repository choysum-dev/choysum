// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ensureRepositoryCreateAllowed, prepareRepositoryCreateEntities } from '../create_helpers';
import { withContext, withUser } from '../../../../runtime/context';

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

  expect(entities).toHaveLength(2);
  expect((entities[0] as any).Id).toBe('generated_id');
  expect((entities[0] as any).Name).toBe('first');
  expect((entities[0] as any).CompanyId).toBe('company_a');
  expect((entities[0] as any).Encoded).toBe(true);
  expect((entities[0] as any).CreatedAt instanceof Date).toBe(true);
  expect((entities[0] as any).UpdatedAt instanceof Date).toBe(true);
  expect((entities[1] as any).Id).toBe('existing_id');
  expect((entities[1] as any).Name).toBe('second');
  expect((entities[1] as any).CompanyId).toBe('company_a');
  expect((entities[1] as any).Encoded).toBe(true);
  expect((entities[1] as any).CreatedAt instanceof Date).toBe(true);
  expect((entities[1] as any).UpdatedAt instanceof Date).toBe(true);

  expect(calls.filter(c => c.method === 'fieldRule')).toEqual([
    { method: 'fieldRule', payload: { Name: 'first' } },
    { method: 'fieldRule', payload: { Id: 'existing_id', Name: 'second' } },
  ]);
  expect(calls.filter(c => c.method === 'generateId')).toEqual([{ method: 'generateId' }]);
  expect(calls.filter(c => c.method === 'defaultCompany')).toEqual([
    { method: 'defaultCompany', entity: { Id: 'generated_id', Name: 'first' } },
    { method: 'defaultCompany', entity: { Id: 'existing_id', Name: 'second' } },
  ]);
  const validates = calls.filter(c => c.method === 'validate');
  expect(validates).toHaveLength(2);
  expect(validates[0].mode).toBe('create');
  expect(validates[0].input.Id).toBe('generated_id');
  expect(validates[0].input.Name).toBe('first');
  expect(validates[0].input.CompanyId).toBe('company_a');
  expect(validates[0].input.CreatedAt instanceof Date).toBe(true);
  expect(validates[1].input.Id).toBe('existing_id');
  expect(validates[1].input.Name).toBe('second');
  const encodes = calls.filter(c => c.method === 'encode');
  expect(encodes).toHaveLength(2);
  expect(encodes[0].input.CreatedAt instanceof Date).toBe(true);
  expect(encodes[1].input.UpdatedAt instanceof Date).toBe(true);
});

test('repository create write helpers stamp monetary scales before validate/encode', async () => {
  const encodedInputs: any[] = [];
  const entities = await prepareRepositoryCreateEntities(
    {
      meta: {
        fields: new Map([
          ['CurrencyId', { type: 'ManyToOneRef', column: { name: 'CurrencyId' } }],
          ['Amount', { type: 'monetary', name: 'Amount', column: { name: 'Amount', currencyField: 'CurrencyId' } }],
        ]),
      } as any,
      async assertFieldRuleWriteAllowed() {},
      generateId() {
        return 'id_m';
      },
      applyDefaultCompanyIdOnCreate(entity) {
        return entity;
      },
      async validateFields(input) {
        expect((input as any).$dec$Amount__scale).toBe(0);
      },
      encodeForDb(input) {
        encodedInputs.push(input);
        return input as any;
      },
    },
    [{ Amount: '12.6', CurrencyId: { Id: 'C1', DecimalDigits: 0 } } as any]
  );

  expect(entities).toHaveLength(1);
  expect(encodedInputs[0].$dec$Amount__scale).toBe(0);
  expect(encodedInputs[0].Id).toBe('id_m');
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
  expect((entities[0] as any).Id).toBe('id_1');
  expect((entities[0] as any).Name).toEqual({ zh_CN: '你好', en_US: '你好' });
  expect((entities[0] as any).Encoded).toBe(true);
  expect((entities[0] as any).CreatedAt instanceof Date).toBe(true);
  expect((entities[0] as any).UpdatedAt instanceof Date).toBe(true);
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

function createPrepareDeps(overrides?: Record<string, unknown>) {
  return {
    meta: { fields: new Map() } as any,
    async assertFieldRuleWriteAllowed() {},
    generateId() {
      return 'id_1';
    },
    applyDefaultCompanyIdOnCreate(entity: any) {
      return entity;
    },
    async validateFields() {},
    encodeForDb(input: any) {
      return input;
    },
    ...overrides,
  };
}

test('repository create prepare stamps CreatedAt/UpdatedAt and actor uids before validate', async () => {
  let validated: any;
  const entities = await withUser('U-CREATE', async () =>
    prepareRepositoryCreateEntities(
      createPrepareDeps({
        async validateFields(input: any) {
          validated = input;
        },
      }) as any,
      [{ Name: 'demo' } as any]
    )
  );

  expect((entities[0] as any).CreatedAt instanceof Date).toBe(true);
  expect((entities[0] as any).UpdatedAt instanceof Date).toBe(true);
  expect((entities[0] as any).CreatedUid).toBe('U-CREATE');
  expect((entities[0] as any).UpdatedUid).toBe('U-CREATE');
  expect(validated.CreatedUid).toBe('U-CREATE');
  expect(validated.CreatedAt instanceof Date).toBe(true);
});

test('repository create prepare preserves explicit CreatedUid and leaves uids empty without actor', async () => {
  const withPreset = await withUser('U-ACTOR', async () =>
    prepareRepositoryCreateEntities(createPrepareDeps() as any, [
      { Name: 'preset', CreatedUid: 'U-PRESET', UpdatedUid: 'U-PRESET-U' } as any,
    ])
  );
  expect((withPreset[0] as any).CreatedUid).toBe('U-PRESET');
  expect((withPreset[0] as any).UpdatedUid).toBe('U-ACTOR');

  const noActor = await prepareRepositoryCreateEntities(createPrepareDeps() as any, [{ Name: 'anon' } as any]);
  expect((noActor[0] as any).CreatedUid).toBeUndefined();
  expect((noActor[0] as any).UpdatedUid).toBeUndefined();
  expect((noActor[0] as any).CreatedAt instanceof Date).toBe(true);
});

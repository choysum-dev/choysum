// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  createRepositoryCreateFacadeDeps,
  createRepositoryCreateMutationPayloadDeps,
  createRepositoryCreateWriteDeps,
  createRepositoryDeleteWriteDeps,
  createRepositoryMutationWriteFacadeDeps,
  createRepositoryMutationWriteConditionDeps,
  createRepositoryMutationWriteTargetDeps,
  createRepositoryUpdateFacadeDeps,
  createRepositoryUpdateMutationPayloadDeps,
  createRepositoryUpdateWriteDeps,
} from '..';

test('repository write deps delegate create write deps unchanged', async () => {
  const calls: Array<Record<string, any>> = [];
  const query = { kind: 'create' };
  const permissionError = new Error('permission-denied');
  const writeError = new Error('write-error');
  const deps = createRepositoryCreateWriteDeps({
    db: { name: 'db' },
    table: 'demo_table',
    meta: { fullModelName: 'auth.Role' } as any,
    async getRecordRuleEnvelope(op) {
      calls.push({ method: 'recordRuleEnvelope', op });
      return { kind: 'condition', condition: ['Id', '!=', null] } as any;
    },
    permissionDenied(code, message, metadata) {
      calls.push({ method: 'permissionDenied', code, message, metadata });
      return permissionError;
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
    async execute(input) {
      calls.push({ method: 'execute', input });
      return [{ Id: 'row_1' }] as any;
    },
    wrapSqlWriteError(error, mode) {
      calls.push({ method: 'wrap', error, mode });
      throw writeError;
    },
    async assertRecordRuleAllCreatedAllowed(createdIds, env) {
      calls.push({ method: 'recordRuleCreated', createdIds, env });
    },
  });

  expect(deps.db).toEqual({ name: 'db' });
  expect(deps.table).toBe('demo_table');
  expect(await deps.getRecordRuleEnvelope('create')).toEqual({ kind: 'condition', condition: ['Id', '!=', null] });
  expect(deps.permissionDenied('record_rule_denied', 'denied', { op: 'create' })).toBe(permissionError);
  await deps.assertFieldRuleWriteAllowed({ Name: 'demo' } as any);
  expect(deps.generateId()).toBe('generated_id');
  expect(deps.applyDefaultCompanyIdOnCreate({ Name: 'demo' } as any)).toEqual({ Name: 'demo', CompanyId: 'company_a' });
  await deps.validateFields({ Name: 'demo' } as any, 'create');
  expect(deps.encodeForDb({ Name: 'demo' } as any)).toEqual({ Name: 'demo', Encoded: true });
  expect(await deps.execute(query)).toEqual([{ Id: 'row_1' }]);
  let actualError: unknown;
  try {
    deps.wrapSqlWriteError(new Error('boom'), 'create');
  } catch (error) {
    actualError = error;
  }
  expect(actualError).toBe(writeError);
  await deps.assertRecordRuleAllCreatedAllowed(['row_1'], { kind: 'condition', condition: ['Id', '!=', null] } as any);
});

test('repository write deps delegate delete write deps unchanged', async () => {
  const calls: Array<Record<string, any>> = [];
  const query = { kind: 'delete' };
  const permissionError = new Error('write-error');
  const deps = createRepositoryDeleteWriteDeps({
    db: { name: 'db' },
    table: 'demo_table',
    meta: { fullModelName: 'auth.Role' } as any,
    softField: 'DeletedAt',
    async locateIdsForCondition(condition) {
      calls.push({ method: 'locate', condition });
      return ['row_1'];
    },
    async assertCompanyWriteAccessForCondition(condition) {
      calls.push({ method: 'company', condition });
      return ['row_1'];
    },
    async assertRecordRuleAllTargetsAllowed(op, targetIds) {
      calls.push({ method: 'recordRule', op, targetIds });
    },
    softDeleteEnabled() {
      calls.push({ method: 'softDeleteEnabled' });
      return true;
    },
    applySoftLayer(condition) {
      calls.push({ method: 'softLayer', condition });
      return { And: [condition, ['DeletedAt', 'is', null] as any] } as any;
    },
    async applyRecordRuleToCondition(condition, op) {
      calls.push({ method: 'applyRecordRule', condition, op });
      return { And: [condition, ['CompanyId', '=', 'company_a'] as any] } as any;
    },
    applyDefaultLayers(condition) {
      calls.push({ method: 'defaultLayers', condition });
      return condition;
    },
    isEmptyCondition(condition) {
      calls.push({ method: 'isEmpty', condition });
      return Array.isArray(condition) && condition.length === 0;
    },
    convertCondition(eb, condition, selfTable) {
      calls.push({ method: 'convert', eb, condition, selfTable });
      return { eb, condition, selfTable };
    },
    async execute(input) {
      calls.push({ method: 'execute', input });
      return [{ numDeletedRows: 1 }] as any;
    },
    invalidateCache() {
      calls.push({ method: 'invalidate' });
    },
    wrapSqlWriteError(error, mode) {
      calls.push({ method: 'wrap', error, mode });
      throw permissionError;
    },
    createRepository(meta) {
      calls.push({ method: 'createRepository', meta });
      return {
        softDeleteEnabled: () => false,
        delete: async () => [],
        hardDelete: async () => [],
        count: async () => 0,
        withFieldRuleBypass: async <T>(fn: () => Promise<T>) => await fn(),
        update: async () => [],
      };
    },
  });

  expect(deps.db).toEqual({ name: 'db' });
  expect(deps.table).toBe('demo_table');
  expect(deps.softField).toBe('DeletedAt');
  expect(await deps.locateIdsForCondition(['Id', '=', '1'] as any)).toEqual(['row_1']);
  expect(await deps.assertCompanyWriteAccessForCondition(['Id', '=', '1'] as any)).toEqual(['row_1']);
  await deps.assertRecordRuleAllTargetsAllowed('delete', ['row_1']);
  expect(deps.softDeleteEnabled()).toBe(true);
  expect(deps.applySoftLayer(['Id', '=', '1'] as any)).toEqual({
    And: [
      ['Id', '=', '1'],
      ['DeletedAt', 'is', null],
    ],
  });
  expect(await deps.applyRecordRuleToCondition(['Id', '=', '1'] as any, 'delete')).toEqual({
    And: [
      ['Id', '=', '1'],
      ['CompanyId', '=', 'company_a'],
    ],
  });
  expect(deps.applyDefaultLayers(['Id', '=', '1'] as any)).toEqual(['Id', '=', '1']);
  expect(deps.isEmptyCondition([] as any)).toBe(true);
  expect(deps.convertCondition('EB', ['Id', '=', '1'] as any, 'demo_table')).toEqual({
    eb: 'EB',
    condition: ['Id', '=', '1'],
    selfTable: 'demo_table',
  });
  expect(await deps.execute(query)).toEqual([{ numDeletedRows: 1 }]);
  deps.invalidateCache();
  let actualError: unknown;
  try {
    deps.wrapSqlWriteError(new Error('boom'), 'update');
  } catch (error) {
    actualError = error;
  }
  expect(actualError).toBe(permissionError);
  const childRepo = deps.createRepository({ fullModelName: 'demo.Model' } as any);
  expect(typeof childRepo.softDeleteEnabled).toBe('function');
  expect(typeof childRepo.delete).toBe('function');
});

test('repository write deps delegate update write deps unchanged', async () => {
  const calls: Array<Record<string, any>> = [];
  const query = { kind: 'update' };
  const deps = createRepositoryUpdateWriteDeps({
    db: { name: 'db' },
    table: 'demo_table',
    meta: { fullModelName: 'auth.Role' } as any,
    getScalarFields(meta) {
      calls.push({ method: 'scalarFields', meta });
      return ['Id', 'Name'];
    },
    makeSelectCtx(builder, selfTable, curMeta) {
      calls.push({ method: 'selectCtx', builder, selfTable, curMeta });
      return { builder, selfTable, curMeta };
    },
    aliasSelection(selection, alias) {
      calls.push({ method: 'alias', selection, alias });
      return { selection, alias };
    },
    applySoftLayer(condition) {
      calls.push({ method: 'softLayer', condition });
      return condition;
    },
    isEmptyCondition(condition) {
      calls.push({ method: 'isEmpty', condition });
      return false;
    },
    convertCondition(eb, condition, selfTable) {
      calls.push({ method: 'convert', eb, condition, selfTable });
      return { eb, condition, selfTable };
    },
    async execute(input) {
      calls.push({ method: 'execute', input });
      return [{ Id: 'row_1' }] as any;
    },
    decodeFromDb(row) {
      calls.push({ method: 'decode', row });
      return { ...row, Decoded: true } as any;
    },
    async assertCompanyWriteAccessForCondition(condition) {
      calls.push({ method: 'company', condition });
      return ['row_1'];
    },
    async locateIdsForCondition(condition) {
      calls.push({ method: 'locate', condition });
      return ['row_1'];
    },
    async assertRecordRuleAllTargetsAllowed(op, targetIds) {
      calls.push({ method: 'recordRule', op, targetIds });
    },
    async assertFieldRuleWriteAllowed(payload) {
      calls.push({ method: 'fieldRule', payload });
    },
    applyDefaultCompanyIdOnUpdate(vals) {
      calls.push({ method: 'defaultCompany', vals });
      return { ...vals, CompanyId: 'company_a' };
    },
    async validateFields(input, mode, current) {
      calls.push({ method: 'validate', input, mode, current });
    },
    encodeForDb(input) {
      calls.push({ method: 'encode', input });
      return { ...input, Encoded: true } as any;
    },
    async applyRecordRuleToCondition(condition, op) {
      calls.push({ method: 'applyRecordRule', condition, op });
      return condition;
    },
    applyDefaultLayers(condition) {
      calls.push({ method: 'defaultLayers', condition });
      return condition;
    },
    invalidateCache() {
      calls.push({ method: 'invalidate' });
    },
  });

  expect(deps.db).toEqual({ name: 'db' });
  expect(deps.table).toBe('demo_table');
  expect(deps.getScalarFields({ fullModelName: 'auth.Role' } as any)).toEqual(['Id', 'Name']);
  expect(deps.makeSelectCtx('QB', 'demo_table', { fullModelName: 'auth.Role' } as any)).toEqual({
    builder: 'QB',
    selfTable: 'demo_table',
    curMeta: { fullModelName: 'auth.Role' },
  });
  expect(deps.aliasSelection('SEL', 'Name')).toEqual({ selection: 'SEL', alias: 'Name' });
  expect(deps.applySoftLayer(['Id', '=', '1'] as any)).toEqual(['Id', '=', '1']);
  expect(deps.isEmptyCondition(['Id', '=', '1'] as any)).toBe(false);
  expect(deps.convertCondition('EB', ['Id', '=', '1'] as any, 'demo_table')).toEqual({
    eb: 'EB',
    condition: ['Id', '=', '1'],
    selfTable: 'demo_table',
  });
  expect(await deps.execute(query)).toEqual([{ Id: 'row_1' }]);
  expect(deps.decodeFromDb({ Id: 'row_1' } as any)).toEqual({ Id: 'row_1', Decoded: true });
  expect(await deps.assertCompanyWriteAccessForCondition(['Id', '=', '1'] as any)).toEqual(['row_1']);
  expect(await deps.locateIdsForCondition(['Id', '=', '1'] as any)).toEqual(['row_1']);
  await deps.assertRecordRuleAllTargetsAllowed('write', ['row_1']);
  await deps.assertFieldRuleWriteAllowed({ Name: 'demo' } as any);
  expect(deps.applyDefaultCompanyIdOnUpdate({ Name: 'demo' } as any)).toEqual({ Name: 'demo', CompanyId: 'company_a' });
  await deps.validateFields({ Name: 'demo' } as any, 'update', { Id: 'row_1' });
  expect(deps.encodeForDb({ Name: 'demo' } as any)).toEqual({ Name: 'demo', Encoded: true });
  expect(await deps.applyRecordRuleToCondition(['Id', '=', '1'] as any, 'write')).toEqual(['Id', '=', '1']);
  expect(deps.applyDefaultLayers(['Id', '=', '1'] as any)).toEqual(['Id', '=', '1']);
  deps.invalidateCache();
});

test('repository write deps delegate mutation target deps unchanged', async () => {
  const calls: Array<Record<string, any>> = [];
  const deps = createRepositoryMutationWriteTargetDeps({
    meta: { fullModelName: 'auth.Role', companyField: 'CompanyId', fields: new Map([['CompanyId', {}]]) } as any,
    async locateIdsForCondition(condition) {
      calls.push({ method: 'locate', condition });
      return ['row_1'];
    },
    async assertCompanyWriteAccessForCondition(condition) {
      calls.push({ method: 'company', condition });
      return ['row_2'];
    },
    async assertRecordRuleAllTargetsAllowed(op, targetIds) {
      calls.push({ method: 'recordRule', op, targetIds });
    },
  });

  expect(deps.meta).toEqual({
    fullModelName: 'auth.Role',
    companyField: 'CompanyId',
    fields: new Map([['CompanyId', {}]]),
  });
  expect(await deps.locateIdsForCondition(['Id', '=', '1'] as any)).toEqual(['row_1']);
  expect(await deps.assertCompanyWriteAccessForCondition(['Id', '=', '2'] as any)).toEqual(['row_2']);
  await deps.assertRecordRuleAllTargetsAllowed('write', ['row_1']);
  expect(calls).toEqual([
    { method: 'locate', condition: ['Id', '=', '1'] },
    { method: 'company', condition: ['Id', '=', '2'] },
    { method: 'recordRule', op: 'write', targetIds: ['row_1'] },
  ]);
});

test('repository write deps delegate mutation condition deps unchanged', async () => {
  const calls: Array<Record<string, any>> = [];
  const deps = createRepositoryMutationWriteConditionDeps({
    table: 'demo_table',
    async applyRecordRuleToCondition(condition, op) {
      calls.push({ method: 'recordRule', condition, op });
      return { And: [condition, ['CompanyId', '=', 'company_a'] as any] } as any;
    },
    applyDefaultLayers(condition) {
      calls.push({ method: 'defaultLayers', condition });
      return { And: [condition, ['DeletedAt', 'is', null] as any] } as any;
    },
    isEmptyCondition(condition) {
      calls.push({ method: 'isEmpty', condition });
      return false;
    },
    convertCondition(eb, condition, selfTable) {
      calls.push({ method: 'convert', eb, condition, selfTable });
      return { eb, condition, selfTable };
    },
  });

  expect(deps.table).toBe('demo_table');
  expect(await deps.applyRecordRuleToCondition(['Id', '=', '1'] as any, 'delete')).toEqual({
    And: [
      ['Id', '=', '1'],
      ['CompanyId', '=', 'company_a'],
    ],
  });
  expect(deps.applyDefaultLayers(['Id', '=', '1'] as any)).toEqual({
    And: [
      ['Id', '=', '1'],
      ['DeletedAt', 'is', null],
    ],
  });
  expect(deps.isEmptyCondition(['Id', '=', '1'] as any)).toBe(false);
  expect(deps.convertCondition('EB', ['Id', '=', '1'] as any, 'demo_table')).toEqual({
    eb: 'EB',
    condition: ['Id', '=', '1'],
    selfTable: 'demo_table',
  });
  expect(calls).toEqual([
    { method: 'recordRule', condition: ['Id', '=', '1'], op: 'delete' },
    { method: 'defaultLayers', condition: ['Id', '=', '1'] },
    { method: 'isEmpty', condition: ['Id', '=', '1'] },
    { method: 'convert', eb: 'EB', condition: ['Id', '=', '1'], selfTable: 'demo_table' },
  ]);
});

test('repository write deps facade merges mutation target and condition deps unchanged', async () => {
  const calls: Array<Record<string, any>> = [];
  const deps = createRepositoryMutationWriteFacadeDeps({
    meta: { fullModelName: 'auth.Role', companyField: 'CompanyId', fields: new Map([['CompanyId', {}]]) } as any,
    table: 'demo_table',
    async locateIdsForCondition(condition) {
      calls.push({ method: 'locate', condition });
      return ['row_1'];
    },
    async assertCompanyWriteAccessForCondition(condition) {
      calls.push({ method: 'company', condition });
      return ['row_2'];
    },
    async assertRecordRuleAllTargetsAllowed(op, targetIds) {
      calls.push({ method: 'recordRule', op, targetIds });
    },
    async applyRecordRuleToCondition(condition, op) {
      calls.push({ method: 'applyRecordRule', condition, op });
      return condition;
    },
    applyDefaultLayers(condition) {
      calls.push({ method: 'defaultLayers', condition });
      return condition;
    },
    isEmptyCondition(condition) {
      calls.push({ method: 'isEmpty', condition });
      return false;
    },
    convertCondition(eb, condition, selfTable) {
      calls.push({ method: 'convert', eb, condition, selfTable });
      return { eb, condition, selfTable };
    },
  });

  expect(deps.meta).toEqual({
    fullModelName: 'auth.Role',
    companyField: 'CompanyId',
    fields: new Map([['CompanyId', {}]]),
  });
  expect(deps.table).toBe('demo_table');
  expect(await deps.locateIdsForCondition(['Id', '=', '1'] as any)).toEqual(['row_1']);
  expect(await deps.assertCompanyWriteAccessForCondition(['Id', '=', '2'] as any)).toEqual(['row_2']);
  await deps.assertRecordRuleAllTargetsAllowed('write', ['row_1']);
  expect(await deps.applyRecordRuleToCondition(['Id', '=', '1'] as any, 'write')).toEqual(['Id', '=', '1']);
  expect(deps.applyDefaultLayers(['Id', '=', '1'] as any)).toEqual(['Id', '=', '1']);
  expect(deps.isEmptyCondition(['Id', '=', '1'] as any)).toBe(false);
  expect(deps.convertCondition('EB', ['Id', '=', '1'] as any, 'demo_table')).toEqual({
    eb: 'EB',
    condition: ['Id', '=', '1'],
    selfTable: 'demo_table',
  });
  expect(calls).toEqual([
    { method: 'locate', condition: ['Id', '=', '1'] },
    { method: 'company', condition: ['Id', '=', '2'] },
    { method: 'recordRule', op: 'write', targetIds: ['row_1'] },
    { method: 'applyRecordRule', condition: ['Id', '=', '1'], op: 'write' },
    { method: 'defaultLayers', condition: ['Id', '=', '1'] },
    { method: 'isEmpty', condition: ['Id', '=', '1'] },
    { method: 'convert', eb: 'EB', condition: ['Id', '=', '1'], selfTable: 'demo_table' },
  ]);
});

test('repository write create facade merges create assembler and payload deps unchanged', async () => {
  const calls: Array<Record<string, any>> = [];
  const deps = createRepositoryCreateFacadeDeps({
    db: { name: 'db' },
    table: 'demo_table',
    meta: { fullModelName: 'auth.Role' } as any,
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
    async execute(input) {
      calls.push({ method: 'execute', input });
      return [{ Id: 'row_1' }] as any;
    },
    wrapSqlWriteError(error, mode) {
      calls.push({ method: 'wrap', error, mode });
      throw error;
    },
    async assertRecordRuleAllCreatedAllowed(createdIds, env) {
      calls.push({ method: 'recordRuleCreated', createdIds, env });
    },
  });

  expect(deps.db).toEqual({ name: 'db' });
  expect(await deps.getRecordRuleEnvelope('create')).toEqual({ kind: 'condition', condition: ['Id', '!=', null] });
  await deps.assertFieldRuleWriteAllowed({ Name: 'demo' } as any);
  expect(deps.generateId()).toBe('generated_id');
  expect(deps.applyDefaultCompanyIdOnCreate({ Name: 'demo' } as any)).toEqual({ Name: 'demo', CompanyId: 'company_a' });
  await deps.validateFields({ Name: 'demo' } as any, 'create');
  expect(deps.encodeForDb({ Name: 'demo' } as any)).toEqual({ Name: 'demo', Encoded: true });
  expect(await deps.execute({ kind: 'create' })).toEqual([{ Id: 'row_1' }]);
  await deps.assertRecordRuleAllCreatedAllowed(['row_1'], { kind: 'condition', condition: ['Id', '!=', null] } as any);
});

test('repository write update facade merges update assembler payload and mutation facade deps unchanged', async () => {
  const calls: Array<Record<string, any>> = [];
  const deps = createRepositoryUpdateFacadeDeps({
    db: { name: 'db' },
    table: 'demo_table',
    meta: { fullModelName: 'auth.Role' } as any,
    getScalarFields(meta) {
      calls.push({ method: 'scalarFields', meta });
      return ['Id', 'Name'];
    },
    makeSelectCtx(builder, selfTable, curMeta) {
      calls.push({ method: 'selectCtx', builder, selfTable, curMeta });
      return { builder, selfTable, curMeta };
    },
    aliasSelection(selection, alias) {
      calls.push({ method: 'alias', selection, alias });
      return { selection, alias };
    },
    applySoftLayer(condition) {
      calls.push({ method: 'softLayer', condition });
      return condition;
    },
    async execute(input) {
      calls.push({ method: 'execute', input });
      return [{ Id: 'row_1' }] as any;
    },
    decodeFromDb(row) {
      calls.push({ method: 'decode', row });
      return { ...row, Decoded: true } as any;
    },
    invalidateCache() {
      calls.push({ method: 'invalidate' });
    },
    locateIdsForCondition(condition) {
      calls.push({ method: 'locate', condition });
      return Promise.resolve(['row_1']);
    },
    assertCompanyWriteAccessForCondition(condition) {
      calls.push({ method: 'company', condition });
      return Promise.resolve(['row_1']);
    },
    assertRecordRuleAllTargetsAllowed(op, targetIds) {
      calls.push({ method: 'recordRule', op, targetIds });
      return Promise.resolve();
    },
    applyRecordRuleToCondition(condition, op) {
      calls.push({ method: 'applyRecordRule', condition, op });
      return Promise.resolve(condition);
    },
    applyDefaultLayers(condition) {
      calls.push({ method: 'defaultLayers', condition });
      return condition;
    },
    isEmptyCondition(condition) {
      calls.push({ method: 'isEmpty', condition });
      return false;
    },
    convertCondition(eb, condition, selfTable) {
      calls.push({ method: 'convert', eb, condition, selfTable });
      return { eb, condition, selfTable };
    },
    async assertFieldRuleWriteAllowed(payload) {
      calls.push({ method: 'fieldRule', payload });
    },
    applyDefaultCompanyIdOnUpdate(vals) {
      calls.push({ method: 'defaultCompany', vals });
      return { ...vals, CompanyId: 'company_a' } as any;
    },
    async validateFields(input, mode, current) {
      calls.push({ method: 'validate', input, mode, current });
    },
    encodeForDb(input) {
      calls.push({ method: 'encode', input });
      return { ...input, Encoded: true } as any;
    },
  });

  expect(deps.table).toBe('demo_table');
  expect(deps.getScalarFields({ fullModelName: 'auth.Role' } as any)).toEqual(['Id', 'Name']);
  expect(deps.makeSelectCtx('QB', 'demo_table', { fullModelName: 'auth.Role' } as any)).toEqual({
    builder: 'QB',
    selfTable: 'demo_table',
    curMeta: { fullModelName: 'auth.Role' },
  });
  expect(deps.aliasSelection('SEL', 'Name')).toEqual({ selection: 'SEL', alias: 'Name' });
  expect(deps.applySoftLayer(['Id', '=', '1'] as any)).toEqual(['Id', '=', '1']);
  expect(await deps.locateIdsForCondition(['Id', '=', '1'] as any)).toEqual(['row_1']);
  expect(await deps.assertCompanyWriteAccessForCondition(['Id', '=', '1'] as any)).toEqual(['row_1']);
  await deps.assertRecordRuleAllTargetsAllowed('write', ['row_1']);
  expect(await deps.applyRecordRuleToCondition(['Id', '=', '1'] as any, 'write')).toEqual(['Id', '=', '1']);
  expect(deps.applyDefaultLayers(['Id', '=', '1'] as any)).toEqual(['Id', '=', '1']);
  expect(deps.isEmptyCondition(['Id', '=', '1'] as any)).toBe(false);
  expect(deps.convertCondition('EB', ['Id', '=', '1'] as any, 'demo_table')).toEqual({ eb: 'EB', condition: ['Id', '=', '1'], selfTable: 'demo_table' });
  await deps.assertFieldRuleWriteAllowed({ Name: 'demo' } as any);
  expect(deps.applyDefaultCompanyIdOnUpdate({ Name: 'demo' } as any)).toEqual({ Name: 'demo', CompanyId: 'company_a' });
  await deps.validateFields({ Name: 'demo' } as any, 'update', { Id: 'row_1' });
  expect(deps.encodeForDb({ Name: 'demo' } as any)).toEqual({ Name: 'demo', Encoded: true });
});

test('repository write deps delegate create and update payload deps unchanged', async () => {
  const createCalls: Array<Record<string, any>> = [];
  const createDeps = createRepositoryCreateMutationPayloadDeps({
    async assertFieldRuleWriteAllowed(payload) {
      createCalls.push({ method: 'fieldRule', payload });
    },
    applyDefaultCompanyIdOnCreate(entity) {
      createCalls.push({ method: 'defaultCreate', entity });
      return { ...entity, CompanyId: 'company_a' } as any;
    },
    async validateFields(input, mode) {
      createCalls.push({ method: 'validate', input, mode });
    },
    encodeForDb(input) {
      createCalls.push({ method: 'encode', input });
      return { ...input, Encoded: true } as any;
    },
  });

  await createDeps.assertFieldRuleWriteAllowed({ Name: 'demo' } as any);
  expect(createDeps.applyDefaultCompanyIdOnCreate({ Name: 'demo' } as any)).toEqual({ Name: 'demo', CompanyId: 'company_a' });
  await createDeps.validateFields({ Name: 'demo' } as any, 'create');
  expect(createDeps.encodeForDb({ Name: 'demo' } as any)).toEqual({ Name: 'demo', Encoded: true });
  expect(createCalls).toEqual([
    { method: 'fieldRule', payload: { Name: 'demo' } },
    { method: 'defaultCreate', entity: { Name: 'demo' } },
    { method: 'validate', input: { Name: 'demo' }, mode: 'create' },
    { method: 'encode', input: { Name: 'demo' } },
  ]);

  const updateCalls: Array<Record<string, any>> = [];
  const updateDeps = createRepositoryUpdateMutationPayloadDeps({
    async assertFieldRuleWriteAllowed(payload) {
      updateCalls.push({ method: 'fieldRule', payload });
    },
    applyDefaultCompanyIdOnUpdate(vals) {
      updateCalls.push({ method: 'defaultUpdate', vals });
      return { ...vals, CompanyId: 'company_a' } as any;
    },
    async validateFields(input, mode, current) {
      updateCalls.push({ method: 'validate', input, mode, current });
    },
    encodeForDb(input) {
      updateCalls.push({ method: 'encode', input });
      return { ...input, Encoded: true } as any;
    },
  });

  await updateDeps.assertFieldRuleWriteAllowed({ Name: 'demo' } as any);
  expect(updateDeps.applyDefaultCompanyIdOnUpdate({ Name: 'demo' } as any)).toEqual({ Name: 'demo', CompanyId: 'company_a' });
  await updateDeps.validateFields({ Name: 'demo' } as any, 'update', { Id: 'row_1' });
  expect(updateDeps.encodeForDb({ Name: 'demo' } as any)).toEqual({ Name: 'demo', Encoded: true });
  expect(updateCalls).toEqual([
    { method: 'fieldRule', payload: { Name: 'demo' } },
    { method: 'defaultUpdate', vals: { Name: 'demo' } },
    { method: 'validate', input: { Name: 'demo' }, mode: 'update', current: { Id: 'row_1' } },
    { method: 'encode', input: { Name: 'demo' } },
  ]);
});

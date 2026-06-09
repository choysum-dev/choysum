// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Repository } from '../../repository';

function createRepositoryHarness() {
  return new Repository({
    fullModelName: 'demo.Model',
    tableName: () => 'demo_table',
  } as any) as any;
}

test('repository write assembler composes delete deps from target soft-delete-prewrite query-prepare runtime and post-write groups', () => {
  const repository = createRepositoryHarness();
  const calls: string[] = [];
  const target = {
    meta: { fullModelName: 'demo.Model' } as any,
    locateIdsForCondition: async () => ['row_1'],
    assertCompanyWriteAccessForCondition: async () => ['row_1'],
    assertRecordRuleAllTargetsAllowed: async () => undefined,
  };
  const softDeletePreWrite = {
    softField: 'DeletedAt',
    softDeleteEnabled: () => true,
    applySoftLayer: (condition: any) => condition,
    createRepository: () => ({ kind: 'child' }),
  };
  const queryPrepare = {
    db: { name: 'db' },
    table: 'demo_table',
    applyRecordRuleToCondition: async (condition: any) => condition,
    applyDefaultLayers: (condition: any) => condition,
    isEmptyCondition: () => false,
    convertCondition: () => ({ kind: 'converted' }),
  };
  const runtime = {
    execute: async () => [],
    wrapSqlWriteError: () => {
      throw new Error('write');
    },
  };
  const postWrite = {
    invalidateCache: () => undefined,
  };

  repository.createDeleteWriteTargetDeps = () => {
    calls.push('target');
    return target;
  };
  repository.createDeleteWriteSoftDeletePreWriteDeps = () => {
    calls.push('softDeletePreWrite');
    return softDeletePreWrite;
  };
  repository.createDeleteWriteQueryPrepareDeps = () => {
    calls.push('queryPrepare');
    return queryPrepare;
  };
  repository.createDeleteWriteRuntimeDeps = () => {
    calls.push('runtime');
    return runtime;
  };
  repository.createDeleteWritePostWriteDeps = () => {
    calls.push('postWrite');
    return postWrite;
  };

  const deps = repository.createDeleteWriteDeps();

  expect(calls).toEqual(['target', 'softDeletePreWrite', 'queryPrepare', 'runtime', 'postWrite']);
  expect(deps.db).toBe(queryPrepare.db);
  expect(deps.softField).toBe('DeletedAt');
  expect(deps.meta).toBe(target.meta);
  expect(deps.table).toBe(queryPrepare.table);
  expect(deps.locateIdsForCondition).toBe(target.locateIdsForCondition);
  expect(deps.assertCompanyWriteAccessForCondition).toBe(target.assertCompanyWriteAccessForCondition);
  expect(deps.assertRecordRuleAllTargetsAllowed).toBe(target.assertRecordRuleAllTargetsAllowed);
  expect(deps.applyRecordRuleToCondition).toBe(queryPrepare.applyRecordRuleToCondition);
  expect(deps.applyDefaultLayers).toBe(queryPrepare.applyDefaultLayers);
  expect(deps.isEmptyCondition).toBe(queryPrepare.isEmptyCondition);
  expect(deps.convertCondition).toBe(queryPrepare.convertCondition);
  expect(deps.softDeleteEnabled).toBe(softDeletePreWrite.softDeleteEnabled);
  expect(deps.applySoftLayer).toBe(softDeletePreWrite.applySoftLayer);
  expect(deps.execute).toBe(runtime.execute);
  expect(deps.invalidateCache).toBe(postWrite.invalidateCache);
  expect(deps.wrapSqlWriteError).toBe(runtime.wrapSqlWriteError);
  expect(deps.createRepository).toBe(softDeletePreWrite.createRepository);
});

test('repository write assembler composes update deps from projection soft-filter runtime post-write mutation and payload groups', () => {
  const repository = createRepositoryHarness();
  const calls: string[] = [];
  const projection = {
    getScalarFields: () => ['Id'],
    makeSelectCtx: () => ({ kind: 'ctx' }),
    aliasSelection: () => ({ kind: 'alias' }),
    decodeFromDb: (row: any) => row,
  };
  const softFilter = {
    applySoftLayer: (condition: any) => condition,
  };
  const runtime = {
    execute: async () => [],
  };
  const postWrite = {
    invalidateCache: () => undefined,
  };
  const mutationFacade = {
    meta: { fullModelName: 'demo.Model' } as any,
    table: 'demo_table',
    locateIdsForCondition: async () => ['row_1'],
    assertCompanyWriteAccessForCondition: async () => ['row_1'],
    assertRecordRuleAllTargetsAllowed: async () => undefined,
    applyRecordRuleToCondition: async (condition: any) => condition,
    applyDefaultLayers: (condition: any) => condition,
    isEmptyCondition: () => false,
    convertCondition: () => ({ kind: 'converted' }),
  };
  const payload = {
    assertFieldRuleWriteAllowed: async () => undefined,
    applyDefaultCompanyIdOnUpdate: (entity: any) => entity,
    validateFields: async () => undefined,
    encodeForDb: (entity: any) => entity,
  };

  repository.createUpdateWriteProjectionDeps = () => {
    calls.push('projection');
    return projection;
  };
  repository.createUpdateWriteSoftFilterDeps = () => {
    calls.push('softFilter');
    return softFilter;
  };
  repository.createUpdateWriteRuntimeDeps = () => {
    calls.push('runtime');
    return runtime;
  };
  repository.createUpdateWritePostWriteDeps = () => {
    calls.push('postWrite');
    return postWrite;
  };
  repository.createMutationWriteFacadeDeps = () => {
    calls.push('mutation');
    return mutationFacade;
  };
  repository.createUpdateWritePayloadDeps = () => {
    calls.push('payload');
    return payload;
  };

  const deps = repository.createUpdateWriteDeps();

  expect(calls).toEqual(['projection', 'softFilter', 'runtime', 'postWrite', 'mutation', 'payload']);
  expect(deps.db).toBe(repository.db);
  expect(deps.meta).toBe(mutationFacade.meta);
  expect(deps.table).toBe(mutationFacade.table);
  expect(deps.getScalarFields).toBe(projection.getScalarFields);
  expect(deps.makeSelectCtx).toBe(projection.makeSelectCtx);
  expect(deps.aliasSelection).toBe(projection.aliasSelection);
  expect(deps.decodeFromDb).toBe(projection.decodeFromDb);
  expect(deps.applySoftLayer).toBe(softFilter.applySoftLayer);
  expect(deps.execute).toBe(runtime.execute);
  expect(deps.invalidateCache).toBe(postWrite.invalidateCache);
  expect(deps.locateIdsForCondition).toBe(mutationFacade.locateIdsForCondition);
  expect(deps.assertCompanyWriteAccessForCondition).toBe(mutationFacade.assertCompanyWriteAccessForCondition);
  expect(deps.assertRecordRuleAllTargetsAllowed).toBe(mutationFacade.assertRecordRuleAllTargetsAllowed);
  expect(deps.applyRecordRuleToCondition).toBe(mutationFacade.applyRecordRuleToCondition);
  expect(deps.applyDefaultLayers).toBe(mutationFacade.applyDefaultLayers);
  expect(deps.isEmptyCondition).toBe(mutationFacade.isEmptyCondition);
  expect(deps.convertCondition).toBe(mutationFacade.convertCondition);
  expect(deps.assertFieldRuleWriteAllowed).toBe(payload.assertFieldRuleWriteAllowed);
  expect(deps.applyDefaultCompanyIdOnUpdate).toBe(payload.applyDefaultCompanyIdOnUpdate);
  expect(deps.validateFields).toBe(payload.validateFields);
  expect(deps.encodeForDb).toBe(payload.encodeForDb);
});

test('repository write assembler composes create deps from authz payload runtime and post-write groups', () => {
  const repository = createRepositoryHarness();
  const calls: string[] = [];
  const authz = {
    getRecordRuleEnvelope: async () => ({ kind: 'condition', condition: ['Id', '!=', null] }),
    permissionDenied: () => new Error('permission'),
  };
  const postWrite = {
    assertRecordRuleAllCreatedAllowed: async () => undefined,
  };
  const payload = {
    assertFieldRuleWriteAllowed: async () => undefined,
    applyDefaultCompanyIdOnCreate: (entity: any) => entity,
    validateFields: async () => undefined,
    encodeForDb: (entity: any) => entity,
  };
  const runtime = {
    generateId: () => 'generated_id',
    execute: async () => [],
    wrapSqlWriteError: () => {
      throw new Error('write');
    },
  };

  repository.createCreateWriteAuthzDeps = () => {
    calls.push('authz');
    return authz;
  };
  repository.createCreateWritePayloadDeps = () => {
    calls.push('payload');
    return payload;
  };
  repository.createCreateWriteRuntimeDeps = () => {
    calls.push('runtime');
    return runtime;
  };
  repository.createCreateWritePostWriteDeps = () => {
    calls.push('postWrite');
    return postWrite;
  };

  const deps = repository.createCreateWriteDeps();

  expect(calls).toEqual(['authz', 'payload', 'runtime', 'postWrite']);
  expect(deps.db).toBe(repository.db);
  expect(deps.table).toBe('demo_table');
  expect(deps.meta).toBe(repository.meta);
  expect(deps.getRecordRuleEnvelope).toBe(authz.getRecordRuleEnvelope);
  expect(deps.permissionDenied).toBe(authz.permissionDenied);
  expect(deps.assertRecordRuleAllCreatedAllowed).toBe(postWrite.assertRecordRuleAllCreatedAllowed);
  expect(deps.assertFieldRuleWriteAllowed).toBe(payload.assertFieldRuleWriteAllowed);
  expect(deps.applyDefaultCompanyIdOnCreate).toBe(payload.applyDefaultCompanyIdOnCreate);
  expect(deps.validateFields).toBe(payload.validateFields);
  expect(deps.encodeForDb).toBe(payload.encodeForDb);
  expect(deps.generateId).toBe(runtime.generateId);
  expect(deps.execute).toBe(runtime.execute);
  expect(deps.wrapSqlWriteError).toBe(runtime.wrapSqlWriteError);
});

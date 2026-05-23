// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import type { RepositoryPermissionDeniedFn } from '../authz/types';
import type {
  BaseQueryCondition,
  ConditionEnvelope,
  Entity,
  RepositoryExecute,
  RepositoryGetScalarFieldsDepsLike,
  RepositoryMutationPayloadGuardEncodeDepsLike,
  RepositoryMutationPayloadValidateDepsLike,
  RepositoryRecordRuleConditionPipelineDepsLike,
  RepositorySelectionAliaserLike,
  RepositorySelectCtxFactoryLike,
  RepositorySoftConditionPipelineDepsLike,
} from '../types';
import type { RepositoryDeleteChild } from './delete_child_factory';
import type { ObjectRecord } from '../../../../utils/types';

type RepositoryDbLike = unknown;
type RepositorySelectCtxFactory = RepositorySelectCtxFactoryLike<ModelMetadata>;
type RepositorySelectionAliaser = RepositorySelectionAliaserLike;

type RepositoryCreateMutationPayloadDepsParams = RepositoryMutationPayloadGuardEncodeDepsLike<Entity> &
  RepositoryMutationPayloadValidateDepsLike<Entity, 'create'> & {
    applyDefaultCompanyIdOnCreate: (entity: Entity) => Entity;
  };

type RepositoryUpdateMutationPayloadDepsParams = RepositoryMutationPayloadGuardEncodeDepsLike<Entity> &
  RepositoryMutationPayloadValidateDepsLike<Entity, 'update', ObjectRecord> & {
    applyDefaultCompanyIdOnUpdate: (vals: Entity) => Entity;
  };

type RepositoryMutationWriteTargetDepsParams<TOp extends 'delete' | 'write'> = {
  meta: ModelMetadata;
  locateIdsForCondition: (condition: BaseQueryCondition) => Promise<string[]>;
  assertCompanyWriteAccessForCondition: (condition: BaseQueryCondition) => Promise<string[]>;
  assertRecordRuleAllTargetsAllowed: (op: TOp, targetIds: string[]) => Promise<void>;
};

type RepositoryMutationWriteConditionDepsParams<TOp extends 'delete' | 'write'> = {
  table: string;
} & RepositoryRecordRuleConditionPipelineDepsLike<TOp, BaseQueryCondition>;

type RepositoryMutationWriteFacadeDepsParams<TOp extends 'delete' | 'write'> = RepositoryMutationWriteTargetDepsParams<TOp> &
  RepositoryMutationWriteConditionDepsParams<TOp>;

type RepositoryCreateFacadeDepsParams = RepositoryCreateWriteDepsParams;

type RepositoryUpdateFacadeDepsParams = RepositoryUpdateWriteDepsParams;

type RepositoryCreateWriteDepsParams = {
  db: RepositoryDbLike;
  table: string;
  meta: ModelMetadata;
  getRecordRuleEnvelope: (op: 'create') => Promise<ConditionEnvelope>;
  permissionDenied: RepositoryPermissionDeniedFn;
  generateId: () => string;
  execute: RepositoryExecute;
  wrapSqlWriteError: (error: unknown, mode: 'create') => never;
  assertRecordRuleAllCreatedAllowed: (createdIds: string[], env: ConditionEnvelope) => Promise<void>;
  recomputePersistForCreate?: (createdIds: string[], sanitizedEntities: Entity[]) => Promise<void>;
} & RepositoryCreateMutationPayloadDepsParams;

type RepositoryDeleteWriteDepsParams = RepositoryMutationWriteFacadeDepsParams<'delete'> &
  RepositorySoftConditionPipelineDepsLike<BaseQueryCondition> & {
    db: RepositoryDbLike;
    softField: string;
    softDeleteEnabled: () => boolean;
    execute: RepositoryExecute;
    invalidateCache: () => void;
    wrapSqlWriteError: (error: unknown, mode: 'update') => never;
    createRepository: (meta: ModelMetadata) => RepositoryDeleteChild;
  };

type RepositoryUpdateWriteDepsParams = RepositoryMutationWriteFacadeDepsParams<'write'> &
  RepositoryUpdateMutationPayloadDepsParams &
  RepositoryGetScalarFieldsDepsLike<ModelMetadata> &
  RepositorySoftConditionPipelineDepsLike<BaseQueryCondition> & {
    db: RepositoryDbLike;
    makeSelectCtx: RepositorySelectCtxFactory;
    aliasSelection: RepositorySelectionAliaser;
    execute: RepositoryExecute;
    decodeFromDb: (row: Entity) => Entity;
    invalidateCache: () => void;
    recomputePersistForUpdate?: (payload: { targetIds: string[]; sanitized: Entity; condition: BaseQueryCondition; rows: unknown[] }) => Promise<void>;
  };

export function createRepositoryCreateMutationPayloadDeps(params: RepositoryCreateMutationPayloadDepsParams) {
  return {
    assertFieldRuleWriteAllowed: params.assertFieldRuleWriteAllowed,
    applyDefaultCompanyIdOnCreate: params.applyDefaultCompanyIdOnCreate,
    validateFields: params.validateFields,
    encodeForDb: params.encodeForDb,
  };
}

export function createRepositoryUpdateMutationPayloadDeps(params: RepositoryUpdateMutationPayloadDepsParams) {
  return {
    assertFieldRuleWriteAllowed: params.assertFieldRuleWriteAllowed,
    applyDefaultCompanyIdOnUpdate: params.applyDefaultCompanyIdOnUpdate,
    validateFields: params.validateFields,
    encodeForDb: params.encodeForDb,
  };
}

export function createRepositoryMutationWriteTargetDeps<TOp extends 'delete' | 'write'>(params: RepositoryMutationWriteTargetDepsParams<TOp>) {
  return {
    meta: params.meta,
    locateIdsForCondition: params.locateIdsForCondition,
    assertCompanyWriteAccessForCondition: params.assertCompanyWriteAccessForCondition,
    assertRecordRuleAllTargetsAllowed: params.assertRecordRuleAllTargetsAllowed,
  };
}

export function createRepositoryMutationWriteConditionDeps<TOp extends 'delete' | 'write'>(params: RepositoryMutationWriteConditionDepsParams<TOp>) {
  return {
    table: params.table,
    applyRecordRuleToCondition: params.applyRecordRuleToCondition,
    applyDefaultLayers: params.applyDefaultLayers,
    isEmptyCondition: params.isEmptyCondition,
    convertCondition: params.convertCondition,
  };
}

export function createRepositoryMutationWriteFacadeDeps<TOp extends 'delete' | 'write'>(params: RepositoryMutationWriteFacadeDepsParams<TOp>) {
  return {
    ...createRepositoryMutationWriteTargetDeps(params),
    ...createRepositoryMutationWriteConditionDeps(params),
  };
}

export function createRepositoryCreateFacadeDeps(params: RepositoryCreateFacadeDepsParams) {
  return createRepositoryCreateWriteDeps({
    ...params,
    ...createRepositoryCreateMutationPayloadDeps(params),
  });
}

export function createRepositoryUpdateFacadeDeps(params: RepositoryUpdateFacadeDepsParams) {
  return createRepositoryUpdateWriteDeps({
    ...params,
    ...createRepositoryMutationWriteFacadeDeps(params),
    ...createRepositoryUpdateMutationPayloadDeps(params),
  });
}

export function createRepositoryDeleteWriteDeps(params: RepositoryDeleteWriteDepsParams) {
  return {
    db: params.db,
    table: params.table,
    meta: params.meta,
    softField: params.softField,
    locateIdsForCondition: params.locateIdsForCondition,
    assertCompanyWriteAccessForCondition: params.assertCompanyWriteAccessForCondition,
    assertRecordRuleAllTargetsAllowed: params.assertRecordRuleAllTargetsAllowed,
    softDeleteEnabled: params.softDeleteEnabled,
    applySoftLayer: params.applySoftLayer,
    applyRecordRuleToCondition: params.applyRecordRuleToCondition,
    applyDefaultLayers: params.applyDefaultLayers,
    isEmptyCondition: params.isEmptyCondition,
    convertCondition: params.convertCondition,
    execute: params.execute,
    invalidateCache: params.invalidateCache,
    wrapSqlWriteError: params.wrapSqlWriteError,
    createRepository: params.createRepository,
  };
}

export function createRepositoryCreateWriteDeps(params: RepositoryCreateWriteDepsParams) {
  return {
    db: params.db,
    table: params.table,
    meta: params.meta,
    getRecordRuleEnvelope: params.getRecordRuleEnvelope,
    permissionDenied: params.permissionDenied,
    assertFieldRuleWriteAllowed: params.assertFieldRuleWriteAllowed,
    generateId: params.generateId,
    applyDefaultCompanyIdOnCreate: params.applyDefaultCompanyIdOnCreate,
    validateFields: params.validateFields,
    encodeForDb: params.encodeForDb,
    execute: params.execute,
    wrapSqlWriteError: params.wrapSqlWriteError,
    assertRecordRuleAllCreatedAllowed: params.assertRecordRuleAllCreatedAllowed,
    recomputePersistForCreate: params.recomputePersistForCreate,
  };
}

export function createRepositoryUpdateWriteDeps(params: RepositoryUpdateWriteDepsParams) {
  return {
    db: params.db,
    table: params.table,
    meta: params.meta,
    getScalarFields: params.getScalarFields,
    makeSelectCtx: params.makeSelectCtx,
    aliasSelection: params.aliasSelection,
    applySoftLayer: params.applySoftLayer,
    isEmptyCondition: params.isEmptyCondition,
    convertCondition: params.convertCondition,
    execute: params.execute,
    decodeFromDb: params.decodeFromDb,
    assertCompanyWriteAccessForCondition: params.assertCompanyWriteAccessForCondition,
    locateIdsForCondition: params.locateIdsForCondition,
    assertRecordRuleAllTargetsAllowed: params.assertRecordRuleAllTargetsAllowed,
    assertFieldRuleWriteAllowed: params.assertFieldRuleWriteAllowed,
    applyDefaultCompanyIdOnUpdate: params.applyDefaultCompanyIdOnUpdate,
    validateFields: params.validateFields,
    encodeForDb: params.encodeForDb,
    applyRecordRuleToCondition: params.applyRecordRuleToCondition,
    applyDefaultLayers: params.applyDefaultLayers,
    invalidateCache: params.invalidateCache,
    recomputePersistForUpdate: params.recomputePersistForUpdate,
  };
}
